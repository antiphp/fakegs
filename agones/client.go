// Package agones handles Agones communication.
package agones

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"agones.dev/agones/pkg/sdk"
	"github.com/cenkalti/backoff/v4"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// State represents an Agones state.
type State string

const (
	// StateReady is the Agones state Ready.
	// The state indicates that the game server is ready to receive traffic.
	StateReady State = "Ready"

	// StateAllocated is the Agones state Allocated.
	// The state indicates that the game server hosts a game session.
	StateAllocated State = "Allocated"

	// StateShutdown is the Agones state Shutdown.
	// The state indicates that the game server is shutting down.
	StateShutdown State = "Shutdown"
)

// Client is the Agones client.
type Client struct {
	sdk sdk.SDKClient

	once   sync.Once
	health sdk.SDK_HealthClient

	mu       sync.Mutex
	listener []chan<- *sdk.GameServer

	wg      sync.WaitGroup
	watchCh chan *sdk.GameServer

	doneCh chan struct{}
}

// NewSDKClient returns a new Agones SDK client.
func NewSDKClient(addr string) (sdk.SDKClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithChainUnaryInterceptor(
			timeout.UnaryClientInterceptor(10*time.Second),
		),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dialing Agones %s: %w", addr, err)
	}

	return sdk.NewSDKClient(conn), nil
}

// NewClient returns a new agones client.
func NewClient(sdkClient sdk.SDKClient) *Client {
	return &Client{
		sdk:     sdkClient,
		watchCh: make(chan *sdk.GameServer),
		doneCh:  make(chan struct{}),
	}
}

// Close closes active connections.
func (a *Client) Close() {
	close(a.doneCh)
	a.wg.Wait()
	close(a.watchCh)
}

// ReportHealth reports the health of the game server.
func (a *Client) ReportHealth(ctx context.Context) error {
	var err error
	a.once.Do(func() {
		a.health, err = a.sdk.Health(ctx)
	})
	if err != nil {
		return fmt.Errorf("creating health client: %w", err)
	}

	if err = a.health.Send(&sdk.Empty{}); err != nil {
		return fmt.Errorf("sending health report: %w", err)
	}
	return nil
}

// UpdateState updates the state.
func (a *Client) UpdateState(ctx context.Context, st State) error {
	switch st {
	case StateReady:
		_, err := a.sdk.Ready(ctx, &sdk.Empty{})
		if err != nil {
			return fmt.Errorf("updating state to ready: %w", err)
		}
	case StateAllocated:
		_, err := a.sdk.Allocate(ctx, &sdk.Empty{})
		if err != nil {
			return fmt.Errorf("updating state to allocated: %w", err)
		}
	case StateShutdown:
		_, err := a.sdk.Shutdown(ctx, &sdk.Empty{})
		if err != nil {
			return fmt.Errorf("updating state to shutdown: %w", err)
		}
	default:
		return errors.New("unknown state: " + string(st))
	}
	return nil
}

// Run runs the state watcher and manages the distribution of the updates.
func (a *Client) Run(ctx context.Context) {
	a.wg.Add(1)
	defer a.wg.Done()

	ctx, cancel := context.WithCancel(ctx) // For doneCh.
	defer cancel()

	go a.watchGameServer(ctx)

	for {
		var gs *sdk.GameServer
		select {
		case <-a.doneCh:
			return
		case <-ctx.Done():
			return
		case gs = <-a.watchCh:
		}

		func() {
			a.mu.Lock()
			defer a.mu.Unlock()

			for _, c := range a.listener {
				select {
				case <-ctx.Done():
				case c <- gs:
				}
			}
		}()
	}
}

// WatchStateChanged returns a channel to watch for state changes.
func (a *Client) WatchStateChanged(ctx context.Context) <-chan State {
	watchCh := make(chan *sdk.GameServer)
	go func() {
		defer close(watchCh)
		<-ctx.Done()
	}()

	a.mu.Lock()
	a.listener = append(a.listener, watchCh)
	a.mu.Unlock()

	ch := make(chan State)
	go func() {
		defer close(ch)

		var gs *sdk.GameServer
		var prevState State
		for {
			select {
			case <-ctx.Done():
				return
			case gs = <-watchCh:
			}

			state := State(gs.GetStatus().GetState())
			if state == prevState {
				continue
			}

			if prevState != "" { // Skip first change, usually not interesting.
				select {
				case <-ctx.Done():
				case ch <- state:
				}
			}

			prevState = state
		}
	}()
	return ch
}

func (a *Client) watchGameServer(ctx context.Context) {
	a.wg.Add(1)
	defer a.wg.Done()

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // Retry forever.

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(bo.NextBackOff()):
		}

		conn, err := a.sdk.WatchGameServer(ctx, &sdk.Empty{})
		if err != nil {
			continue
		}

		bo.Reset()

		for {
			gs, err := conn.Recv()
			if err != nil {
				break
			}

			select {
			case <-ctx.Done():
			case a.watchCh <- gs:
			}
		}
	}
}

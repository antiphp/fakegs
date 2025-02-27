package fakegs

import (
	"context"
	"sync"
	"time"
)

// ExitAfter is an exit handler.
//
// The exit handler emits an exit message after a given duration.
type ExitAfter struct {
	dur time.Duration

	wg     sync.WaitGroup
	doneCh chan struct{}
}

// NewExitAfter returns a new handler.
func NewExitAfter(dur time.Duration) *ExitAfter {
	return &ExitAfter{
		dur:    dur,
		doneCh: make(chan struct{}),
	}
}

// Start starts the exit timer.
func (e *ExitAfter) Start(ctx context.Context, ch chan<- Message) error {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()

		t := time.NewTimer(e.dur)
		defer t.Stop()

		select {
		case <-ctx.Done():
		case <-e.doneCh:
		case <-t.C:
			select {
			case <-ctx.Done():
			case <-e.doneCh:
			case ch <- Message{
				Type:        MessageTypeExit,
				Description: "exit timer elapsed after " + e.dur.String(),
			}:
			}
		}
	}()

	return nil
}

// Stop stops the exit timer.
func (e *ExitAfter) Stop() {
	close(e.doneCh)
	e.wg.Wait()
}

// Handle handles emitted messages.
func (e *ExitAfter) Handle(context.Context, Message) error {
	return nil
}

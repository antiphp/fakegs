// Package fakegs contains the fake game server runtime code.
package fakegs

import (
	"context"
	"fmt"

	"github.com/hamba/logger/v2"
)

// MessageType indicates the type of message.
type MessageType int

const (
	// MessageTypeExit indicates that the server should exit.
	MessageTypeExit MessageType = iota
)

// Message is a message sent to the server.
type Message struct {
	// Type is the type of the message.
	Type MessageType
	// Description is the description of the message.
	Description string
	// Data is the data payload of the message.
	Data any
}

// Handler is a message handler.
//
// A handler can emit messages.
// A handler receives any emitted message.
type Handler interface {
	Start(context.Context, chan<- Message) error
	Stop()
	Handle(context.Context, Message) error
}

// Server is the fake game server.
type Server struct {
	hdlrs Handlers
	log   *logger.Logger
}

// NewServer returns a new fake game server.
func NewServer(log *logger.Logger) *Server {
	return &Server{
		hdlrs: Handlers{},
		log:   log,
	}
}

// Add adds a handler.
func (s *Server) Add(hdlr Handler) {
	s.hdlrs = append(s.hdlrs, hdlr)
}

// Run runs the runtime routine for the fake game server and returns a stop reason or error.
func (s *Server) Run(ctx context.Context) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan Message)
	defer close(ch)

	if err := s.hdlrs.Start(ctx, ch); err != nil {
		return "", fmt.Errorf("starting handlers: %w", err)
	}
	defer s.hdlrs.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err().Error(), nil
		case msg := <-ch:
			if err := s.hdlrs.Handle(ctx, msg); err != nil {
				return "", err
			}

			if msg.Type == MessageTypeExit {
				return msg.Description, nil
			}
		}
	}
}

// Handlers is a collection of handlers.
type Handlers []Handler

// Start starts the handlers.
func (h Handlers) Start(ctx context.Context, ch chan<- Message) error {
	for _, handler := range h {
		if err := handler.Start(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops the handlers.
func (h Handlers) Stop() {
	for i := len(h) - 1; i >= 0; i-- {
		h[i].Stop()
	}
}

// Handle handles messages.
func (h Handlers) Handle(ctx context.Context, msg Message) error {
	for _, handler := range h {
		if err := handler.Handle(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

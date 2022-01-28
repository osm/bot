// Light weight event library
package event

import (
	"fmt"
	"reflect"
	"sync"
)

// Handler is a convenience type that represents an event handler function
type Handler interface{}

// Payload is a convenience type that represents a payload that a handler function receives
type Payload interface{}

// Hub defines the methods that are needed by an event hub
type Hub interface {
	// Adds a handler for an event
	Handle(e string, fn Handler) (err error)

	// Send sends a payload for an event
	Send(e string, p Payload) (err error)
}

// hub implements the Hub interface
type hub struct {
	// handlers contains a map of all events and their handlers
	handlers map[string][]Handler

	// mu prevents race conditions for the handlers map
	mu sync.Mutex
}

// NewHub creates a new event hub
func NewHub() Hub {
	return &hub{
		handlers: make(map[string][]Handler),
	}
}

// Handle adds a new handler for an event
func (h *hub) Handle(e string, fn Handler) error {
	// Inspect the handler
	t := reflect.TypeOf(fn)

	// Make sure that the handler is a function
	if t.Kind() != reflect.Func {
		return fmt.Errorf("handler for event \"%s\" is not a function", e)
	}

	// Make sure the function only has one parameter
	if t.NumIn() != 1 {
		return fmt.Errorf("handler for event \"%s\" does not have one parameter", e)
	}

	// Acquire a lock before appending to the handlers slice
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add the function to the event handlers map
	h.handlers[e] = append(h.handlers[e], fn)

	return nil
}

// Send sends data to an event
func (h *hub) Send(e string, p Payload) error {
	// Acquire a lock before start processing the event
	h.mu.Lock()
	defer h.mu.Unlock()

	// Make sure that we have event handlers for the event
	if _, ok := h.handlers[e]; !ok {
		return fmt.Errorf("no event handler added for event \"%s\"", e)
	}

	// Get the payload value
	pv := []reflect.Value{reflect.ValueOf(p)}

	// Get the type of the event payload
	pvType := reflect.TypeOf(p)

	// Iterate over all handlers for the event
	for _, cb := range h.handlers[e] {
		// Get the type of the function parameter
		// We only accept handlers with one parameter, so this is safe to do
		fnType := reflect.TypeOf(cb).In(0)

		// Let's just continue if the function parameter type doesn't match the event payload type
		if fnType != pvType {
			continue
		}

		// Create a callable function of the callback
		fn := reflect.ValueOf(cb)

		// Execute the callback in a separate go routine
		go func() {
			fn.Call(pv)
		}()
	}

	// Return
	return nil
}

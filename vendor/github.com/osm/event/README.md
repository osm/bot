## About

Lightweight and easy to use event library for go applications.

## Handlers

The handler function must only have one function parameter in its definition, the type can be whatever you like. This means that you can have multiple event handlers attached to the same event and that the event payload can be of different data types.

Example:

```go
// Three event handlers for the same event
// Two that handles strings and one that handle integers
h.Handle("foo", func(s string) { fmt.Println("handler of strings 1", s) })
h.Handle("foo", func(s string) { fmt.Println("handler of strings 2", s) })
h.Handle("foo", func(i int) { fmt.Println("handler of integers", i) })

// This will be handled by the two first event handlers
h.Send("foo", "foo bar baz")

// This will only trigger the integer event handler
h.Send("foo", 12345)
```

## Complete example

```go
package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/osm/event"
)

func main() {
	// Keep track of how many times we have called the event handler
	count := 0

	// Initialize channel for handling SIGINT
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	// Initialize a new event handler hub
	h := event.NewHub()

	// Add a handler for the foo event
	err := h.Handle("foo", func(s string) {
		// Print to stdout
		fmt.Printf("event handler called %d times with the payload %s\n", count, s)

		// Increase execution count
		count = count + 1

		// Stop after five executions
		if count == 5 {
			close(c)
		}
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	// Print some helpful information
	fmt.Println("waiting for SIGINT")

	// Send a new event for each SIGINT
	for range c {
		h.Send("foo", "foo bar baz")
	}
}
```

## Docs

```sh
godoc github.com/osm/event
```

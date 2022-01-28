## About

Event driven IRC library for go

## Features

* Easy to use
* Async event handlers
* Nick reclaim
* Reconnect on disconnect

## Handlers

An event handler receives a pointer to the message that was sent to the client from the server.
See `godoc github.com/osm/irc Message` for a complete description of the Message struct.

Example:

```go
c.Handle("PRIVMSG", func(m *Message) {
	fmt.Println(m.Raw)
})
```

## Complete example

```go
package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/osm/irc"
)

func main() {
	// Create a new IRC client
	c := irc.NewClient(
		irc.WithAddr("localhost:6667"),
		irc.WithNick("foo"),
		irc.WithUser("bar"),
		irc.WithRealName("foo bar"),
		irc.WithChannel("#dev"),
		irc.WithDebug(),
		irc.WithLogger(log.New(os.Stdout, "IRC: ", log.LstdFlags)))

	// Setup an event handler for PRIVMSG
	c.Handle("PRIVMSG", func(m *irc.Message) {
		if m.Params == "#dev :hello" {
			c.Privmsg(m.ParamsArray[0], "world")
		}
	})

	// Create a wait group
	var wg sync.WaitGroup
	wg.Add(1)

	// Connect is blocking, so we need to run it in a goroutine
	go func() {
		if err := c.Connect(); err != nil {
			fmt.Println(err)
		}

		wg.Done()
	}()

	// Wait for the client to end
	wg.Wait()
}
```

## Docs

```sh
godoc github.com/osm/irc
```

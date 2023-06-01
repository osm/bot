package irc

import (
	"log"
	"net"
	"os"
	"sync"

	"github.com/osm/event"
)

type postConnectMessage struct {
	target  string
	message string
}

// Client contains the IRC client
type Client struct {
	// Connection and address
	conn net.Conn
	addr string

	// Event hub
	hub event.Hub

	// Logger
	logger *log.Logger

	// Quit channel
	// Send data on this channel to exit the main loop
	quit chan bool

	// Client related variables
	nick                string
	user                string
	realName            string
	channels            []string
	version             string
	currentNick         string
	currentUser         string
	currentHost         string
	postConnectMessages []postConnectMessage
	infoMu              sync.Mutex

	// If this is true, all output will be logged
	debug bool
}

// NewClient creates a new IRC client
func NewClient(opts ...Option) *Client {
	// Create a new client
	c := &Client{
		hub:     event.NewHub(),
		logger:  log.New(os.Stdout, "IRC: ", log.LstdFlags),
		quit:    make(chan bool),
		version: "github.com/osm/irc",
	}

	// Apply all options
	for _, opt := range opts {
		opt(c)
	}

	// Attach all core event handlers
	c.coreEvents()

	// Return the client
	return c
}

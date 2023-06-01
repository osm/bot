package irc

import (
	"log"
	"net"
)

// Option should be implemented by all client options
type Option func(*Client)

// WithAddr sets the address of the IRC server, this can be omitted if you supply a connection with WithConn
func WithAddr(addr string) Option {
	return func(c *Client) {
		c.addr = addr
	}
}

// WithChannel sets the channel that the client should join on connect, this can be called mupltiple times
func WithChannel(ch string) Option {
	return func(c *Client) {
		if ch != "" {
			c.channels = append(c.channels, ch)
		}
	}
}

// WithConn sets the client connection, this can be omitted if you supply an address with WithAddr
func WithConn(conn net.Conn) Option {
	return func(c *Client) {
		c.conn = conn
		c.addr = conn.RemoteAddr().String()
	}
}

// WithDebug sets the debug flag, set this if you want to log the communication
func WithDebug() Option {
	return func(c *Client) { c.debug = true }
}

// WithLogger sets the logger
func WithLogger(logger *log.Logger) Option {
	return func(c *Client) { c.logger = logger }
}

// WithNick sets the nick for the client
func WithNick(n string) Option {
	return func(c *Client) { c.nick = n }
}

// WithRealName sets the real name for the client
func WithRealName(r string) Option {
	return func(c *Client) { c.realName = r }
}

// WithUser sets the user for the client
func WithUser(u string) Option {
	return func(c *Client) { c.user = u }
}

// WithVersion sets the CTCP VERSION reply string
func WithVersion(v string) Option {
	return func(c *Client) { c.version = v }
}

func WithPostConnectMessage(t, m string) Option {
	return func(c *Client) { c.postConnectMessages = append(c.postConnectMessages, postConnectMessage{t, m}) }
}

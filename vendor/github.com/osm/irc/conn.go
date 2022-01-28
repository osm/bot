package irc

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"time"
	"unicode/utf8"
)

// Connect connects to the IRC server
func (c *Client) Connect() error {
	var err error

	// Make sure we have either a connection or an address set
	if c.conn == nil && c.addr == "" {
		return fmt.Errorf("no conn or addr found, use WithConn or WithAddr")
	}

	// Check if we have set a nick
	if c.nick == "" {
		return fmt.Errorf("no nick set, use WithNick to set the nick")
	}

	// Set current nick to nick
	// This is used so we can get our wanted nick back if it is taken during the connect
	c.currentNick = c.nick

	// Set user to nick if it isn't set
	if c.user == "" {
		c.user = c.nick
	}

	// Set real name to nick if it isn't set
	if c.realName == "" {
		c.realName = c.nick
	}

	// Dial the server, if we don't have a connection already
	if c.conn == nil {
		if c.conn, err = net.Dial("tcp", c.addr); err != nil {
			return err
		}
	}

	// Send the USER command
	if err = c.Sendf("USER %s * * :%s", c.user, c.realName); err != nil {
		return err
	}

	// Send the NICK command
	if err = c.Nick(c.currentNick); err != nil {
		return err
	}

	// Start main loop and return the value
	return c.loop()
}

// reconnect tries to reconnect to the server
func (c *Client) reconnect() error {
	// Close the connection
	c.conn.Close()
	c.conn = nil

	// Reconnect time
	rt := 5 * time.Second

	// Try to reconnect 10 times before giving up
	for i := 0; i < 10; i++ {
		// Retry after rt seconds has passed
		c.log("connection closed, trying to reconnect in %d seconds", rt/time.Second)
		time.Sleep(rt)

		// Connect to the server
		err := c.Connect()

		// If no error we assume that the connect was successful
		if err == nil {
			return nil
		}

		// Log the error
		c.log(err.Error())

		// Increase the retry time for each attempt
		rt *= 2
	}

	return fmt.Errorf("unable to reconnect, giving up")
}

// fixEncoding checks whether or not the given buf is utf-8 encoded, if it
// isn't we'll assume it is encoded using ISO8859-1, in which case we'll
// encode it to use UTF-8 instead.
func fixEncoding(buf []byte) string {
	if utf8.Valid(buf) {
		return string(buf)
	}

	ret := make([]rune, len(buf))
	for i, b := range buf {
		ret[i] = rune(b)
	}
	return string(ret)
}

// loop is responsible for reading and parsing messages from the server
func (c *Client) loop() error {
	// Initialize connection reader
	rd := bufio.NewReader(c.conn)
	tr := textproto.NewReader(rd)

	// Main loop
	for {
		select {
		case <-c.quit:
			// Quit ends the connection to the IRC server
			goto quit

		default:
			// Read one line from the connection
			b, err := tr.ReadLineBytes()
			l := fixEncoding(b)

			// Print the line if we have debugging enabled
			c.log(l)

			// EOF received, try to reconnect
			if err == io.EOF {
				goto reconnect
			}

			// Other errors are just returned
			if err != nil {
				return err
			}

			// Parse the message
			// If we fail to parse the message we log it and continue in the loop
			m, err := parse(l)
			if err != nil {
				c.log(err.Error())
				continue
			}

			// If we are joinning a channel we'll store the
			// current user and current host in the client, this
			// will be used to calculate the correct number of
			// bytes that we are allowed to send to the server.
			if m.Command == "JOIN" && m.Name == c.currentNick {
				c.infoMu.Lock()
				c.currentUser = m.User
				c.currentHost = m.Host
				c.infoMu.Unlock()
			}

			// Send the message to the event hub
			// We use the command as event name
			c.hub.Send(m.Command, m)

			// Let's also send the message to the wildcard event
			c.hub.Send("*", m)
		}
	}

reconnect:
	// Try to reconnect to the server
	return c.reconnect()

quit:
	// Quit closes the connection and returns from the function
	c.conn.Close()
	return nil
}

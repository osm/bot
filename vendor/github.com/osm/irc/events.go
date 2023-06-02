package irc

import (
	"fmt"
	"time"
)

// Handle registers a new event handler
func (c *Client) Handle(event string, fn func(m *Message)) {
	c.hub.Handle(event, fn)
}

// coreEvents setups event handlers for the most common tasks that everyone most likely wants
func (c *Client) coreEvents() {
	// Handle PING PONG
	// We also try to reclaim our nick on each PING from the server
	c.Handle("PING", func(m *Message) {
		// Send PONG
		c.Sendf("PONG %s", m.Params)

		// Try to reclaim our nick on each PING
		c.ReclaimNick()
	})

	// If the nick that PARTs is our configured nick we'll reclaim it.
	c.Handle("QUIT", func(m *Message) {
		if m.Name == c.nick {
			// Acquire lock
			c.infoMu.Lock()

			// Send NICK command
			c.Nick(c.nick)

			// Set current nick to what we just changed it to be
			c.currentNick = c.nick

			// Release the lock
			c.infoMu.Unlock()
		}
	})

	// 401 is returned by the server after a WHOIS request if the nick is not in use
	// Let's verify if the WHOIS request was made from a nick reclaim attempt
	c.Handle("401", func(m *Message) {
		// Our current nick is not the nick that we want
		// Let's acquire a lock and change it
		if m.Params == fmt.Sprintf("%s %s :No such nick or channel name", c.currentNick, c.nick) ||
			m.Params == fmt.Sprintf("%s %s :No such nick", c.currentNick, c.nick) {
			// Acquire lock
			c.infoMu.Lock()

			// Send NICK command
			c.Nick(c.nick)

			// Set current nick to what we just changed it to be
			c.currentNick = c.nick

			// Release the lock
			c.infoMu.Unlock()
		}
	})

	// Things to do after a successful connect
	c.Handle("001", func(m *Message) {
		// The post connect messages and modes should occur before
		// joining any channels.
		for _, pcm := range c.postConnectMessages {
			c.Privmsg(pcm.target, pcm.message)
		}
		for _, m := range c.postConnectModes {
			c.Sendf("MODE %s %s", c.currentNick, m)
		}

		// To make sure all the messages and modes has been
		// successfully applied before we join a channel we'll sleep
		// for a short while.
		time.Sleep(3 * time.Second)

		for _, ch := range c.channels {
			c.Sendf("JOIN %s", ch)
		}
	})

	// Handle CTCP version requests
	c.Handle("PRIVMSG", func(m *Message) {
		// Make sure that the CTCP VERSION request is made to our current nick
		if m.Params == fmt.Sprintf("%s :\x01VERSION\x01", c.currentNick) {
			// Reply
			c.Noticef(m.Name, "\x01VERSION %s\x01", c.version)
		}
	})

	// Handle nick in use
	c.Handle("433", func(m *Message) {
		// Acquire lock
		c.infoMu.Lock()

		// Update the nick
		c.currentNick = fmt.Sprintf("%s_", c.currentNick)

		// Send nick to server
		c.Nick(c.currentNick)

		// Release the lock
		c.infoMu.Unlock()
	})
}

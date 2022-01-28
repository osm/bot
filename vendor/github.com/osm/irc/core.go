package irc

import (
	"fmt"
	"time"

	"github.com/osm/ww"
)

// log logs the message with the logger
func (c *Client) log(format string, args ...interface{}) {
	if c.debug && format != "" {
		c.logger.Printf(format, args...)
	}
}

// Sendf sends a message to the server and appends CR-LF at the end of the string
func (c *Client) Sendf(format string, args ...interface{}) error {
	// Make sure that conn isn't nil before we proceed.
	if c.conn == nil {
		return nil
	}

	// Format the string
	s := fmt.Sprintf(format+eol, args...)

	// An IRC message has a limit of maximum 510 characters, so we'll just
	// truncate the rest of the message if it's too big.
	// We are calling the ww.Wrap function before the data gets here, but
	// it is a possibility that a really long word (510 characters) gets
	// to this point, and if it does we'll truncate the message.
	if len(s) > 510 {
		s = s[0:510] + eol
	}

	// Log message if we have debugging enabled
	c.log(s)

	// Write it to server and return
	_, err := c.conn.Write([]byte(s))
	return err
}

// Privmsg sends a message to a channel or nick
func (c *Client) Privmsg(target, message string) error {
	prefix := fmt.Sprintf(": %s!%s@%s", c.currentNick, c.currentUser, c.currentHost)
	cmd := fmt.Sprintf("PRIVMSG %s :", target)

	for i, m := range ww.Wrap(message, 510-len(prefix)-len(cmd)) {
		if err := c.Sendf("%s%s", cmd, m); err != nil {
			return err
		}

		if i >= 1 {
			time.Sleep(time.Millisecond * 500)
		}
	}

	return nil
}

// Privmsgf sends a privmsg and accepts a format string as message argument
func (c *Client) Privmsgf(target, format string, args ...interface{}) error {
	return c.Privmsg(target, fmt.Sprintf(format, args...))
}

// Notice sends a notice
func (c *Client) Notice(target, message string) error {
	prefix := fmt.Sprintf(": %s!%s@%s", c.currentNick, c.currentUser, c.currentHost)
	cmd := fmt.Sprintf("NOTICE %s :", target)

	for i, m := range ww.Wrap(message, 510-len(prefix)-len(cmd)) {
		if err := c.Sendf("%s%s", cmd, m); err != nil {
			return err
		}

		if i >= 1 {
			time.Sleep(time.Millisecond * 500)
		}
	}

	return nil
}

// Noticef sends a notice and accepts a format string as message argument
func (c *Client) Noticef(target, format string, args ...interface{}) error {
	return c.Notice(target, fmt.Sprintf(format, args...))
}

// Mode sets mode on a channel for a nick
func (c *Client) Mode(channel, mode, target string) error {
	return c.Sendf("MODE %s %s %s", channel, mode, target)
}

// Nick sets the nick
func (c *Client) Nick(nick string) error {
	return c.Sendf("NICK %s", nick)
}

// GetNick returns the current nick
func (c *Client) GetNick() string {
	return c.currentNick
}

// ReclaimNick tries to reclaim the nick
func (c *Client) ReclaimNick() {
	// Acquire a lock to prevent race condition
	c.infoMu.Lock()

	// Check if we actually don't have the wanted nick
	if c.nick != c.currentNick {
		// Perform a WHOIS request
		// We check for event 401 in events.go and tries to reclaim the nick if it's free
		c.Whois(c.nick)
	}

	// Release the lock
	c.infoMu.Unlock()
}

// Whois sends a WHOIS request
func (c *Client) Whois(nick string) error {
	return c.Sendf("WHOIS %s", nick)
}

// Quit sends a QUIT message to the server and terminates the connection
func (c *Client) Quit(message string) {
	c.Sendf("QUIT :%s", message)
	c.quit <- true
}

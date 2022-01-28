package irc

import (
	"fmt"
	"strings"
)

// Message represents the RFC1459 definition of an IRC message
// See the full definition in section 2.3.1 in the RFC
type Message struct {
	// Raw contains the unparsed message
	Raw string

	// Command contains a three digit number or a string
	Command string

	// Params is filled with all parameters that the message contains
	Params string

	// ParamsArray is equal to Params, but are splitted on space for easier manipulation
	ParamsArray []string

	// Name is an optional field, if it contains data it holds either the server name or a nick
	Name string

	// User is an optional parameter that contains the user if the message originates from a client
	User string

	// Host is also an optional parameter that contains the host if the message originates from a client
	Host string
}

// Constants to improve code readability
const (
	prefix     string = ":"
	userPrefix string = "!"
	hostPrefix string = "@"
	eol        string = "\r\n"
	maxSize    int64  = 512
)

// parse takes an IRC message and parses it into the Message format
func parse(m string) (*Message, error) {
	// r contains a pointer to the Message that we parse the data into
	r := &Message{Raw: m}

	// Messages are separated by space (0x20)
	p := strings.Fields(m)

	// Empty lines are OK, just return an empty message
	if strings.Compare(m, eol) == 0 {
		return nil, nil
	}

	// The message must contain at least two parts
	if len(p) < 2 || int64(len(m)) > maxSize {
		return nil, fmt.Errorf("malformed message '%s'", m)
	}

	// Check if the message is prefixed, if so, parse the prefix
	if strings.Index(p[0], prefix) == 0 {
		// Extract index of user and host prefixes
		ui := strings.Index(p[0], userPrefix)
		hi := strings.Index(p[0], hostPrefix)

		// First case means that there is an extended prefix
		// Otherwise it's just a server message without any additional client information
		if ui != hi {
			r.Name = p[0][1:ui]
			r.User = p[0][ui+1 : hi]
			r.Host = p[0][hi+1:]
		} else {
			r.Name = p[0][1:]
		}

		// We are done with this data, so let's discard it to make parsing easier
		p = p[1:]
	}

	// Next part of the data contains the command
	// The command can be either a three digit number of a string
	r.Command = p[0]

	// The remaining data is the command parameters
	r.Params = strings.Join(p[1:], " ")
	r.ParamsArray = p[1:]

	// Return the message
	return r, nil
}

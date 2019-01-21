package main

import (
	"strings"

	"github.com/osm/irc"
)

// privmsgAction is a structure that IRC PRIVMSG messages can be parsed into
// so that command handling becomes easier.
type privmsgAction struct {
	args         []string
	cmd          string
	host         string
	msg          string
	nick         string
	target       string
	validChannel bool
}

// joinAction is a structure that IRC JOIN messages can be parsed into so that
// command handling becomes easier.
type joinAction struct {
	channel      string
	host         string
	nick         string
	validChannel bool
}

// parseAction takes an IRC message and parses it into an action struct.  The
// action struct is meant to make code easier to understand, since all the IRC
// protocol separators etc will be stripped out.
//
// target will contain the message target, this can either be a channel or the
// current name of the bot.  The first word in a message will be parsed into
// the cmd property.  Remaining words will be placed within the args property.
func (b *bot) parseAction(m *irc.Message) interface{} {
	switch m.Command {
	case "PRIVMSG":
		// A PRIVMSG should look similar to this:
		// :the_nick!~bar@172.17.0.1 PRIVMSG #channel :foo bar baz
		return &privmsgAction{
			// args will be set to [bar, baz] if the example is
			// used.
			args: m.ParamsArray[2:],

			// cmd will be set to foo, note that the initial ':'
			// is removed.
			cmd: m.ParamsArray[1][1:],

			// host will be set to ~bar@172.17.0.1
			host: parseHost(m),

			// msg will be foo bar baz, returned as a string.
			msg: strings.Join(m.ParamsArray[1:], " ")[1:],

			// nick will contain the_nick
			nick: m.Name,

			// target can be either a channel or a nick.
			target: m.ParamsArray[0],

			// validChannel will be true if channel is equal to
			// the channel in the config.
			validChannel: m.ParamsArray[0] == b.IRC.Channel,
		}
	case "JOIN":
		// A JOIN message should look similar to this:
		// :the_nick!~bar@172.17.0.1 JOIN :#foo
		ch := m.Raw[strings.LastIndex(m.Raw, "#"):]

		return &joinAction{
			// channel will be set to #foo, the initial ':' will
			// be stripped.
			channel: ch,

			// host will be set to ~bar@172.17.0.1
			host: parseHost(m),

			// nick will contain the_nick
			nick: m.Name,

			// validChannel will be true if channel is equal to
			// the channel in the config.
			validChannel: ch == b.IRC.Channel,
		}
	default:
		return nil
	}
}

// parseHost returns the host of the message that is passed.  An example
// message might look like this:
// :the_nick!~bar@172.17.0.1 PRIVMSG #channel :foo bar baz
// From this message we will return ~bar@172.17.0.1
func parseHost(m *irc.Message) string {
	start := strings.Index(m.Raw, "!")
	end := strings.Index(m.Raw, " ")
	return m.Raw[start+1 : end]
}

package main

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/osm/irc"
)

// command holds the information needed to process a command.
type command struct {
	acceptArgs bool
	bin        string
	args       []string
}

// parseCommand returns a command based om the given input data.
func parseCommand(acceptArgs bool, data string) command {
	parts := strings.Split(data, " ")

	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	return command{
		acceptArgs: acceptArgs,
		bin:        parts[0],
		args:       args,
	}
}

// initFactoidDefaults sets default values for all settings.
func (b *bot) initCommandDefaults() {
	if b.IRC.CommandErrExec == "" {
		b.IRC.CommandErrExec = "command execution error"
	}
}

// commandArgumentRegexp is a regexp that makes sure that we don't get
// unwanted characters passed to the external commands that are to be
// executed.
var commandArgumentRegexp = regexp.MustCompile("^[a-zA-Z0-9åäöÅÄÖ0-9+*-\\/?= ]*$")

// commandHandler exposes external commands that are defined in the Commands
// secion of the configuration file. If the IRC message matches a key of the
// defined command it executes it and returns the output to the user. If the
// message was seen in a channel it will return it back to the channel,
// otherwise it will be sent back to the user.
func (b *bot) commandHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	c, ok := b.IRC.commands[a.cmd]
	if !ok {
		return
	}

	if b.shouldIgnore(m) {
		return
	}

	if c.acceptArgs && len(a.args) > 0 && !commandArgumentRegexp.MatchString(strings.Join(a.args, " ")) {
		return
	}

	args := c.args
	if c.acceptArgs && len(a.args) > 0 {
		args = append(args, a.args...)
	}

	cmd := exec.Command(c.bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		b.logger.Printf("commandHandler: %v", err)
		b.privmsg(b.IRC.CommandErrExec)
		return
	}

	// The output can span over multiple lines. Since the IRC protocol
	// doesn't support sending multiple lines in the same message we'll
	// have to split it and send each line separately.
	rows := strings.Split(string(out), "\n")
	for _, o := range rows {
		if o == "" {
			continue
		}

		b.privmsg(o)
	}
}

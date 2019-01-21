package main

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/osm/irc"
)

// initFactoidDefaults sets default values for all settings.
func (b *bot) initCommandDefaults() {
	if b.IRC.CommandErrExec == "" {
		b.IRC.CommandErrExec = "command execution error"
	}
}

// commandArgumentRegexp is a regexp that makes sure that we don't get
// unwanted characters passed to the external commands that are to be
// executed.
var commandArgumentRegexp = regexp.MustCompile("^[a-zA-Z0-9 ]*$")

// commandHandler exposes external commands that are defined in the Commands
// secion of the configuration file. If the IRC message matches a key of the
// defined command it executes it and returns the output to the user. If the
// message was seen in a channel it will return it back to the channel,
// otherwise it will be sent back to the user.
func (b *bot) commandHandler(m *irc.Message) {
	if b.shouldIgnore(m) {
		return
	}

	a := b.parseAction(m).(*privmsgAction)

	bin, ok := b.IRC.Commands[a.cmd]
	if !ok {
		return
	}

	if len(a.args) > 0 && !commandArgumentRegexp.MatchString(strings.Join(a.args, " ")) {
		return
	}

	cmd := exec.Command(bin, a.args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		b.logger.Printf("commandHandler: %v", err)
		b.privmsgf(b.IRC.CommandErrExec)
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

		b.privmsgf(o)
	}
}

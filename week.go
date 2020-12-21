package main

import (
	"github.com/osm/irc"
)

// initWeekDefaults sets default values for all settings.
func (b *bot) initWeekDefaults() {
	if b.IRC.WeekCmd == "" {
		b.IRC.WeekCmd = "!week"
	}
}

// weekCommandHandler handles the commands issued from the IRC channel.
func (b *bot) weekCommandHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}

	if a.cmd != b.IRC.WeekCmd {
		return
	}

	if b.shouldIgnore(m) {
		return
	}

	date := ""
	if len(a.args) == 1 {
		date = a.args[0]
	}

	b.privmsg(getWeek(date))
}

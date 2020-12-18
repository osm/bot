package main

import (
	"fmt"
	"strconv"
	"time"

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

	t := time.Now()
	if len(a.args) == 1 {
		if d, err := time.Parse("2006-01-02", a.args[0]); err == nil {
			fmt.Println(err)
			t = d
		}

	}

	_, week := t.ISOWeek()
	b.privmsg(strconv.Itoa(week))
}

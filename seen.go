package main

import (
	"database/sql"

	"github.com/osm/irc"
)

// initSeenHandler initializes the seen handler default messages.
func (b *bot) initSeenHandler() {
	if b.IRC.SeenMsgFound == "" {
		b.IRC.SeenMsgFound = "<nick> <date> <time>, saying <message>"
	}
	if b.IRC.SeenMsgNotFound == "" {
		b.IRC.SeenMsgNotFound = "<nick> has never been here"
	}
}

// seenHandler handle the !seen requests
func (b *bot) seenHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}
	if a.cmd != b.IRC.SeenCmd {
		return
	}
	if b.shouldIgnore(m) {
		return
	}
	if len(a.args) < 1 {
		return
	}

	nick := a.args[0]
	var timestamp string
	var message string
	err := b.queryRow(
		"SELECT timestamp, message FROM log WHERE nick = $1 ORDER BY timestamp DESC LIMIT 1",
		nick,
	).Scan(&timestamp, &message)

	if err != nil && err != sql.ErrNoRows {
		b.logger.Printf("seenHandler: %v", err)
		b.privmsg(b.DB.Err)
		return
	}

	if len(timestamp) == 0 {
		b.privmsgph(b.IRC.SeenMsgNotFound, map[string]string{
			"<nick>": nick,
		})
	} else {
		b.privmsgph(b.IRC.SeenMsgFound, map[string]string{
			"<nick>":    nick,
			"<date>":    timestamp[0:10],
			"<time>":    timestamp[11:16],
			"<message>": message,
		})
	}
}

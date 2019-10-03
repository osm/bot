package main

import (
	"github.com/osm/irc"
)

// loggingHandler implements basic logging of all the received messages. This
// functionality can be toggled by setting enableLogging to false in the
// configuration file.  The bot is written to support one and only one
// channel, so the channel information is NOT stored in the database.
func (b *bot) loggingHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}

	stmt, err := b.prepare("INSERT INTO log (id, timestamp, nick, message) VALUES(?, ?, ?, ?)")
	if err != nil {
		b.logger.Printf("loggingHandler: %v", err)
		b.privmsg(b.DB.Err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(newUUID(), newTimestamp(), a.nick, a.msg)
	if err != nil {
		b.logger.Printf("loggingHandler: %v", err)
		b.privmsg(b.DB.Err)
		return
	}
}

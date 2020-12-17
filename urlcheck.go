package main

import (
	"database/sql"
	"regexp"

	"github.com/osm/irc"
)

// initURLCheckDefaults sets default values for all settings.
func (b *bot) initURLCheckDefaults() {
	if b.IRC.URLCheckMsg == "" {
		b.IRC.URLCheckMsg = "LOL, <nick> posted <url> on <timestamp>"
	}
}

// urlRegexp extracts URLs from a text string. This regexp is used to extract
// URLs from IRC messages so that they can be compared to the already seen
// URLs that are stored in the database.
var urlRegexp = regexp.MustCompile(`(http|ftp|https):\/\/([\w\-_]+(?:(?:\.[\w\-_]+)+))([\w\-\.,@?^=%&amp;:/~\+#]*[\w\-\@?^=%&amp;/~\+#])?`)

// urlCheckHandler extracts and examines all URLs that the bot sees. When the
// URL has been seen before a humiliating message is sent to the channel.  If
// the URL hasn't been seen before it is stored in the database.  The bot is
// written to support one and only one channel, so the channel information is
// NOT stored in the database.
func (b *bot) urlCheckHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}

	url := urlRegexp.FindString(a.msg)
	if url == "" {
		return
	}

	var nick, timestamp string
	err := b.queryRow("SELECT nick, timestamp FROM url_check WHERE url = $1", url).Scan(&nick, &timestamp)
	if err != nil && err != sql.ErrNoRows {
		b.logger.Printf("urlCheckHandler: %v", err)
		b.privmsg(b.DB.Err)
		return
	}

	if nick != "" {
		b.privmsgph(b.IRC.URLCheckMsg, map[string]string{
			"<nick>":      nick,
			"<url>":       url,
			"<timestamp>": timestamp,
		})
		return
	}

	stmt, err := b.prepare("INSERT INTO url_check (id, timestamp, nick, url) VALUES($1, $2, $3, $4)")
	if err != nil {
		b.logger.Printf("urlCheckHandler: %v", err)
		b.privmsg(b.DB.Err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(newUUID(), newTimestamp(), a.nick, url)
	if err != nil {
		b.logger.Printf("urlCheckHandler: %v", err)
		b.privmsg(b.DB.Err)
		return
	}
}

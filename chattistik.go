package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/osm/irc"
)

// initChattistikDefaults sets default values for all settings.
func (b *bot) initChattistikDefaults() {
	if b.IRC.ChattistikCmd == "" {
		b.IRC.ChattistikCmdToday = "!chattistik"
	}
	if b.IRC.ChattistikCmdToday == "" {
		b.IRC.ChattistikCmdToday = "today"
	}
	if b.IRC.ChattistikCmdYesterday == "" {
		b.IRC.ChattistikCmdYesterday = "yesterday"
	}
	if b.IRC.ChattistikMsgNoStats == "" {
		b.IRC.ChattistikMsgNoStats = "There are no stats for the date"
	}
}

// chattistikDateRegexp defines a iso 8601 regexp.
var chattistikDateRegexp = regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2}$")

// chattistikHandler adds IRC chat statistic commands. The chattistik commands
// will only be available when IRC logging has been enabled.  The bot is
// written to support one and only one channel, so the channel information is
// NOT stored in the database.
func (b *bot) chattistikHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)
	if !a.validChannel {
		return
	}

	if a.cmd != b.IRC.ChattistikCmd {
		return
	}

	if len(a.args) != 1 {
		return
	}

	if b.shouldIgnore(m) {
		return
	}

	arg := a.args[0]
	if arg == b.IRC.ChattistikCmdToday {
		b.chattistik(time.Now().Format("2006-01-02"), "")
	} else if arg == b.IRC.ChattistikCmdYesterday {
		b.chattistik(time.Now().AddDate(0, 0, -1).Format("2006-01-02"), "")
	} else if m := chattistikDateRegexp.FindStringSubmatch(arg); len(m) == 1 {
		b.chattistik(arg, "")
	} else {
		b.chattistik(time.Now().Format("2006-01-02"), arg)
	}
}

// chattistik compiles a map of the nick and word count for all nicks that has
// been active during the from and to date range.
func (b *bot) chattistik(date, word string) {
	rows, err := b.query("SELECT message, nick FROM log WHERE substr(timestamp, 0, 11) = ?", date)
	if err != nil {
		b.logger.Printf("chattistik: %v", err)
		return
	}
	defer rows.Close()

	// Count how many words there has been for each nick We split on
	// spaces, so each space separated string is considered to be a word.
	stats := make(map[string]int)
	for rows.Next() {
		var msg, nick string
		rows.Scan(&msg, &nick)

		if _, ok := stats[nick]; !ok {
			stats[nick] = 0
		}

		if word == "" {
			stats[nick] += len(strings.Split(msg, " "))
		} else {
			for _, w := range strings.Split(msg, " ") {
				if strings.ToLower(w) == strings.ToLower(word) {
					stats[nick] += 1
				}
			}
		}
	}

	if len(stats) == 0 {
		b.privmsg(b.IRC.ChattistikMsgNoStats)
		return
	}

	// Construct a map of the stats but where the key is the word count
	// instead of the nick.
	count := make(map[int]string)
	for k, v := range stats {
		if _, ok := count[v]; !ok {
			count[v] = k
		} else {
			count[v] = fmt.Sprintf("%s, %s", count[v], k)
		}
	}

	// Sort the count map so that we can output the stats in a nicer way.
	sortedKeys := make([]int, 0, len(count))
	for k := range count {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedKeys)))

	for _, k := range sortedKeys {
		b.privmsg(fmt.Sprintf("%d: %s", k, count[k]))
	}
}

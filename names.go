package main

import (
	"strings"

	"github.com/osm/irc"
)

// handleNamesReply handles the 353 command (RPL_NAMREPLY) that is issued when
// the bot joins the channel for the first time. It returns a list of all the
// names that currently is in the channel. Big channels might return multiple
// 353 commands, so we'll have to lock the map before we use it to prevent
// race conditions.
func (b *bot) handleNamesReply(m *irc.Message) {
	// The raw IRC message will look something like this.
	// :irc.example.net 353 foo = #foo :foo +bar @baz
	// So we'll remove the foo = #foo part and keep the names.
	msg := strings.Join(m.ParamsArray[3:], " ")

	// There can be more than one response in big channels, so we'll have
	// to acquire a lock before we add anything to the names map.
	b.IRC.namesMu.Lock()
	defer b.IRC.namesMu.Unlock()

	// Names are separated with a space character, so we split on it and
	// check whether the name exists in the map before or not.
	for _, n := range strings.Split(msg, " ") {
		// Strip any protocol and status characters from the given name.
		if n[0] == ':' || n[0] == '+' || n[0] == '%' || n[0] == '@' {
			n = n[1:]
		}

		if _, ok := b.IRC.names[n]; !ok {
			b.IRC.names[n] = true
		}
	}
}

// handleNamesAdd adds a non existing name to the names map.
func (b *bot) handleNamesAdd(m *irc.Message) {
	b.IRC.namesMu.Lock()
	defer b.IRC.namesMu.Unlock()
	if _, ok := b.IRC.names[m.Name]; !ok {
		b.IRC.names[m.Name] = true
	}
}

// handleNamesRemove updates the names map of the bot and removes the name of
// the user that is either PARTing or QUITing.
func (b *bot) handleNamesRemove(m *irc.Message) {
	b.IRC.namesMu.Lock()
	defer b.IRC.namesMu.Unlock()
	if _, ok := b.IRC.names[m.Name]; ok {
		delete(b.IRC.names, m.Name)
	}
}

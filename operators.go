package main

import (
	"github.com/osm/irc"
)

// operatorsHandler will automatically give a joining user operator status if
// the host matches a host that exists in the operators section in the
// configuration.
func (b *bot) operatorsHandler(m *irc.Message) {
	if len(b.IRC.Operators) < 1 {
		return
	}

	a := b.parseAction(m).(*joinAction)
	if !a.validChannel {
		return
	}

	if _, ok := b.IRC.operators[a.host]; ok {
		b.IRC.client.Sendf("MODE %s +o %s", b.IRC.Channel, a.nick)
	}
}

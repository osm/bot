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

	for _, o := range b.IRC.operators {
		if o.Match([]byte(a.host)) {
			b.IRC.client.Sendf("MODE %s +o %s", b.IRC.Channel, a.nick)
			return
		}
	}
}

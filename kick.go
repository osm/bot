package main

import (
	"math/rand"
	"time"

	"github.com/osm/irc"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (b *bot) kickHandler(m *irc.Message) {
	// Wait between 1 and 5 seconds before rejoining the channel.
	time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
	b.IRC.client.Sendf("JOIN %s", b.IRC.Channel)
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/osm/irc"
)

// initLyssnarDefaults sets default values for all settings.
func (b *bot) initLyssnarDefaults() {
	if b.IRC.LyssnarCmd == "" {
		b.IRC.LyssnarCmd = "!lyssnar"
	}
	if b.IRC.LyssnarMsg == "" {
		b.IRC.LyssnarMsg = "<nick> listening to <song>"
	}
	if b.IRC.LyssnarMsgUserIsNotListening == "" {
		b.IRC.LyssnarMsgUserIsNotListening = "the user is not listening to anything right now"
	}
	if b.IRC.LyssnarErr == "" {
		b.IRC.LyssnarErr = "there's something wrong with the lyssnar.com integration"
	}
	if b.IRC.LyssnarErrUserNotConfigured == "" {
		b.IRC.LyssnarErrUserNotConfigured = "the user is not configured"
	}
}

// lyssnarHandler handles the lyssnar request from the IRC channel.
func (b *bot) lyssnarHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)
	if !a.validChannel {
		return
	}

	if a.cmd != b.IRC.LyssnarCmd {
		return
	}

	if b.shouldIgnore(m) {
		return
	}

	if len(a.args) != 1 {
		return
	}

	nick := a.args[0]
	spotifyUsername, ok := b.IRC.Lyssnare[nick]
	if !ok {
		b.privmsg(b.IRC.LyssnarErrUserNotConfigured)
		return
	}

	res, err := http.Get(fmt.Sprintf("http://lyssnar.com/v1/user/%s/currently-playing-short", spotifyUsername))
	if err != nil {
		b.logger.Printf("lyssnar: %v", err)
		b.privmsg(b.IRC.LyssnarErr)
		return
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		b.logger.Printf("lyssnar: %v", err)
		b.privmsg(b.IRC.LyssnarErr)
		return
	}

	var obj struct {
		Playing string `json:"playing"`
	}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		b.logger.Printf("lyssnar: %v", err)
		b.privmsg(b.IRC.LyssnarErr)
		return
	}

	if obj.Playing == "" {
		b.privmsg(b.IRC.LyssnarMsgUserIsNotListening)
		return
	}

	b.privmsgph(b.IRC.LyssnarMsg, map[string]string{
		"<nick>": nick,
		"<song>": obj.Playing,
	})
}

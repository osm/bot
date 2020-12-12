package main

import (
	"github.com/osm/pastebin"
)

// newPaste uploads the given title and content and prints the url to the
// channel.
func (b *bot) newPaste(title, content string) {
	if b.IRC.PastebinAPIKey == "" {
		b.logger.Printf("pastebin: you need to set a pastebin api key\n")
		return
	}
	pb := pastebin.New(b.IRC.PastebinAPIKey)

	var url string
	url, err := pb.NewPaste(content, title, pastebin.Unlisted, pastebin.TenMinutes)
	if err != nil {
		b.logger.Printf("pastebin: pastebin err: %v\n", err)
		return
	}

	b.privmsg(url)
}

package main

import (
	"strings"

	"github.com/osm/dumpinen"
	"github.com/osm/pastebin"
)

// newPaste uploads the given title and content and prints the url to the
// channel.
func (b *bot) newPaste(title, content string) {
	var url string
	var err error

	if b.IRC.EnableDumpinen {
		opts := []dumpinen.Option{
			dumpinen.WithAddr("https://dumpinen.com"),
			dumpinen.WithContentType("text/plain; charset=utf-8"),
		}

		d := dumpinen.NewClient(opts...)
		url, err = d.Dump(strings.NewReader(content))
		if err != nil {
			b.logger.Printf("dumpinen err: %v\n", err)
			return
		}
	} else {
		if b.IRC.PastebinAPIKey == "" {
			b.logger.Printf("pastebin: you need to set a pastebin api key\n")
			return
		}
		pb := pastebin.New(b.IRC.PastebinAPIKey)

		url, err = pb.NewPaste(content, title, pastebin.Unlisted, pastebin.TenMinutes)
		if err != nil {
			b.logger.Printf("pastebin: pastebin err: %v\n", err)
			return
		}
	}

	b.privmsg(url)
}

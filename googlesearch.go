package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/osm/irc"
)

// initGoogleSearchDefaults initializes the default properties for the google
// search command
func (b *bot) initGoogleSearchDefaults() {
	if b.IRC.GoogleSearchCmd == "" {
		b.IRC.WeatherCmd = "!g"
	}
}

// googleSearchCommandHandler handles the google search command.
func (b *bot) googleSearchCommandHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}
	if a.cmd != b.IRC.GoogleSearchCmd {
		return
	}
	if b.shouldIgnore(m) {
		return
	}
	if len(a.args) < 1 {
		return
	}

	// Initialize a new client that doesn't follow redirects.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Construct the I'm lucky URL.
	url := fmt.Sprintf(
		`https://www.google.com/search?q=%s&btnI=Jag+har+tur'`,
		url.QueryEscape(strings.Join(a.args, " ")),
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		b.logger.Printf("googleSearch: %v", err)
		return
	}

	// Google doesn't allow us to search unless we have a "real" user
	// agent.
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.141 Safari/537.36")
	req.Header.Set("Referer", "https://www.google.com/")

	// Perform the request.
	res, err := client.Do(req)
	if err != nil {
		b.logger.Printf("googleSearch: %v", err)
		return
	}

	// Extract the location from the response.
	l, err := res.Location()
	if err != nil || l == nil {
		b.logger.Printf("googleSearch: %v", err)
		return
	}

	// The response should include a "q" parameter which holds the URL of
	// the "I'm lucky" response.
	p := l.Query()
	if q, ok := p["q"]; ok && len(q) == 1 {
		b.privmsg(q[0])
		return
	}
}

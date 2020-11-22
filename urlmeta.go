package main

import (
	"net/http"
	"strings"

	"github.com/osm/irc"
	"github.com/osm/mex"
)

func (b *bot) urlMetaHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}

	// We'll reuse the regexp that is defined in urlcheck.go for this code
	// as well.
	url := urlRegexp.FindString(a.msg)
	if url == "" {
		return
	}

	// Check whether or not the given URL matches any of the configured
	// URLs that we want to fetch metadata for.
	fetchMeta := false
	for _, u := range b.IRC.URLMetaURLs {
		if strings.Contains(url, u) {
			fetchMeta = true
			break
		}
	}

	// No meta should be fetched for the URL, return.
	if !fetchMeta {
		return
	}

	// Download the URL, we'll return in silence if there's an error.
	res, err := http.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()

	// Extract the meta data and print it, if anything is returned.
	md := mex.Extract(res.Body)
	if md.Title != "" {
		b.privmsg(md.Title)
	}
	if md.Description != "" {
		b.privmsg(md.Description)
	}
}

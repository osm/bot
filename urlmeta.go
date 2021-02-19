package main

import (
	"io"
	"net/http"
	"strings"

	"github.com/osm/irc"
	"github.com/osm/mex"
)

// initURLMetaDefaults sets default values for all settings.
func (b *bot) initURLMetaDefaults() {
	if b.IRC.URLMetaMsg == "" {
		b.IRC.URLMetaMsg = "<description||title>"
	}
}

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

	// If no URLs are specified, we'll fetch meta for all found URLs.
	if len(b.IRC.URLMetaURLs) == 0 {
		fetchMeta = true
	}

	// But if we've got any URLs set, we'll make sure that only those are
	// processed.
	for _, u := range b.IRC.URLMetaURLs {
		if strings.Contains(url, u) {
			fetchMeta = true
			break
		}
	}

	// ... and last if the URL matches any of the ignore URLs we'll
	// disallow further processing.
	for _, u := range b.IRC.URLMetaIgnoreURLs {
		if strings.Contains(url, u) {
			fetchMeta = false
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

	// Limit the amount of data that is read to 10 MB.
	lr := &io.LimitedReader{R: res.Body, N: 10000000}

	// Extract the meta data and print it, if anything is returned.
	md := mex.Extract(lr)

	description := strings.TrimSpace(md.Description)
	title := strings.TrimSpace(md.Title)

	// No meta data found, return.
	if description == "" && title == "" {
		return
	}

	// Construct the "description or title"-field.
	dort := description
	if dort == "" {
		dort = title
	}

	b.privmsgph(b.IRC.URLMetaMsg, map[string]string{
		"<description>":        md.Description,
		"<title>":              md.Title,
		"<description||title>": dort,
	})
}

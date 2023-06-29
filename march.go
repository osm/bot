package main

import (
	_ "database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/osm/irc"
)

// initMarchDefaults initializes default values and makes sure that all
// required values has been set, if something is missing we'll exit with an
// error.
func (b *bot) initMarchDefaults() {
	if b.IRC.MarchURL == "" {
		fmt.Fprintf(os.Stderr, "error: no marchURL in config")
		os.Exit(1)
	}
	if b.IRC.MarchCredentials == "" {
		fmt.Fprintf(os.Stderr, "error: no marchCredentials in config")
		os.Exit(1)
	}

	for _, r := range b.IRC.MarchURLRegexps {
		b.IRC.marchURLRegexps = append(b.IRC.marchURLRegexps, regexp.MustCompile(r))
	}
}

// marchHandler listens for urls in the channel, if the url matches any
// regexps that's defined in the config we'll pass it on to the march url. The
// returned ID will be stored in the march table.
func (b *bot) marchHandler(m *irc.Message) {
	// Parse the action and make sure that it's a valid channel.
	a := b.parseAction(m).(*privmsgAction)
	if !a.validChannel {
		return
	}

	// Search for URLs in the message.
	u := urlRegexp.FindString(a.msg)
	if u == "" {
		return
	}

	// Iterate over the regexps, if no URL matches we'll return.
	var shouldArchive = false
	for _, r := range b.IRC.marchURLRegexps {
		if r.Match([]byte(u)) {
			shouldArchive = true
			break
		}
	}
	if !shouldArchive {
		return
	}

	// Make sure that we haven't archived the URL before, if so we'll
	// return early.
	var urlExists bool
	b.queryRow("SELECT 1 FROM march WHERE url = $1", u).Scan(&urlExists)
	if urlExists {
		return
	}

	// Prepare the POST data that we'll send to our archive.
	params := url.Values{}
	params.Set("url", u)
	postData := strings.NewReader(params.Encode())

	// Initialize a new http client that we'll use to send the request to
	// the march server.
	client := &http.Client{}
	req, err := http.NewRequest("POST", b.IRC.MarchURL, postData)
	req.Header.Add("Authorization", "Basic "+base64Encode(b.IRC.MarchCredentials))
	req.Header.Add("Content-type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)

	// Do the basic error checking.
	if err != nil {
		b.logger.Printf("march: post error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		b.logger.Printf("march: unexpected status code: %d", resp.StatusCode)
		return
	}
	foreignID, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		b.logger.Printf("march: unable to read body: %v", err)
		return
	}
	if string(foreignID) == "" {
		b.logger.Printf("march: no foreign id returned: %v", err)
		return
	}
	if !isUUID(string(foreignID)) {
		b.logger.Printf("march: foreign id isn't a valid uuid: %v", err)
		return
	}

	// Everything seems to be in order, let's insert the archived item.
	stmt, err := b.prepare("INSERT INTO march (id, url, foreign_id, inserted_at) VALUES($1, $2, $3, $4)")
	if err != nil {
		b.logger.Printf("march: prepare insert failed: %v", err)
		b.privmsg(b.DB.Err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(newUUID(), u, foreignID, newTimestamp())
	if err != nil {
		b.logger.Printf("march: insert failed: %v", err)
		b.privmsg(b.DB.Err)
		return
	}
}

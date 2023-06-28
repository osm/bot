package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

// updateNotifierLatestCommitRegexp contains the regular expression that
// extracts the latest commit sha1 from the github page of the bot.
var updateNotifierLatestCommitRegexp *regexp.Regexp

// init initializes the regexp.
func init() {
	updateNotifierLatestCommitRegexp = regexp.MustCompile("commit-tease-sha.*([0-9a-f]{40})")
}

// initUpdateNotifier sets default values for the update notifier.
func (b *bot) initUpdateNotifier() {
	if b.IRC.UpdateNotifierMsg == "" {
		b.IRC.UpdateNotifierMsg = "new version of the bot is available (<version>)"
	}
}

// updateNotifierHandler downloads the HTML source of the main github page and
// extracts the latest commit, if the currently running version of the bot
// isn't equal to the latest commit we'll send a message to all users in the
// update notifier names array.
func (b *bot) updateNotifierHandler() {
	// Return early if we don't have anyone to notify.
	if len(b.IRC.UpdateNotifierNames) == 0 {
		return
	}

	for {
		// Let's sleep for an hour before we perform the check.
		time.Sleep(1 * time.Hour)

		// Download the HTML source of the bot repo.
		res, err := http.Get("https://github.com/osm/bot")
		if err != nil {
			b.logger.Printf("updateNotifier: %v", err)
			return
		}

		// Read the body.
		defer res.Body.Close()
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			b.logger.Printf("updateNotifier: %v", err)
			return
		}

		// Try to extract the latest commit.
		matches := updateNotifierLatestCommitRegexp.FindStringSubmatch(string(data))
		if matches == nil {
			b.logger.Printf("updateNotifier: nothing found")
			return
		}

		// The version has changed, so let's notify everyone that needs to
		// know. We don't want to keep notifying, so we'll end the
		// goroutine after the notification has been sent.
		version := matches[1][0:8]
		if version != VERSION {
			for _, name := range b.IRC.UpdateNotifierNames {
				b.privmsgpht(b.IRC.UpdateNotifierMsg, name, map[string]string{
					"<version>": version,
				})
				return
			}
		}
	}
}

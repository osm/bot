package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

// initSupernytt sets default values for the supernytt integration.
func (b *bot) initSupernytt() {
	if b.IRC.SupernyttGrammarMessage == "" {
		b.IRC.SupernyttGrammarMessage = "<title> - <content>"
	}
}

// supernyttHandler is the goroutine that handles fetching and posting of the
// news.
func (b *bot) supernyttHandler() {
	for {
		// newEntries will be populated with entries that we haven't
		// stored or sent to the IRC channel.
		var newEntries []SNEntry

		// Fetch the latest external id from the database. The ID will
		// be used to determine how many new news entries we've got.
		latestExternalID := b.getLatestSupernyttExternalID()

		// Download supernytt data, if the method returns nil
		// something went wrong and we'll immediately goto the sleep
		// part of the loop.
		entries := b.getSupernyttData()
		if entries == nil {
			goto sleep
		}

		if latestExternalID == "" {
			// We didn't get an external id from the database, so
			// this probably means that we haven't executed the
			// supernytt handler before, so we'll use the latest
			// entry and insert it and send it to the channel.
			newEntries = append(newEntries, entries[0])
		} else {
			// If the first entry in the downloaded entries has the same
			// id as our latest external id we'll sleep right away
			// since we are caught up with all the latest news.
			if entries[0].ID == latestExternalID {
				goto sleep
			}

			// Iterate over the entries and extract the latest news.
			for _, e := range entries {
				// When the id of the entry is the same as our latest
				// external id we should stop the loop since we are
				// caught up.
				if e.ID == latestExternalID {
					break
				}

				// Store it in the newEntries slice.
				newEntries = append(newEntries, e)
			}
		}

		// No newEntries means that we should sleep.
		if len(newEntries) == 0 {
			goto sleep
		}

		// Iterate over the new entries and insert them into the
		// database and output them to the channel.
		for _, ne := range newEntries {
			b.insertSNEntry(&ne)
			b.privmsgph(b.IRC.SupernyttGrammarMessage, map[string]string{
				"<title>":   ne.Title.Value,
				"<content>": ne.getContent(),
			})
		}

	sleep:
		// Wait 10 minutes before we fetch the news again.
		time.Sleep(1 * time.Minute)
	}
}

// getSupernyttData downloads the latest news from supernytt and returns a
// slice of SNEntry objects.
func (b *bot) getSupernyttData() []SNEntry {
	res, err := http.Get("https://direkte.vg.no/api/ab/newsflow/5d0b819f4c641c00121148c9/entries?offset=0&showAdverts=false")
	if err != nil {
		b.logger.Printf("getSupernyttData: %w", err)
		return nil
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		b.logger.Printf("getSupernyttData: %w", err)
		return nil
	}

	var sn struct {
		Entries []SNEntry
	}
	err = json.Unmarshal(data, &sn)
	if err != nil {
		b.logger.Printf("getSupernyttData: %w", err)
		return nil
	}

	return sn.Entries
}

// getLatestSupernyttExternalID fetches the latest external id from the
// supernytt table.
func (b *bot) getLatestSupernyttExternalID() string {
	var externalID string
	err := b.queryRow("SELECT external_id FROM supernytt ORDER BY inserted_at DESC").Scan(&externalID)
	if err != nil && err != sql.ErrNoRows {
		b.logger.Printf("getLatestSupernyttID: %v", err)
		return ""
	}
	return externalID
}

// insertSNEntry inserts a supernytt entry into the database.
func (b *bot) insertSNEntry(e *SNEntry) {
	stmt, err := b.prepare("INSERT INTO supernytt (id, external_id, title, content, external_created, inserted_at) VALUES(?, ?, ?, ?, ?, ?);")
	if err != nil {
		b.logger.Printf("insertSNEntry: %w", err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		newUUID(),
		e.ID,
		e.Title.Value,
		e.getContent(),
		e.Changes.Created,
		newTimestamp(),
	)
	if err != nil {
		b.logger.Printf("insertSNEntry: %w", err)
	}
}

// SupernyttEntri defines a supernytt entry.
type SNEntry struct {
	ID    string
	Title struct {
		Value string
	}
	Changes struct {
		Created string
		Update  string
	}
	Components []struct {
		Type string
		Text struct {
			Value string
		}
	}
}

// getContent returns an IRC friendly content string.
func (se *SNEntry) getContent() string {
	var content string

	for _, c := range se.Components {
		// Skip all components that aren't plain text.
		if c.Type != "text" {
			continue
		}

		// We don't want to create too long messages, so if we have
		// more than 200 characters in the string we'll skip the
		// remaining text components.
		if len(content) > 200 {
			continue
		}

		// Append the text to our content string
		content += " " + c.Text.Value
	}

	return content
}

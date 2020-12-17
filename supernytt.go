package main

import (
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
		// Download supernytt data, if the method returns nil
		// something went wrong and we'll immediately goto the sleep
		// part of the loop.
		entries := b.getSupernyttData()
		if entries == nil {
			goto sleep
		}

		// Iterate over the entries, we'll check if the ID of the
		// entry already exists in our database, if it does we'll
		// ignore it so that we don't store and output the same entry
		// again.
		for _, e := range entries {
			// We've already handled the entity, carry on.
			if b.hasSNExternalID(e.ID) {
				continue
			}

			// Store it.
			b.insertSNEntry(&e)

			// Output it to the channel
			b.privmsgph(b.IRC.SupernyttGrammarMessage, map[string]string{
				"<title>":   e.Title.Value,
				"<content>": e.getContent(),
			})

			// To prevent spamming we'll sleep for a minute before
			// we proceed.
			time.Sleep(1 * time.Minute)
		}

	sleep:
		// Wait 10 minutes before we fetch the news again.
		time.Sleep(10 * time.Minute)
	}
}

// getSupernyttData downloads the latest news from supernytt and returns a
// slice of SNEntry objects.
func (b *bot) getSupernyttData() []SNEntry {
	res, err := http.Get("https://direkte.vg.no/api/ab/newsflow/5d0b819f4c641c00121148c9/entries?offset=0&limit=5&showAdverts=false")
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

// hasSNExternalID checks whether or not the external id exists.
func (b *bot) hasSNExternalID(externalID string) bool {
	var exists string
	err := b.queryRow("SELECT 1 FROM supernytt WHERE external_id = $1", externalID).Scan(&exists)
	if err != nil {
		return false
	}
	return true
}

// insertSNEntry inserts a supernytt entry into the database.
func (b *bot) insertSNEntry(e *SNEntry) {
	stmt, err := b.prepare("INSERT INTO supernytt (id, external_id, title, content, external_created, inserted_at) VALUES($1, $2, $3, $4, $5, $6);")
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

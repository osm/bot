package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/osm/irc"
)

// Define Tenor errors
var (
	TenorNoAPIKey     = errors.New("You need to set a Tenor API key")
	TenorNothingFound = errors.New("Nothing found")
)

// init seeds the randomizer.
func init() {
	rand.Seed(time.Now().Unix())
}

// iniTenorDefaults sets default values for all settings.
func (b *bot) initTenorDefaults() {
	if b.IRC.TenorCmd == "" {
		b.IRC.TenorCmd = "!tenor"
	}
	if b.IRC.TenorLang == "" {
		b.IRC.TenorLang = "en_US"
	}
	if b.IRC.TenorMsgNothingFound == "" {
		b.IRC.TenorMsgNothingFound = "nothing found"
	}
}

// tenorHandler handles the Tenor integration.
func (b *bot) tenorHandler(m *irc.Message) {
	if b.shouldIgnore(m) {
		return
	}

	a := b.parseAction(m).(*privmsgAction)
	if !a.validChannel {
		return
	}

	if a.cmd != b.IRC.TenorCmd {
		return
	}

	if b.IRC.TenorAPIKey == "" {
		b.logger.Printf("tenorHandler: you need to set a Tenor API key\n")
		return
	}

	url, err := b.tenorSearch(a.msg)
	if err == TenorNothingFound {
		b.privmsgf(b.IRC.TenorMsgNothingFound)
	} else if url != "" {
		b.privmsgf(url)
	}
}

// tenorSearch search and return a random gif for the given query.
func (b *bot) tenorSearch(query string) (string, error) {
	if b.IRC.TenorAPIKey == "" {
		b.logger.Printf("tenorSearch: you need to set a Tenor API key\n")
		return "", TenorNoAPIKey
	}

	url := fmt.Sprintf(
		"https://api.tenor.com/v1/random?key=%s&q=%s&locale=%s&media_filter=minimal",
		b.IRC.TenorAPIKey,
		strings.Replace(strings.Replace(query, fmt.Sprintf("%s ", b.IRC.TenorCmd), "", 1), " ", "%20", -1),
		b.IRC.TenorLang,
	)

	res, err := http.Get(url)
	if err != nil {
		b.logger.Printf("tenor: %w", err)
		return "", err
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		b.logger.Printf("tenor: %w", err)
		return "", err
	}

	var d struct {
		Results []struct {
			Media []map[string]struct {
				URL string `json:"url"`
			}
		}
	}
	err = json.Unmarshal(data, &d)
	if err != nil {
		b.logger.Printf("tenor: %w", err)
		return "", err
	}

	if len(d.Results) == 0 {
		return "", TenorNothingFound
	}

	return d.Results[0].Media[0]["gif"].URL, nil
}

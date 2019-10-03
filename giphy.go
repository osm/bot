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

// Define Giphy errors
var (
	GiphyNoAPIKey          = errors.New("You need to set a Giphy API key")
	GiphyUnknownStatusCode = errors.New("Expected status code 200 from the Giphy API")
	GiphyNothingFound      = errors.New("Nothing found")
)

// init seeds the randomizer.
func init() {
	rand.Seed(time.Now().Unix())
}

// initGiphyDefaults sets default values for all settings.
func (b *bot) initGiphyDefaults() {
	if b.IRC.GiphyCmd == "" {
		b.IRC.GiphyCmd = "!giphy"
	}
	if b.IRC.GiphyLang == "" {
		b.IRC.GiphyLang = "en"
	}
	if b.IRC.GiphyMsgNothingFound == "" {
		b.IRC.GiphyMsgNothingFound = "nothing found"
	}
}

// GiphyData structure of the data we need from the Giphy API.
type GiphyData struct {
	Images struct {
		Original struct {
			URL string `json:"url"`
		}
	}
}

// GiphyMeta the meta data returned by the Giphy API in each request.
type GiphyMeta struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

// giphyHandler handles the giphy integration.
// If arguments are passed a search will be performed, otherwise a random
// giphy will be returned to the channel.
func (b *bot) giphyHandler(m *irc.Message) {
	if b.shouldIgnore(m) {
		return
	}

	a := b.parseAction(m).(*privmsgAction)
	if !a.validChannel {
		return
	}

	if a.cmd != b.IRC.GiphyCmd {
		return
	}

	if b.IRC.GiphyAPIKey == "" {
		b.logger.Printf("giphyHandler: you need to set a giphy api key\n")
		return
	}

	var giphy string
	var err error
	if len(a.args) == 0 {
		giphy, err = b.giphyRandom()
	} else {
		giphy, err = b.giphySearch(a.msg)
	}

	if err == GiphyNothingFound {
		b.privmsg(b.IRC.GiphyMsgNothingFound)
	} else if giphy != "" {
		b.privmsg(giphy)
	}
}

// giphyStripCid strips everything after the ?cid part of the URL.
func giphyStripCid(url string) string {
	return url[:strings.Index(url, "?cid")]
}

// giphyRandom prints a random giphy to the chanel.
func (b *bot) giphyRandom() (string, error) {
	if b.IRC.GiphyAPIKey == "" {
		b.logger.Printf("giphyRandom: you need to set a giphy api key\n")
		return "", GiphyNoAPIKey
	}

	url := fmt.Sprintf(
		"https://api.giphy.com/v1/gifs/random?api_key=%s&rating=R",
		b.IRC.GiphyAPIKey,
	)

	res, err := http.Get(url)
	if err != nil {
		b.logger.Printf("giphy: %w", err)
		return "", err
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		b.logger.Printf("giphy: %w", err)
		return "", err
	}

	var g struct {
		Data GiphyData
		Meta GiphyMeta
	}
	err = json.Unmarshal(data, &g)
	if err != nil {
		b.logger.Printf("giphy: %w", err)
		return "", err
	}

	if g.Meta.Status != 200 {
		return "", GiphyUnknownStatusCode
	}

	return giphyStripCid(g.Data.Images.Original.URL), nil
}

// giphySearch search and return a random giphy for the given query.
func (b *bot) giphySearch(query string) (string, error) {
	if b.IRC.GiphyAPIKey == "" {
		b.logger.Printf("giphySearch: you need to set a giphy api key\n")
		return "", GiphyNoAPIKey
	}

	url := fmt.Sprintf(
		"https://api.giphy.com/v1/gifs/search?api_key=%s&q=%s&rating=R&lang=%s",
		b.IRC.GiphyAPIKey,
		strings.Replace(strings.Replace(query, fmt.Sprintf("%s ", b.IRC.GiphyCmd), "", 1), " ", "%20", -1),
		b.IRC.GiphyLang,
	)

	res, err := http.Get(url)
	if err != nil {
		b.logger.Printf("giphy: %w", err)
		return "", err
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		b.logger.Printf("giphy: %w", err)
		return "", err
	}

	var g struct {
		Data []GiphyData
		Meta GiphyMeta
	}
	err = json.Unmarshal(data, &g)
	if err != nil {
		b.logger.Printf("giphy: %w", err)
		return "", err
	}

	if g.Meta.Status != 200 {
		return "", GiphyUnknownStatusCode
	}

	if len(g.Data) == 0 {
		return "", GiphyNothingFound
	}

	return giphyStripCid(g.Data[rand.Intn(len(g.Data))].Images.Original.URL), nil
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/osm/irc"
)

type dictionary struct {
	dictionary  map[string]string
	foundMsg    string
	notFoundMsg string
}

var dictionaries map[string]dictionary

// initDictionaries initializes the dictionaries.
func (b *bot) initDictionaries() {
	dictionaries = make(map[string]dictionary)

	// Iterate over the dictionaries, read the dictionary into memory and
	// store it in the global dictionary map.
	for _, d := range b.IRC.Dictionaries {
		file, err := ioutil.ReadFile(d.Dictionary)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dictionary: cant open dictionary %s\n", d.Dictionary)
			os.Exit(1)
		}

		var di map[string]string
		err = json.Unmarshal(file, &di)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dictionary: cant decode dictionary %s\n", d.Dictionary)
			os.Exit(1)
		}

		dictionaries[d.Trigger] = dictionary{
			dictionary:  di,
			foundMsg:    d.FoundMsg,
			notFoundMsg: d.NotFoundMsg,
		}
	}
}

func (b *bot) dictionaryHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}

	if b.shouldIgnore(m) {
		return
	}

	entry, hasEntry := dictionaries[a.cmd]
	if !hasEntry {
		return
	}

	key := strings.Join(a.args, " ")
	value, hasValue := entry.dictionary[key]
	if !hasValue {
		b.privmsgph(entry.notFoundMsg, map[string]string{
			"<key>": key,
		})
		return
	}

	b.privmsgph(entry.foundMsg, map[string]string{
		"<key>":   key,
		"<value>": value,
	})
}

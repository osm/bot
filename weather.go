package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/osm/irc"
)

// initWeatherDefaults sets default values for all settings.
func (b *bot) initWeatherDefaults() {
	if b.IRC.WeatherCmd == "" {
		b.IRC.WeatherCmd = "!w"
	}
	if b.IRC.WeatherErr == "" {
		b.IRC.WeatherErr = "unknown city"
	}
	if b.IRC.WeatherMsg == "" {
		b.IRC.WeatherMsg = "<city>, <main>, <description>: <temp>"
	}
}

// weatherHandler contains the entry point for if the weather module is
// enabled. It will run all the messages through the regexp that is defined in
// the configuration file and pass it to the weather api.  This handler
// requires a valid API key in the configuration.
func (b *bot) weatherHandler(m *irc.Message) {
	if b.shouldIgnore(m) {
		return
	}

	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}

	if a.cmd != b.IRC.WeatherCmd {
		return
	}

	if b.IRC.WeatherAPIKey == "" {
		b.logger.Printf("weatherHandler: you need to set a weather api key\n")
		return
	}

	if len(a.args) != 1 {
		return
	}

	res, err := http.Get(fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?appid=%s&q=%s", b.IRC.WeatherAPIKey, a.args[0]))
	if err != nil {
		b.logger.Printf("weather: %v", err)
		b.privmsg(b.IRC.WeatherErr)
		return
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		b.logger.Printf("weather: %v", err)
		b.privmsg(b.IRC.WeatherErr)
		return
	}

	var w struct {
		Cod  json.Number `json:"cod"`
		Name string
		Main struct {
			Temp float32
		}
		Weather []struct {
			Main        string
			Description string
		}
	}

	err = json.Unmarshal(data, &w)
	if err != nil {
		b.logger.Printf("weather: %v", err)
		b.privmsg(b.IRC.WeatherErr)
		return
	}

	// Invalid response from the API, this usually indicates that someone
	// has been trying to be funny and sent an invalid city name. We
	// don't log these errors.
	if string(w.Cod) != "200" {
		b.privmsg(b.IRC.WeatherErr)
		return
	}

	b.privmsgph(b.IRC.WeatherMsg, map[string]string{
		"<city>":        w.Name,
		"<main>":        w.Weather[0].Main,
		"<description>": w.Weather[0].Description,
		"<temp>":        fmt.Sprintf("%v", int(w.Main.Temp-272.15)),
	})
}

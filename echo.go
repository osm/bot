package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// initEcho initializes the default values for the echo route.
func (b *bot) initEcho() {
	if b.HTTP.EchoRoute == "" {
		b.HTTP.EchoRoute = "/echo"
	}
	if b.HTTP.EchoMethod == "" {
		b.HTTP.EchoMethod = "POST"
	}
}

// echoHandler takes the POST-data and sends it back to the IRC channel as a
// regular IRC message.
func (b *bot) echoHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		b.HTTP.logger.Printf("echo: unable to read body, %w", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}

	data := string(body)
	if data == "" {
		return
	}

	b.privmsgf(data)
	fmt.Fprintf(w, data)
}

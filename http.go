package main

import (
	"fmt"
	"net/http"
)

// initHTTP initializes the HTTP server.
func (b *bot) initHTTP() {
	if b.HTTP.EnableEcho {
		b.initEcho()
	}

	// Handle all routing from here.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b.HTTP.logger.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		if b.HTTP.EnableEcho && r.URL.Path == b.HTTP.EchoRoute && r.Method == b.HTTP.EchoMethod {
			b.echoHandler(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404")
		}
	})

	// Fire up the server in a goroutine so that we aren't blocking.
	go func() {
		http.ListenAndServe(b.HTTP.Address+":"+b.HTTP.Port, nil)
		b.mainWG.Done()
	}()
}

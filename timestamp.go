package main

import (
	"time"
)

// newTimestamp returns a timestamp.
func newTimestamp() string {
	return time.Now().Format("2006-01-02T15:04:05.999")
}

// newUnixTimestamp returns a UNIX timestamp.
func newUnixTimestamp() int {
	return int(time.Now().Unix())
}

// newDate returns the current date.
func newDate() string {
	return time.Now().Format("2006-01-02")
}

// newHour returns the current hour.
func newHour() string {
	return time.Now().Format("15")
}

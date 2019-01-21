package main

import (
	"time"
)

// newTimestamp returns a timestamp.
func newTimestamp() string {
	return time.Now().Format("2006-01-02T15:04:05.999")
}

package main

import (
	"fmt"
	"strconv"
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

// newDateWithDuration returns the current date and adds the given duration.
func newDateWithDuration(d time.Duration) string {
	return time.Now().Add(d).Format("2006-01-02")
}

// newHour returns the current hour.
func newHour() string {
	return time.Now().Format("15")
}

// getWeek returns the week number for the given date, or the current week if
// the argument is empty.
func getWeek(date string) string {
	t := time.Now()
	if date != "" {
		if d, err := time.Parse("2006-01-02", date); err == nil {
			fmt.Println(err)
			t = d
		}

	}

	_, week := t.ISOWeek()
	return strconv.Itoa(week)
}

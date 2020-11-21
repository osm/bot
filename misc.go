package main

import (
	"strconv"
)

// stringToInt converts a string to an integer and ignores possible errors.
func stringToInt(t string) int {
	i, _ := strconv.Atoi(t)
	return i
}

package main

import (
	"strconv"
	"strings"
)

// stringToInt converts a string to an integer and ignores possible errors.
func stringToInt(t string) int {
	i, _ := strconv.Atoi(t)
	return i
}

// fmtNumber formats the number and returns it as a string.
func fmtNumber(s, l string) string {
	if l == "sv-SE" {
		return strings.Replace(s, ".", ",", -1)
	}

	return s
}

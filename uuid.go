package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"regexp"
)

// newUUID generates a random UUID according to RFC 4122
func newUUID() string {
	// Generate 16 random bytes
	uuid := make([]byte, 16)
	io.ReadFull(rand.Reader, uuid)

	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80

	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40

	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// isUUIDRe defines a regular expression that checks for UUIDs.
var isUUIDRe = regexp.MustCompile("(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

// isUUID checks whether the submitted subject is an UUID.
func isUUID(s string) bool {
	return isUUIDRe.Match([]byte(s))
}

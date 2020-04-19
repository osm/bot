package main

import (
	"encoding/base64"
)

// base64Encode takes the given string and returns a base64 encoded version of
// it.
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

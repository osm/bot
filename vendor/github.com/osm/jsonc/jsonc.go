package jsonc

import (
	"strings"
	"unicode/utf8"
)

// jsonc is an internal structure that is used when stripping comments from
// the provided JSON string.
type jsonc struct {
	input  string
	isJSON bool
	pos    int
	width  int
}

// isEOF checks whether or not the pos is greater than the total length of the
// input string.
func (j *jsonc) isEOF() bool {
	return j.pos >= len(j.input)
}

// next returns the next rune from the input string.
func (j *jsonc) next() rune {
	r, w := utf8.DecodeRuneInString(j.input[j.pos:])
	j.width = w
	j.pos += j.width
	return r
}

// prev sets the position to the previous rune.
func (j *jsonc) prev() {
	j.pos -= j.width
}

// peek check what the coming rune is.
func (j *jsonc) peek() rune {
	r := j.next()
	j.prev()
	return r
}

// discardUntil discards every character up until the provided character
// occurs.
func (j *jsonc) discardUntil(c string) {
	for !j.isEOF() && !strings.ContainsRune(c, j.next()) {
	}
	j.prev()
}

// toJSON strips all comments from a JSON string.
func (j *jsonc) toJSON() string {
	var output []rune

	for !j.isEOF() {
		c := j.next()

		// Determine whether or not we are inside a JSON block, we
		// also need to check that the current character isn't an
		// escaped double quote, otherwise we might end the JSON block
		// too early.
		if !j.isJSON && c != '\\' && j.peek() == '"' {
			j.isJSON = true
		} else if j.isJSON && c != '\\' && j.peek() == '"' {
			j.isJSON = false
		}

		// If we aren't inside a JSON block and the two coming
		// characters are two slashes we'll assume that a comment is
		// coming up, so we'll discard everything up until a new line
		// character occurs.
		// unil the next new line character, since it's a comment.
		if !j.isJSON && c == '/' && j.peek() == '/' {
			j.discardUntil("\n")
			continue
		}

		// All other characters should be stored and returned.
		output = append(output, c)
	}

	return string(output)
}

// ToJSON returns a JSON string without comments.
func ToJSON(input string) string {
	j := &jsonc{input: input}
	return j.toJSON()
}

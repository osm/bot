package ww

import (
	"fmt"
	"strings"
)

// Wrap returns a slice of the given string, wrapped when the given width is
// reached.
func Wrap(str string, width int) []string {
	// We should never really wrap with width set to 1, but if we get
	// called with width set to 1 we'll just return a slice of the given
	// string.
	if width == 1 {
		return []string{str}
	}

	// If the length of the string is shorter than the width we'll return
	// it immediately.
	if len(str) <= width {
		return []string{str}
	}

	// Split the string on spaces, if the length of the splitted array is
	// equal to 1 it means that we don't have any spaces so we can return
	// the string right away.
	words := strings.Split(str, " ")
	if len(words) == 1 {
		return []string{str}
	}

	var r []string
	var b strings.Builder
	for _, w := range words {
		// The string exists and the length of the current string +
		// the length of the word that we are currently processing
		// exceeds the line width, that means it's time to wrap, so
		// we'll append the current string and reset the string
		// builder.
		if b.Len() > 0 && b.Len()+len(w) >= width {
			r = append(r, b.String())
			b.Reset()
		}

		// If we have an empty string we'll just write the current
		// word to the string builder. If not, we'll write a space and
		// the current word to the string.
		if b.Len() == 0 {
			fmt.Fprintf(&b, "%s", w)
		} else {
			fmt.Fprintf(&b, " %s", w)
		}
	}

	// We have started to build a string that never got added within the
	// loop, so let's add it.
	if b.Len() > 0 {
		r = append(r, b.String())
	}

	return r
}

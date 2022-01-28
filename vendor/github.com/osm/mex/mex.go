package mex

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// MetaData holds the data that is returned by the Exttract function.
type MetaData struct {
	Title       string
	Description string
}

func Extract(resp io.Reader) *MetaData {
	md := &MetaData{}

	// Initialize a new tokenizer.
	tokenizer := html.NewTokenizer(resp)

	// Iterate over the tokenizer and try to extract the meta data.
	for {
		// Get the next token type and make sure it isn't an error.
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			return md
		}

		// Get the current token.
		t := tokenizer.Token()

		// </head> is reached, so we are not interested in the content
		// any more.
		if t.Type == html.EndTagToken && t.Data == "head" {
			return md
		}

		// We've reached a <meta>-tag, extract the values.
		if t.Data == "meta" {
			var p, c string

			for _, attr := range t.Attr {
				switch attr.Key {
				case "name", "property":
					p = strings.ToLower(attr.Val)
					break

				case "content":
					c = attr.Val
					break
				}
			}

			switch p {
			case "og:title":
				md.Title = c
				break
			case "og:description":
				md.Description = c
				break
			}

		}
	}
}

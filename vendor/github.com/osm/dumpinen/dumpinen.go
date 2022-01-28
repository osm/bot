package dumpinen

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// dumpinen contains the private structure of the dumpinent client and
// options.
type dumpinen struct {
	addr        string
	username    string
	password    string
	deleteAfter *string
	contentType *string
}

// NewClient initializes a new dumpinen client.
func NewClient(opts ...Option) *dumpinen {
	d := &dumpinen{}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

type Option func(*dumpinen)

// WithAddr sets the address of the dumpinen server.
func WithAddr(addr string) Option {
	return func(d *dumpinen) {
		d.addr = addr
	}
}

// WithCredentials sets the credentials to be used when talking to the
// dumpinen server.
func WithCredentials(c string) (Option, error) {
	p := strings.Split(c, ":")
	if len(p) < 2 {
		return nil, fmt.Errorf("credentials should be formatted as username:password")
	}

	return func(d *dumpinen) {
		d.username = p[0]
		d.password = strings.Join(p[1:], ":")
	}, nil
}

// WithDeleteAfter sets the delete after duration that will be used in the
// request to the dumpinen server.
func WithDeleteAfter(dur string) (Option, error) {
	_, err := time.ParseDuration(dur)
	if err != nil {
		return nil, fmt.Errorf(`error: the given duration "%s" is not valid`, dur)
	}

	return func(d *dumpinen) {
		d.deleteAfter = &dur
	}, nil
}

// WithContentType sets the content type for the dumped file.
func WithContentType(ct string) Option {
	return func(d *dumpinen) {
		d.contentType = &ct
	}
}

// Dump takes a io.Reader and uploads the contents to the dumpinen server.
func (d *dumpinen) Dump(r io.Reader) (string, error) {
	req, err := http.NewRequest("POST", d.addr, r)
	if err != nil {
		return "", err
	}

	if d.username != "" && d.password != "" {
		req.SetBasicAuth(d.username, d.password)
	}

	if d.deleteAfter != nil || d.contentType != nil {
		q := req.URL.Query()

		if d.deleteAfter != nil {
			q.Add("deleteAfter", *d.deleteAfter)
		}

		if d.contentType != nil {
			q.Add("contentType", *d.contentType)
		}

		req.URL.RawQuery = q.Encode()
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("server returned error: %s", body)
	}

	return fmt.Sprintf("%s", string(body)), nil
}

// Get tries to fetch the given id from the dumpinen server.
func (d *dumpinen) Get(id string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", d.addr, id), nil)
	if err != nil {
		return nil, err
	}

	if d.username != "" && d.password != "" {
		req.SetBasicAuth(d.username, d.password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	return body, nil
}

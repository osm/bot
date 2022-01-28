package pastebin

import (
	"bytes"
	"net/http"
	"net/url"
)

type Visibility string

const (
	Public   Visibility = "0"
	Unlisted            = "1"
)

type ExpireDate string

const (
	Never      ExpireDate = "N"
	OneHour               = "1H"
	TenMinutes            = "10M"
)

type pastebin struct {
	apiKey string
}

func New(apiKey string) *pastebin {
	return &pastebin{apiKey}
}

func (pb *pastebin) NewPaste(code, name string, visibility Visibility, expireDate ExpireDate) (string, error) {
	values := url.Values{}
	values.Set("api_dev_key", pb.apiKey)
	values.Set("api_option", "paste")
	values.Set("api_paste_code", code)
	values.Set("api_paste_expire_date", string(expireDate))
	values.Set("api_paste_name", name)
	values.Set("api_paste_private", string(visibility))

	var err error
	var res *http.Response
	if res, err = http.PostForm("https://pastebin.com/api/api_post.php", values); err != nil {
		return "", err
	}
	defer res.Body.Close()

	var buf bytes.Buffer
	buf = bytes.Buffer{}
	if _, err = buf.ReadFrom(res.Body); err != nil {
		return "", err
	}

	return buf.String(), nil
}

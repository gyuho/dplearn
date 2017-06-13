// Package urlutil implements URL utilities.
package urlutil

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	humanize "github.com/dustin/go-humanize"
)

// TrimQuery removes query string from URL.
func TrimQuery(ep string) string {
	u, err := url.Parse(strings.TrimSpace(ep))
	if err != nil {
		return ep
	}
	raw := strings.TrimSpace(u.String())
	if u.RawQuery != "" {
		raw = strings.Replace(raw, "?"+u.RawQuery, "", -1)
	}
	return raw
}

// GetContentLength fetches the file size of the content.
func GetContentLength(ep string) (uint64, string, error) {
	resp, err := http.Head(ep)
	if err != nil {
		return 0, "", err
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	return uint64(resp.ContentLength), humanize.Bytes(uint64(resp.ContentLength)), nil
}

// Get downloads the URL contents.
func Get(ep string) ([]byte, error) {
	resp, err := http.Get(ep)
	if err != nil {
		return nil, err
	}

	var data []byte
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	return data, nil
}

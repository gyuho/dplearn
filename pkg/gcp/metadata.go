package gcp

import (
	"io/ioutil"
	"net/http"
	"path"
	"strings"
)

// GetComputeMetadata fetches the metadata from instance or project.
// e.g. curl -L http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip -H 'Metadata-Flavor:Google'
func GetComputeMetadata(key string) (string, error) {
	const endpoint = "http://metadata.google.internal/computeMetadata/v1/"
	ep := path.Join(endpoint, key)

	req, err := http.NewRequest("GET", ep, nil)
	if err != nil {
		return "", err
	}
	req.Header = map[string][]string{"Metadata-Flavor": []string{"Google"}}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	bts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	return strings.TrimSpace(string(bts)), nil
}

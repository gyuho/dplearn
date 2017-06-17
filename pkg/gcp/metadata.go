package gcp

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
)

// GetComputeMetadata fetches the metadata from instance or project.
// e.g. curl -L http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip -H 'Metadata-Flavor:Google'
func GetComputeMetadata(key string, try int, interval time.Duration) ([]byte, error) {
	const endpoint = "http://metadata.google.internal/computeMetadata/v1/"
	if strings.HasPrefix(key, "/") {
		key = key[1:]
	}
	ep := endpoint + key

	req, err := http.NewRequest("GET", ep, nil)
	if err != nil {
		return nil, err
	}
	req.Header = map[string][]string{"Metadata-Flavor": {"Google"}}

	for i := 0; i < try; i++ {
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			glog.Warning(err)
			time.Sleep(interval)
			continue
		}
		var data []byte
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			glog.Warning(err)
			time.Sleep(interval)
			continue
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		return data, nil
	}

	return nil, fmt.Errorf("could not fetch %q (%v)", key, err)
}

package web

import (
	"fmt"
	"net/http"
	"testing"
)

/*
go test -v -run TestID -logtostderr=true
*/

func TestID(t *testing.T) {
	req := &http.Request{
		Header: map[string][]string{
			"X-Forwarded-For": {"127.0.0.1"},
			"User-Agent":      {"linux chrome/"},
		},
	}
	fmt.Println(generateUserID(req))
}

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
			"X-Forwarded-For": []string{"127.0.0.1"},
			"User-Agent":      []string{"linux chrome/"},
		},
	}
	fmt.Println(generateUserID(req))
}

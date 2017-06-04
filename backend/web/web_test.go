package web

import (
	"testing"
	"time"
)

/*
https://godoc.org/github.com/golang/glog
go test -v -logtostderr=true
*/

func TestServer(t *testing.T) {
	srv, err := StartServer(0)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)
	srv.Stop()

	select {
	case <-srv.StopNotify():
	case <-time.After(3 * time.Second):
		t.Fatal("took too long to shut down")
	}
}

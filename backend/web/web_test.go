package web

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

/*
go test -v -run TestServer -logtostderr=true
*/

func TestServer(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "etcd-queue")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	srv, err := StartServer(0, 5555, dataDir)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)

	if err = srv.Stop(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-srv.StopNotify():
	case <-time.After(3 * time.Second):
		t.Fatal("took too long to shut down")
	}
}

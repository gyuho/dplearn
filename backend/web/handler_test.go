package web

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	etcdqueue "github.com/gyuho/deephardway/pkg/etcd-queue"
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

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	qu, err := etcdqueue.NewEmbeddedQueue(rootCtx, 5555, 5556, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer qu.Stop()

	srv, err := StartServer(0, qu)
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

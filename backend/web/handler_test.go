package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	etcdqueue "github.com/gyuho/deephardway/pkg/etcd-queue"

	"github.com/golang/glog"
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

	srv, err := StartServer(42200, qu)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	glog.Info("test post on client request endpoint")
	var resp *http.Response
	resp, err = http.Post(
		srv.webURL.String()+"/word-predict-request",
		"application/json",
		strings.NewReader(`{"data_from_frontend": "hello world!", "create_request": true}`))
	if err != nil {
		t.Fatal(err)
	}
	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	var item etcdqueue.Item
	if err = json.Unmarshal(rb, &item); err != nil {
		t.Fatal(err)
	}
	if item.Error != "" {
		t.Fatalf("got non-empty error: %+v", item)
	}
	if item.RequestID == "" {
		t.Fatalf("got empty request ID: %+v", item)
	}
	if item.Key == "" {
		t.Fatalf("got empty key: %+v", item)
	}
	if item.Value == "" || !strings.Contains(item.Value, "hello world!") || !strings.HasPrefix(item.Value, "[BACKEND - ACK]") {
		t.Fatalf("got non-expected value: %+v", item)
	}
	if item.Progress != 0 {
		t.Fatalf("got unexpected progress: %+v", item)
	}

	time.Sleep(3 * time.Second)

	glog.Info("test fetch on queue endpoint; blocks until there is at least one item")
	resp, err = http.Get(srv.webURL.String() + "/word-predict-request/queue")
	if err != nil {
		t.Fatal(err)
	}
	rb, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	var itemFromQueueFetch etcdqueue.Item
	if err = json.Unmarshal(rb, &itemFromQueueFetch); err != nil {
		t.Fatal(err)
	}
	if itemFromQueueFetch.Bucket != item.Bucket {
		t.Fatalf("unexpected Bucket (%+v), expected %+v", itemFromQueueFetch, item)
	}
	if itemFromQueueFetch.Key != item.Key {
		t.Fatalf("unexpected Key (%+v), expected %+v", itemFromQueueFetch, item)
	}
	if itemFromQueueFetch.RequestID != item.RequestID {
		t.Fatalf("unexpected RequestID (%+v), expected %+v", itemFromQueueFetch, item)
	}
	if itemFromQueueFetch.Value != "hello world!" {
		t.Fatalf("unexpected Value (%+v), expected %+v", itemFromQueueFetch, item)
	}

	time.Sleep(3 * time.Second)

	glog.Info("test post on queue endpoint; simulate worker")
	itemDone := itemFromQueueFetch
	itemDone.Progress = 100
	itemDone.Value = "done!"
	rb, err = json.Marshal(itemDone)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.Post(
		srv.webURL.String()+"/word-predict-request/queue",
		"application/json",
		bytes.NewReader(rb))
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	var itemFromQueuePost etcdqueue.Item
	if err = json.Unmarshal(rb, &itemFromQueuePost); err != nil {
		t.Fatal(err)
	}
	if itemDone.Equal(&itemFromQueuePost) != nil {
		t.Fatalf("item expected %+v, got %+v", itemDone, itemFromQueuePost)
	}

	time.Sleep(3 * time.Second)

	glog.Info("test fetch on client endpoint; blocks until the job is done")
	u, uerr := url.Parse(srv.webURL.String() + "/word-predict-request")
	if uerr != nil {
		t.Fatal(err)
	}
	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
		Header: map[string][]string{"request_id": {itemDone.RequestID}},
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	rb, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	var itemFromClientRequestFetch etcdqueue.Item
	if err = json.Unmarshal(rb, &itemFromClientRequestFetch); err != nil {
		t.Fatal(err)
	}
	if itemDone.Equal(&itemFromClientRequestFetch) != nil {
		t.Fatalf("item expected %+v, got %+v", itemDone, itemFromClientRequestFetch)
	}

	glog.Info("test stoppping server")
	if err = srv.Stop(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-srv.StopNotify():
	case <-time.After(3 * time.Second):
		t.Fatal("took too long to shut down")
	}
}

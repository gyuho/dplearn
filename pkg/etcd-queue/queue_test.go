package etcdqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/golang/glog"
)

/*
go test -v -run TestQueue -logtostderr=true
*/

var basePort int32 = 22379

func TestQueue(t *testing.T) {
	cport := int(atomic.LoadInt32(&basePort))
	atomic.StoreInt32(&basePort, int32(cport)+2)

	dataDir, err := ioutil.TempDir(os.TempDir(), "etcd-queue")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	qu, err := NewEmbeddedQueue(cport, cport+1, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer qu.Stop()

	var cli *clientv3.Client
	cli, err = clientv3.New(clientv3.Config{Endpoints: qu.ClientEndpoints()})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	if _, err = cli.Put(context.Background(), "foo", "bar"); err != nil {
		t.Fatal(err)
	}
	if _, err = qu.Client().Put(context.Background(), "foo", "bar"); err != nil {
		t.Fatal(err)
	}

	var resp *clientv3.GetResponse
	resp, err = cli.Get(context.Background(), "foo")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Kvs) != 1 {
		t.Fatalf("len(resp.Kvs) expected 1, got %d", len(resp.Kvs))
	}
	if !bytes.Equal(resp.Kvs[0].Value, []byte("bar")) {
		t.Fatalf("value expected 'bar', got %q", string(resp.Kvs[0].Value))
	}

	item1 := CreateItem("my-job", 1500, "my text goes here... 1")
	wch1, err := qu.Add(context.Background(), item1)
	if err != nil {
		t.Fatal(err)
	}
	item2 := CreateItem("my-job", 15000, "my text goes here... 2")
	wch2, err := qu.Add(context.Background(), item2)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)

	// expects 'item1' to be scheduled
	todoKey := path.Join(pfxWorker, "my-job")
	resp, err = cli.Get(context.Background(), todoKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Kvs) != 1 {
		t.Fatalf("%q should have 1 key-value, got %+v", todoKey, resp.Kvs)
	}
	var rt Item
	if err = json.Unmarshal(resp.Kvs[0].Value, &rt); err != nil {
		t.Fatal(err)
	}
	if item1.Value != rt.Value {
		t.Fatalf("rt.Value expected %s, got %s", string(item1.Value), rt.Value)
	}

	// simulate job event on item1
	item1.Progress = 100
	item1.Value = "finished!"
	item1Marshaled, err := json.Marshal(item1)
	if err != nil {
		t.Fatal(err)
	}
	glog.Infof("writing to %q", todoKey)
	if _, err = cli.Put(context.Background(), todoKey, string(item1Marshaled)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	// expects events from wch1
	select {
	case updatedItem := <-wch1:
		var ui Item
		if err = json.Unmarshal([]byte(updatedItem.Value), &ui); err != nil {
			t.Fatal(err)
		}
		if item1.Value != ui.Value {
			t.Fatalf("item1.Value from watch expected %+v, got %+v", item1, ui)
		}
		var gresp *clientv3.GetResponse
		gresp, err = cli.Get(context.Background(), path.Join(pfxCompleted, updatedItem.Key))
		if err != nil {
			t.Fatal(err)
		}
		var item Item
		if err = json.Unmarshal(gresp.Kvs[0].Value, &item); err != nil {
			t.Fatal(err)
		}
		if item1.Value != item.Value {
			t.Fatalf("item1.Value from 'completed' bucket expected %+v, got %+v", item1, item)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("took too long to receive event on item1 %s", item1.Key)
	}
	select {
	case deletedItem, ok := <-wch1:
		if !ok {
			t.Fatal("should not be closed before receiving delete event")
		}
		if _, ok = <-wch1; ok {
			t.Fatal("must be closed after delete event")
		}
		var gresp *clientv3.GetResponse
		gresp, err = cli.Get(context.Background(), path.Join(pfxCompleted, deletedItem.Key))
		if err != nil {
			t.Fatal(err)
		}
		var di Item
		if err = json.Unmarshal(gresp.Kvs[0].Value, &di); err != nil {
			t.Fatal(err)
		}
		if item1.Value != di.Value {
			t.Fatalf("item1.Value from 'completed' bucket expected %+v, got %+v", item1, di)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("took too long to receive event on item1 %s", item1.Key)
	}

	// simulate job event on item2
	item2.Progress = 100
	item2.Value = "finished!"
	item2ValBytes, err := json.Marshal(item2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cli.Put(context.Background(), todoKey, string(item2ValBytes)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	// expects events from wch2
	select {
	case updatedItem := <-wch2:
		var ui Item
		if err = json.Unmarshal([]byte(updatedItem.Value), &ui); err != nil {
			t.Fatal(err)
		}
		if item2.Value != ui.Value {
			t.Fatalf("item2.Value from watch expected %+v, got %+v", item2, ui)
		}
		var gresp *clientv3.GetResponse
		gresp, err = cli.Get(context.Background(), path.Join(pfxCompleted, updatedItem.Key))
		if err != nil {
			t.Fatal(err)
		}
		var item Item
		if err = json.Unmarshal(gresp.Kvs[0].Value, &item); err != nil {
			t.Fatal(err)
		}
		if item2.Value != item.Value {
			t.Fatalf("item2.Value from 'completed' bucket expected %+v, got %+v", item2, item)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("took too long to receive event on item2 %s", item2.Key)
	}
	select {
	case deletedItem, ok := <-wch2:
		if !ok {
			t.Fatal("should not be closed before receiving delete event")
		}
		if _, ok = <-wch2; ok {
			t.Fatal("must be closed after delete event")
		}
		var gresp *clientv3.GetResponse
		gresp, err = cli.Get(context.Background(), path.Join(pfxCompleted, deletedItem.Key))
		if err != nil {
			t.Fatal(err)
		}
		var di Item
		if err = json.Unmarshal(gresp.Kvs[0].Value, &di); err != nil {
			t.Fatal(err)
		}
		if item2.Value != di.Value {
			t.Fatalf("item2.Value from 'completed' bucket expected %+v, got %+v", item2, di)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("took too long to receive event on item2 %s", item2.Key)
	}
}

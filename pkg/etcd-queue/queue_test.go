package etcdqueue

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
)

/*
go test -v -run TestQueue -logtostderr=true
*/

var basePort int32 = 22379

func TestQueue(t *testing.T) {
	cport := int(atomic.LoadInt32(&basePort))
	atomic.StoreInt32(&basePort, int32(cport)+2)
	testBucket := "test-bucket"

	dataDir, err := ioutil.TempDir(os.TempDir(), "etcd-queue")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	qu, err := NewEmbeddedQueue(context.Background(), cport, cport+1, dataDir)
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

	item1 := CreateItem(testBucket, 1500, "test-data-1")
	wch1, err := qu.Enqueue(context.Background(), item1)
	if err != nil {
		t.Fatal(err)
	}
	item2 := CreateItem(testBucket, 15000, "test-data-2")
	wch2, err := qu.Enqueue(context.Background(), item2)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)

	// first element in the queue must be item2 with higher priority
	item2a, err := qu.Front(context.Background(), testBucket)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(item2, item2a) {
		t.Fatalf("front expected %+v, got %+v", item2, item2a)
	}

	select {
	case ev := <-wch1:
		t.Fatalf("unexpected event from wch1 %+v", ev)
	case ev := <-wch2:
		t.Fatalf("unexpected event from wch2 %+v", ev)
	default:
	}

	// finish 'item2'
	item2a.Progress = 100
	item2a.Value = "new-data"
	wch2a, err := qu.Enqueue(context.Background(), item2a)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case item2b := <-wch2a:
		if !reflect.DeepEqual(item2a, item2b) {
			t.Fatalf("wch2a expected %+v, got %+v", item2a, item2b)
		}
	default:
		t.Fatalf("expected events from qu.Enqueue(item3)")
	}
	select {
	case item2c := <-wch2:
		if !reflect.DeepEqual(item2a, item2c) {
			t.Fatalf("wch2 expected %+v, got %+v", item2a, item2c)
		}
	default:
		t.Fatalf("expected events from wch2")
	}
	resp, err := cli.Get(context.Background(), path.Join(pfxCompleted, item2.Key))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Kvs) != 1 {
		t.Fatalf("len(resp.Kvs) expected 1, got %+v", resp.Kvs)
	}
	var item2d Item
	if err := json.Unmarshal(resp.Kvs[0].Value, &item2d); err != nil {
		t.Fatalf("cannot parse %q (%v)", string(resp.Kvs[0].Value), err)
	}
	if !reflect.DeepEqual(*item2a, item2d) {
		t.Fatalf("item2d expected %+v, got %+v", *item2a, item2d)
	}
	// if finished, the channel must be closed
	if v, ok := <-wch2; ok {
		t.Fatalf("unexpected event from wch2, got %+v", v)
	}
	if v, ok := <-wch2a; ok {
		t.Fatalf("unexpected event from wch2a, got %+v", v)
	}

	// next item in the queue must be item1
	item1a, err := qu.Front(context.Background(), testBucket)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(item1, item1a) {
		t.Fatalf("front expected %+v, got %+v", item1, item1a)
	}

	// proceed 'item1'
	item1a.Progress = 50
	item1a.Value = "new-data"
	wch1a, err := qu.Enqueue(context.Background(), item1a)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case it := <-wch1a:
		t.Fatalf("unexpected events from wch1a %+v", it)
	default:
	}
	select {
	case item1c := <-wch1:
		if !reflect.DeepEqual(item1a, item1c) {
			t.Fatalf("wch1 expected %+v, got %+v", item1a, item1c)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("expected events from wch1 in 5-sec")
	}

	// cancel 'item1'
	if err = qu.Dequeue(context.Background(), item1a); err != nil {
		t.Fatal(err)
	}
	select {
	case it := <-wch1:
		if it.Canceled != true {
			t.Fatalf("%q expected cancel, got %+v", it.Key, it)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("expected events from wch1 in 5-sec")
	}
	select {
	case it := <-wch1a:
		if it.Canceled != true {
			t.Fatalf("%q expected cancel, got %+v", it.Key, it)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("expected events from wch1a in 5-sec")
	}
	// if canceled, the channel must be closed
	if v, ok := <-wch1; ok {
		t.Fatalf("unexpected event from wch1, got %+v", v)
	}
	if v, ok := <-wch1a; ok {
		t.Fatalf("unexpected event from wch1a, got %+v", v)
	}
}

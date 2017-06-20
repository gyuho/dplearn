package etcdqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
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

	testBucket := "test-bucket"

	firstCreate := qu.Front(context.Background(), testBucket)
	select {
	case fi := <-firstCreate:
		t.Fatalf("unexpected events: %+v", fi)
	default:
	}

	item1 := CreateItem(testBucket, 1500, "test-data-1")
	wch1 := qu.Enqueue(context.Background(), item1)
	item2 := CreateItem(testBucket, 15000, "test-data-2")
	wch2 := qu.Enqueue(context.Background(), item2)

	time.Sleep(3 * time.Second)

	select {
	case fi := <-firstCreate:
		if err = equalItem(item1, fi); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item1, fi, err)
		}
	default:
		t.Fatalf("expected events, but got none")
	}

	// first element in the queue must be item2 with higher priority
	fch2a := qu.Front(context.Background(), testBucket)
	if err != nil {
		t.Fatal(err)
	}
	item2a := <-fch2a
	if err = equalItem(item2, item2a); err != nil {
		t.Fatalf("expected %+v, got %+v (%v)", item2, item2a, err)
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
	wch2a := qu.Enqueue(context.Background(), item2a)
	select {
	case item2b := <-wch2a:
		if item2b.Error != "" {
			t.Fatalf("unexpected error: %+v", item2b)
		}
		if err = equalItem(item2a, item2b); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item2, item2b, err)
		}
	default:
		t.Fatalf("expected events from qu.Enqueue(item3)")
	}

	select {
	case item2c := <-wch2:
		if err = equalItem(item2a, item2c); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item2, item2c, err)
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
	if err = equalItem(item2a, &item2d); err != nil {
		t.Fatalf("expected %+v, got %+v (%v)", item2a, item2d, err)
	}

	// if finished, the channel must be closed
	if v, ok := <-wch2; ok {
		t.Fatalf("unexpected event from wch2, got %+v", v)
	}
	if v, ok := <-wch2a; ok {
		t.Fatalf("unexpected event from wch2a, got %+v", v)
	}

	// next item in the queue must be item1
	fch1a := qu.Front(context.Background(), testBucket)
	item1a := <-fch1a
	if item1a.Error != "" {
		t.Fatalf("unexpected error: %+v", item1a)
	}
	if err = equalItem(item1, item1a); err != nil {
		t.Fatalf("expected %+v, got %+v (%v)", item1, item1a, err)
	}

	// proceed 'item1'
	item1a.Progress = 50
	item1a.Value = "new-data"
	wch1a := qu.Enqueue(context.Background(), item1a)
	select {
	case it := <-wch1a:
		t.Fatalf("unexpected events from wch1a %+v", it)
	default:
	}
	select {
	case item1c := <-wch1:
		if item1c.Error != "" {
			t.Fatalf("unexpected error: %+v", item1c)
		}
		if err = equalItem(item1a, item1c); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item1a, item1c, err)
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
		if it.Error != "" {
			t.Fatalf("unexpected error: %+v", it)
		}
		if it.Canceled != true {
			t.Fatalf("%q expected cancel, got %+v", it.Key, it)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("expected events from wch1 in 5-sec")
	}
	select {
	case it := <-wch1a:
		if it.Error != "" {
			t.Fatalf("unexpected error: %+v", it)
		}
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

// truncate CreatedAt to handle added timestamp texts while serialization
func equalItem(item1, item2 *Item) error {
	if item1.CreatedAt.String()[:29] != item2.CreatedAt.String()[:29] {
		return fmt.Errorf("expected CreatedAt %q, got %q", item1.CreatedAt.String()[:29], item2.CreatedAt.String()[:29])
	}
	if item1.Bucket != item2.Bucket {
		return fmt.Errorf("expected Bucket %q, got %q", item1.Bucket, item2.Bucket)
	}
	if item1.Key != item2.Key {
		return fmt.Errorf("expected Key %q, got %q", item1.Key, item2.Key)
	}
	if item1.Value != item2.Value {
		return fmt.Errorf("expected Value %q, got %q", item1.Value, item2.Value)
	}
	if item1.Progress != item2.Progress {
		return fmt.Errorf("expected Progress %d, got %d", item1.Progress, item2.Progress)
	}
	if item1.Canceled != item2.Canceled {
		return fmt.Errorf("expected Canceled %v, got %v", item1.Canceled, item2.Canceled)
	}
	if item1.Error != item2.Error {
		return fmt.Errorf("expected Error %s, got %s", item1.Error, item2.Error)
	}
	if item1.RequestID != item2.RequestID {
		return fmt.Errorf("expected RequestID %s, got %s", item1.RequestID, item2.RequestID)
	}
	return nil
}

// TestQueueEtcd tests some etcd-specific behaviors.
func TestQueueEtcd(t *testing.T) {
	cport := int(atomic.LoadInt32(&basePort))
	atomic.StoreInt32(&basePort, int32(cport)+2)

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

	resp, err := cli.Get(context.Background(), "\x00", clientv3.WithFromKey())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Kvs) != 0 {
		t.Fatalf("len(resp.Kvs) expected 0, got %+v", resp.Kvs)
	}

	watchChan := cli.Watch(context.Background(), "foo", clientv3.WithPrefix())
	donec := make(chan struct{})
	go func() {
		defer close(donec)

		wresp := <-watchChan
		if len(wresp.Events) != 1 {
			t.Fatalf("len(wresp.Events) expected 1, got %+v", wresp.Events)
		}
		if !bytes.Equal(wresp.Events[0].Kv.Key, []byte("foo")) {
			t.Fatalf("key expected 'foo', got %q", string(wresp.Events[0].Kv.Key))
		}
		if !bytes.Equal(wresp.Events[0].Kv.Value, []byte("bar")) {
			t.Fatalf("value expected 'bar', got %q", string(wresp.Events[0].Kv.Value))
		}
	}()

	if _, err = cli.Put(context.Background(), "foo", "bar"); err != nil {
		t.Fatal(err)
	}
	resp, err = cli.Get(context.Background(), "foo")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Kvs) != 1 {
		t.Fatalf("len(resp.Kvs) expected 1, got %+v", resp.Kvs)
	}
	fmt.Printf("Get response: %+v\n", resp)

	ch := cli.Watch(context.Background(), "foo", clientv3.WithRev(resp.Header.Revision))
	select {
	case wresp := <-ch:
		fmt.Printf("Watch response: %+v\n", wresp.Events[0])
	case <-time.After(2 * time.Second):
		t.Fatal("watch timed out")
	}
	ch = cli.Watch(context.Background(), "foo")
	select {
	case wresp := <-ch:
		t.Fatalf("unexpected watch response: %+v", wresp)
	case <-time.After(3 * time.Second):
	}

	<-donec

	if _, err = cli.Put(context.Background(), "foo1", "bar1"); err != nil {
		t.Fatal(err)
	}
	resp, err = cli.Get(context.Background(), "\x00", clientv3.WithFromKey())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Kvs) != 2 {
		t.Fatalf("len(resp.Kvs) expected 2, got %+v", resp.Kvs)
	}
}

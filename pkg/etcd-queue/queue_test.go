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
	"github.com/golang/glog"
)

/*
go test -v -run TestQueue -logtostderr=true
*/

var basePort int32 = 22379

func TestQueueEnqueueFront(t *testing.T) {
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

	testBucket := "test-bucket"

	frontChanFirstCreate := qu.Front(context.Background(), testBucket)
	select {
	case item := <-frontChanFirstCreate:
		t.Fatalf("unexpected events: %+v", item)
	default:
	}

	item1 := CreateItem(testBucket, 1000, "test-data")
	item1EnqueueWatcher := qu.Enqueue(context.Background(), item1)

	item2 := CreateItem(testBucket, 9000, "test-data-2")
	item2EnqueuWatcher := qu.Enqueue(context.Background(), item2)

	time.Sleep(3 * time.Second)

	select {
	case item := <-frontChanFirstCreate:
		if err = item1.Equal(item); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item1, item, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected events, but got none")
	}

	// first element must be the one with higher weight
	frontChan := qu.Front(context.Background(), testBucket)
	if err != nil {
		t.Fatal(err)
	}
	var item2FromQueue *Item
	select {
	case item2FromQueue = <-frontChan:
		if item2FromQueue.Error != "" {
			t.Fatalf("unexpected error: %+v", item2FromQueue)
		}
		if err = item2.Equal(item2FromQueue); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item2, item2FromQueue, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected events, but got none")
	}

	select {
	case ev := <-item1EnqueueWatcher:
		t.Fatalf("unexpected event from item1EnqueueWatcher %+v", ev)
	case ev := <-item2EnqueuWatcher:
		t.Fatalf("unexpected event from item2EnqueuWatcher %+v", ev)
	default:
	}

	// simulate worker
	item2FromQueue.Progress = 100
	item2FromQueue.Value = "new-data"
	item2FromQueueEnqueueWatcher := qu.Enqueue(context.Background(), item2FromQueue)
	select {
	case item := <-item2FromQueueEnqueueWatcher:
		if item.Error != "" {
			t.Fatalf("unexpected error: %+v", item)
		}
		if err = item2FromQueue.Equal(item); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item2FromQueue, item, err)
		}
	default:
		t.Fatal("expected events from qu.Enqueue(item3)")
	}

	select {
	case item := <-item2EnqueuWatcher:
		if err = item2FromQueue.Equal(item); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item2FromQueue, item, err)
		}
	default:
		t.Fatal("expected events from item2EnqueuWatcher")
	}

	resp, err := qu.Client().Get(context.Background(), path.Join(pfxCompleted, item2.Key))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Kvs) != 1 {
		t.Fatalf("len(resp.Kvs) expected 1, got %+v", resp.Kvs)
	}
	var item Item
	if err := json.Unmarshal(resp.Kvs[0].Value, &item); err != nil {
		t.Fatalf("cannot parse %q (%v)", string(resp.Kvs[0].Value), err)
	}
	if err = item2FromQueue.Equal(&item); err != nil {
		t.Fatalf("expected %+v, got %+v (%v)", item2FromQueue, item, err)
	}

	// if finished, channel must be closed
	if v, stillOpen := <-item2EnqueuWatcher; stillOpen {
		t.Fatalf("unexpected event from item2EnqueuWatcher, got %+v", v)
	}
	if v, stillOpen := <-item2FromQueueEnqueueWatcher; stillOpen {
		t.Fatalf("unexpected event from item2FromQueueEnqueueWatcher, got %+v", v)
	}

	// next item in the queue must be item1
	frontChan = qu.Front(context.Background(), testBucket)
	var item1FromQueue *Item
	select {
	case item1FromQueue = <-frontChan:
		if item1FromQueue.Error != "" {
			t.Fatalf("unexpected error: %+v", item1FromQueue)
		}
		if err = item1.Equal(item1FromQueue); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item1, item1FromQueue, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected events, but got none")
	}
}

func TestQueueCancel(t *testing.T) {
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

	testBucket := "test-bucket"

	item1 := CreateItem(testBucket, 1000, "test-data")
	item1EnqueueWatcher := qu.Enqueue(context.Background(), item1)

	time.Sleep(3 * time.Second)

	// cancel 'item1' before finish
	if err = qu.Dequeue(context.Background(), item1); err != nil {
		t.Fatal(err)
	}
	select {
	case item := <-item1EnqueueWatcher:
		if item.Error != "" {
			t.Fatalf("unexpected error: %+v", item)
		}
		if !item.Canceled {
			t.Fatalf("%q expected cancel, got %+v", item.Key, item)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("expected events from item1EnqueueWatcher in 5-sec")
	}

	// if canceled, the channel must be closed
	if v, ok := <-item1EnqueueWatcher; ok {
		t.Fatalf("unexpected event from item1EnqueueWatcher, got %+v", v)
	}
}

func TestQueueWatch(t *testing.T) {
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

	testBucket := "test-bucket"

	item1 := CreateItem(testBucket, 5000, "test-data")
	qu.Enqueue(context.Background(), item1)

	time.Sleep(3 * time.Second)

	// spawn watcher after item writes on the queue
	ctx, cancel := context.WithCancel(context.Background())
	item1Watcher := qu.Watch(ctx, item1.Key)

	// simulate worker to trigger watch event
	item1.Progress = 50
	item1.Value = "new-data"
	qu.Enqueue(context.Background(), item1)

	select {
	case item, stillOpen := <-item1Watcher:
		if !stillOpen {
			t.Fatalf("%q watcher must still be open, got stillOpen %v", item1.Key, stillOpen)
		}
		if err = item1.Equal(item); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item1, item, err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("expected watch response on %q watcher, but got none", item1.Key)
	}

	// cancel the watcher to exit watch routine
	cancel()
}

func TestQueueEnqueueLeaseExpire(t *testing.T) {
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

	testBucket := "test-bucket"

	item1 := CreateItem(testBucket, 1000, "test-data")
	item1EnqueueWatcher := qu.Enqueue(context.Background(), item1, WithTTL(7*time.Second))

	glog.Infof("wait until lease revoke")
	time.Sleep(10 * time.Second)

	select {
	case item := <-qu.Front(context.Background(), testBucket):
		t.Fatalf("unexpected item %+v after lease revoke", item)
	case <-time.After(2 * time.Second):
	}

	select {
	case item := <-item1EnqueueWatcher:
		if item.Error != "" {
			t.Fatalf("unexpected error: %+v", item)
		}
		if !item.Canceled {
			t.Fatalf("%q expected cancel, got %+v", item.Key, item)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("expected events from item1EnqueueWatcher in 5-sec")
	}
}

// TestEtcd tests some etcd-specific behaviors.
func TestEtcd(t *testing.T) {
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

	prevChan := cli.Watch(context.Background(), "foo", clientv3.WithPrevKV())
	select {
	case ev := <-prevChan:
		t.Fatalf("unexpected watch event: %+v", ev)
	case <-time.After(2 * time.Second):
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

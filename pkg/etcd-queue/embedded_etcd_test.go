package etcdqueue

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
)

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

	lresp, lerr := cli.Grant(context.Background(), 2)
	if lerr != nil {
		t.Fatal(err)
	}
	if _, err = cli.Put(context.Background(), "k", "v", clientv3.WithLease(lresp.ID)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	gresp, gerr := cli.Get(context.Background(), "k")
	if gerr != nil {
		t.Fatal(gerr)
	}
	if len(gresp.Kvs) != 0 {
		t.Fatalf("not revoked: %+v", resp.Kvs[0])
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

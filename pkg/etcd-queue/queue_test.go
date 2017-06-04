package etcdqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/golang/glog"
)

/*
go test -v -run TestItem -logtostderr=true
*/
func TestItem(t *testing.T) {
	item1, err := CreateItem("test-job", 1500, []byte("Hello World!"))
	if err != nil {
		t.Fatal(err)
	}
	glog.Infof("created %+v", item1)

	vd, err := json.Marshal(item1.Value)
	if err != nil {
		t.Fatal(err)
	}

	item2, err := ParseItem(item1.Key, vd)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(item1, item2) {
		t.Fatalf("expected %+v, got %+v", item1, item2)
	}

	item3, err := CreateItem("test-job", 1500, []byte("Hello World!"))
	if err != nil {
		t.Fatal(err)
	}
	glog.Infof("created %+v", item3)

	item4, err := CreateItem("test-job", 15000, []byte("Hello World!"))
	if err != nil {
		t.Fatal(err)
	}
	glog.Infof("created %+v", item4)

	items := []*Item{item1, item3, item4}
	itemsSorted := []*Item{item4, item1, item3}
	sort.Sort(Items(items))

	if !reflect.DeepEqual(items, itemsSorted) {
		t.Fatalf("expected %+v, got %+v", itemsSorted, items)
	}
}

var basePort int32 = 22379

/*
go test -v -run TestQueue -logtostderr=true
*/

func TestQueue(t *testing.T) {
	cport := int(atomic.LoadInt32(&basePort))
	atomic.StoreInt32(&basePort, int32(cport)+2)

	qu, err := StartQueue(cport, cport+1)
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
	if _, err = qu.cli.Put(context.Background(), "foo", "bar"); err != nil {
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

	item1, err := CreateItem("my-job", 1500, []byte("my text goes here... 1"))
	if err != nil {
		t.Fatal(err)
	}
	wch1, err := qu.Add(context.Background(), item1)
	if err != nil {
		t.Fatal(err)
	}
	item2, err := CreateItem("my-job", 15000, []byte("my text goes here... 2"))
	if err != nil {
		t.Fatal(err)
	}
	wch2, err := qu.Add(context.Background(), item2)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)

	// expects 'item1' to be scheduled
	todoKey := path.Join(pfxTODO, "my-job")
	resp, err = cli.Get(context.Background(), todoKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Kvs) != 1 {
		t.Fatalf("%q should have 1 key-value, got %+v", todoKey, resp.Kvs)
	}
	item1Bts, err := json.Marshal(item1.Value)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(item1Bts, resp.Kvs[0].Value) {
		t.Fatalf("resp.Kvs[0].Value expected %s, got %s", string(item1Bts), resp.Kvs[0].Value)
	}

	// simulate job event on item1
	item1.Value.StatusCode = StatusCodeDone
	item1.Value.Data = []byte("finished!")
	item1ValBytes, err := json.Marshal(item1.Value)
	if err != nil {
		t.Fatal(err)
	}
	glog.Infof("writing to %q", todoKey)
	if _, err = cli.Put(context.Background(), todoKey, string(item1ValBytes)); err != nil {
		t.Fatal(err)
	}

	// expects events from wch1
	select {
	case wresp := <-wch1:
		v := wresp.Events[0].Kv.Value
		var val Value
		if err = json.Unmarshal(v, &val); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(item1.Value, val) {
			t.Fatalf("item1.Value from watch expected %+v, got %+v", item1.Value, val)
		}
		gresp, err := cli.Get(context.Background(), strings.Replace(val.Key, pfxScheduled, pfxCompleted, 1))
		if err != nil {
			t.Fatal(err)
		}
		v = gresp.Kvs[0].Value
		if err = json.Unmarshal(v, &val); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(item1.Value, val) {
			t.Fatalf("item1.Value from 'completed' bucket expected %+v, got %+v", item1.Value, val)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("took too long to receive event on item1 %s", item1.Key)
	}

	// simulate job event on item2
	item2.Value.StatusCode = StatusCodeDone
	item2.Value.Data = []byte("finished!")
	item2ValBytes, err := json.Marshal(item2.Value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cli.Put(context.Background(), todoKey, string(item2ValBytes)); err != nil {
		t.Fatal(err)
	}

	// expects events from wch2
	select {
	case wresp := <-wch2:
		v := wresp.Events[0].Kv.Value
		var val Value
		if err = json.Unmarshal(v, &val); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(item2.Value, val) {
			t.Fatalf("item2.Value from watch expected %+v, got %+v", item2.Value, val)
		}
		gresp, err := cli.Get(context.Background(), strings.Replace(val.Key, pfxScheduled, pfxCompleted, 1))
		if err != nil {
			t.Fatal(err)
		}
		v = gresp.Kvs[0].Value
		if err = json.Unmarshal(v, &val); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(item2.Value, val) {
			t.Fatalf("item2.Value from 'completed' bucket expected %+v, got %+v", item2.Value, val)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("took too long to receive event on item2 %s", item2.Key)
	}
}

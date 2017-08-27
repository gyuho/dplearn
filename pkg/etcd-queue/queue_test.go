package etcdqueue

import (
	"context"
	"io/ioutil"
	"os"
	"sync/atomic"
	"testing"
	"time"
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

	testBucket := "test-bucket"

	popCh1 := qu.Pop(context.Background(), testBucket)
	select {
	case item := <-popCh1:
		t.Fatalf("unexpected events: %+v", item)
	default:
	}

	item1 := CreateItem(testBucket, 1000, "test-data-1")
	item2 := CreateItem(testBucket, 9000, "test-data-2")
	if err = qu.Add(context.Background(), item1); err != nil {
		t.Fatal(err)
	}
	if err = qu.Add(context.Background(), item2); err != nil {
		t.Fatal(err)
	}

	select {
	case item := <-popCh1:
		if item.Error != "" {
			t.Fatalf("unexpected error: %+v", item)
		}
		if err = item1.Equal(item); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item1, item, err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected events, but got none")
	}

	popCh2 := qu.Pop(context.Background(), testBucket)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case item := <-popCh2:
		if item.Error != "" {
			t.Fatalf("unexpected error: %+v", item)
		}
		if err = item2.Equal(item); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item2, item, err)
		}
	default:
		t.Fatal("expected events, but got none")
	}

	item3 := CreateItem(testBucket, 1000, "test-data-1")
	item4 := CreateItem(testBucket, 9000, "test-data-2")
	if err = qu.Add(context.Background(), item3); err != nil {
		t.Fatal(err)
	}
	if err = qu.Add(context.Background(), item4); err != nil {
		t.Fatal(err)
	}
	popCh3 := qu.Pop(context.Background(), testBucket)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case item := <-popCh3:
		if item.Error != "" {
			t.Fatalf("unexpected error: %+v", item)
		}
		if err = item4.Equal(item); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item1, item, err)
		}
	default:
		t.Fatal("expected events, but got none")
	}
	popCh4 := qu.Pop(context.Background(), testBucket)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case item := <-popCh4:
		if item.Error != "" {
			t.Fatalf("unexpected error: %+v", item)
		}
		if err = item3.Equal(item); err != nil {
			t.Fatalf("expected %+v, got %+v (%v)", item1, item, err)
		}
	default:
		t.Fatal("expected events, but got none")
	}

	item5 := CreateItem(testBucket, 1000, "test-data")
	if err = qu.Add(context.Background(), item5, WithTTL(7*time.Second)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Second)
	popCh5 := qu.Pop(context.Background(), testBucket)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case item := <-popCh5:
		t.Fatalf("unexpected item %+v", item)
	default:
	}
}

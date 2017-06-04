package queue

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/glog"
)

/*
go test -v -run TestQueue --logtostderr=true
*/

func TestQueue(t *testing.T) {
	dir, err := ioutil.TempDir(".", "queue-on-disk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	glog.Infof("created %q", dir)

	var q *Queue
	q, err = NewQueue(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Shutdown()

	job1 := q.Schedule(randTxt(100))
	go func() {
		time.Sleep(2 * time.Second)
		glog.Infof("writing to %q", q.todo.path)
		if err = toFile(randTxt(500), q.todo.path); err != nil {
			t.Fatal(err)
		}
		glog.Infof("wrote to %q", q.todo.path)
	}()
	select {
	case <-job1.Notify():
	case <-time.After(10 * time.Second):
		t.Fatalf("%q did not finish in time", job1.path)
	}
	glog.Infof("%q is successfully finished", job1.path)

	if err = q.Remove(job1); err != nil {
		t.Fatal(err)
	}
}

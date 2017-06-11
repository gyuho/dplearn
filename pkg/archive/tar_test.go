package archive

import (
	"os"
	"testing"

	"github.com/golang/glog"
)

/*
go test -v -run TestTarGz -logtostderr=true
tar -xvzf ./test-etcd.tar.gz
*/
func TestTarGz(t *testing.T) {
	os.RemoveAll("test-etcd.tar.gz")
	defer os.RemoveAll("test-etcd.tar.gz")

	if err := TarGz("test-etcd.data", "test-etcd.tar.gz"); err != nil {
		t.Fatal(err)
	}
	glog.Info("success!")
}

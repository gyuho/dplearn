package gcp

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	compute "google.golang.org/api/compute/v1"
)

/*
go test -v -run TestComputeUbuntu -logtostderr=true
go test -v -run TestComputeContainerLinux -logtostderr=true
*/
func TestComputeUbuntu(t *testing.T)         { testCompute(t, "ubuntu", false) }
func TestComputeContainerLinux(t *testing.T) { testCompute(t, "container-linux", false) }
func testCompute(t *testing.T, osType string, skip bool) {
	testKeyPath := os.Getenv("GCP_TEST_KEY_PATH")
	if testKeyPath == "" {
		t.Skip("GCP_TEST_KEY_PATH is not set; skipping")
	}

	testKey, err := ioutil.ReadFile(testKeyPath)
	if err != nil {
		t.Skipf("%v on %q", err, testKeyPath)
	}

	api, err := NewCompute(context.Background(), compute.ComputeScope, testKey)
	if err != nil {
		t.Fatal(err)
	}

	instances, err := api.ListMachines(context.Background(), "us-west1-b")
	if err != nil {
		t.Fatal(err)
	}
	for i, it := range instances {
		m := ConvertToMachine(*it)
		glog.Infof("[%2d] %+v", i, m)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	_, err = api.ListMachines(ctx, "us-west1-b")
	cancel()
	if err != context.DeadlineExceeded {
		t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
	}

	instanceName := "gcp-test-" + strings.ToLower(randTxt(3))
	glog.Infof("starting to create %q", instanceName)

	metadataItems := make(map[string]string)
	metadataItems["gcp-key"] = string(testKey)

	switch osType {
	case "ubuntu":
		metadataItems["startup-script"] = `#!/usr/bin/env bash
set -e

echo "root ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
apt-get -y --allow-unauthenticated install ansible`
	case "container-linux":
	}

	cfg := InstanceConfig{
		Zone:              "us-west1-b",
		Name:              instanceName,
		OS:                osType,
		CPU:               8,
		Memory:            30,
		DiskSizeGB:        150,
		OnHostMaintenance: "TERMINATE",
		Tags:              []string{"gcp-test-tag"},
		MetadataItems:     metadataItems,
	}
	st1, err1 := api.CreateMacine(context.Background(), cfg)
	if err1 != nil {
		t.Skip(err1)
	}
	glog.Infof("created %+v", st1)

	if skip {
		t.Skip("skip after creating an instance")
	}

	instances, err = api.ListMachines(context.Background(), "us-west1-b")
	if err != nil {
		t.Fatal(err)
	}
	for i, it := range instances {
		m := ConvertToMachine(*it)
		glog.Infof("[%2d] %+v", i, m)
	}

	st2, err2 := api.StopMachine(context.Background(), cfg)
	if err2 != nil {
		t.Skip(err2)
	}
	glog.Infof("stopped %+v", st2)

	st3, err3 := api.StartMachine(context.Background(), cfg)
	if err3 != nil {
		t.Skip(err3)
	}
	glog.Infof("started %+v", st3)

	st4, err4 := api.StopMachine(context.Background(), cfg)
	if err4 != nil {
		t.Skip(err4)
	}
	glog.Infof("stopped %+v", st4)

	st5, err5 := api.DeleteMachine(context.Background(), cfg)
	if err5 != nil {
		t.Skip(err5)
	}
	glog.Infof("deleted %+v", st5)

	glog.Info("done!")
}

// exist returns true if the file or directory exists.
func exist(fpath string) bool {
	st, err := os.Stat(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	if st.IsDir() {
		return true
	}
	if _, err := os.Stat(fpath); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

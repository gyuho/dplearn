package main

import (
	"context"
	"flag"

	"github.com/gyuho/dplearn/backend/web"
	etcdqueue "github.com/gyuho/dplearn/pkg/etcd-queue"

	"github.com/golang/glog"
)

func main() {
	webPort := flag.Int("web-port", 2200, "Specify the port for web server backend.")
	queuePortClient := flag.Int("queue-port-client", 22000, "Specify the client port for queue service.")
	queuePortPeer := flag.Int("queue-port-peer", 22001, "Specify the peer port for queue service.")
	dataDir := flag.String("data-dir", "/var/lib/etcd", "Specify the etcd data directory.")
	flag.Parse()

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	qu, err := etcdqueue.NewEmbeddedQueue(rootCtx, *queuePortClient, *queuePortPeer, *dataDir)
	if err != nil {
		glog.Fatal(err)
	}
	defer qu.Stop()

	glog.Infof("starting web server with :%d (queue :%d/:%d, data-dir %q)", *webPort, *queuePortClient, *queuePortPeer, *dataDir)
	srv, err := web.StartServer(*webPort, qu)
	if err != nil {
		glog.Fatal(err)
	}

	select {
	case <-srv.StopNotify():
		glog.Warning("stopped web server")
	}
}

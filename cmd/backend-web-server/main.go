package main

import (
	"flag"

	"github.com/gyuho/deephardway/backend/web"

	"github.com/golang/glog"
)

func main() {
	webPort := flag.Int("web-port", 2200, "Specify the port for web server backend.")
	queuePort := flag.Int("queue-port", 22000, "Specify the port for queue service.")
	dataDir := flag.String("data-dir", "/var/lib/etcd", "Specify the etcd data directory.")
	flag.Parse()

	glog.Infof("starting web server with :%d (queue :%d, data-dir %q)", *webPort, *queuePort, *dataDir)
	srv, err := web.StartServer(*webPort, *queuePort, *dataDir)
	if err != nil {
		glog.Fatal(err)
	}

	select {
	case <-srv.StopNotify():
		glog.Warning("stopped web server")
	}
}

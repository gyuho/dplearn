package main

import (
	"flag"

	"github.com/gyuho/deephardway/backend/web"

	"github.com/golang/glog"
)

func init() {
	flag.Parse()
}

const (
	webPort   = 2200
	queuePort = 22000
)

func main() {
	glog.Info("starting web server")
	srv, err := web.StartServer(webPort, queuePort)
	if err != nil {
		glog.Fatal(err)
	}

	select {
	case <-srv.StopNotify():
		glog.Warning("stopped web server")
	}
}

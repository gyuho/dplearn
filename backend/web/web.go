package web

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	etcdqueue "github.com/gyuho/deephardway/pkg/etcd-queue"

	"github.com/golang/glog"
)

// Server warps http.Server.
type Server struct {
	mu         sync.RWMutex
	rootCtx    context.Context
	rootCancel func()
	addrURL    url.URL
	httpServer *http.Server
	qu         *etcdqueue.Queue

	donec chan struct{}
}

// StartServer starts a backend webserver with stoppable listener.
func StartServer(webPort, queuePort int) (*Server, error) {
	qu, err := etcdqueue.StartQueue(queuePort, queuePort+1)
	if err != nil {
		return nil, err
	}

	rootCtx, rootCancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()
	mux.Handle("/word-predict-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: ContextHandlerFunc(wordPredictHandler),
	})
	mux.Handle("/cats-and-dogs-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: ContextHandlerFunc(catsAndDogsHandler),
	})

	addrURL := url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", webPort)}
	srv := &Server{
		rootCtx:    rootCtx,
		rootCancel: rootCancel,
		addrURL:    addrURL,
		httpServer: &http.Server{Addr: addrURL.Host, Handler: mux},
		qu:         qu,
		donec:      make(chan struct{}),
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				glog.Fatal(err)
				os.Exit(0)
			}
			srv.rootCancel()
		}()

		glog.Infof("starting server %q", srv.addrURL.String())
		if err := srv.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			glog.Fatal(err)
		}

		select {
		case <-srv.donec:
		default:
			close(srv.donec)
		}
	}()
	return srv, nil
}

// Stop stops the server. Useful for testing.
func (srv *Server) Stop() error {
	glog.Infof("stopping server %q", srv.addrURL.String())

	srv.mu.Lock()
	srv.qu.Stop()

	if srv.httpServer == nil {
		srv.mu.Unlock()
		glog.Infof("already stopped %q", srv.addrURL.String())
		return nil
	}

	ctx, cancel := context.WithTimeout(srv.rootCtx, 5*time.Second)
	err := srv.httpServer.Shutdown(ctx)
	cancel()
	if err != nil && err != context.DeadlineExceeded {
		return err
	}

	srv.httpServer = nil
	srv.mu.Unlock()

	glog.Infof("stopped server %q", srv.addrURL.String())
	return nil
}

// StopNotify returns receive-only stop channel to notify the server has stopped.
func (srv *Server) StopNotify() <-chan struct{} {
	return srv.donec
}

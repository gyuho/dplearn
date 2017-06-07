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
	qu         etcdqueue.Queue

	donec chan struct{}
}

// StartServer starts a backend webserver with stoppable listener.
func StartServer(webPort, queuePort int, dataDir string) (*Server, error) {
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	qu, err := etcdqueue.NewEmbeddedQueue(rootCtx, queuePort, queuePort+1, dataDir)
	if err != nil {
		return nil, err
	}
	defer qu.Stop()

	mux := http.NewServeMux()
	mux.Handle("/word-predict-request-1", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(wordPredictHandler), qu),
	})
	mux.Handle("/word-predict-request-2", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(wordPredictHandler), qu),
	})
	mux.Handle("/cats-vs-dogs-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(catsVsDogsHandler), qu),
	})
	mux.Handle("/mnist-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(mnistHandler), qu),
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

type key int

const (
	queueKey key = iota
	userKey
)

func with(h ContextHandler, qu etcdqueue.Queue) ContextHandler {
	return ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
		ctx = context.WithValue(ctx, queueKey, qu)
		ctx = context.WithValue(ctx, userKey, generateUserID(req))

		return h.ServeHTTPContext(ctx, w, req)
	})
}

// Request defines common requests.
type Request struct {
	UserID  string `json:"userid"`
	URL     string `json:"url"`
	RawData string `json:"rawdata"`
	Result  string `json:"result"`
}

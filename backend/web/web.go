package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

// StartServer starts a backend webserver with stoppable listener.
func StartServer(webPort, queuePortClient, queuePortPeer int, dataDir string) (*Server, error) {
	rootCtx, rootCancel := context.WithCancel(context.Background())
	qu, err := etcdqueue.NewEmbeddedQueue(rootCtx, queuePortClient, queuePortPeer, dataDir)
	if err != nil {
		rootCancel()
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/word-predict-request-1", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(clientRequestHandler), qu),
	})
	mux.Handle("/word-predict-request-2", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(clientRequestHandler), qu),
	})
	mux.Handle("/cats-vs-dogs-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(clientRequestHandler), qu),
	})
	mux.Handle("/mnist-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(clientRequestHandler), qu),
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

// Request defines common requests.
type Request struct {
	UserID  string `json:"userid"`
	URL     string `json:"url"`
	RawData string `json:"rawdata"`
	Result  string `json:"result"`
}

var (
	requestCacheMu sync.RWMutex
	requestCache   = make(map[string]*etcdqueue.Item)
)

func clientRequestHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	reqPath := req.URL.Path
	userID := ctx.Value(userKey).(string)

	switch req.Method {
	case http.MethodPost:
		// TODO: glog.V(2).Infof
		glog.Infof("client request on %q", reqPath)

		rb, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		req.Body.Close()

		creq := Request{}
		if err = json.Unmarshal(rb, &creq); err != nil {
			return json.NewEncoder(w).Encode(&etcdqueue.Item{
				Progress: 0,
				Error:    fmt.Errorf("JSON parse error %q at %s", err.Error(), time.Now().String()[:29]),
			})
		}
		requestID := fmt.Sprintf("%s-%s-%s", userID, reqPath, hashSha512(creq.RawData))

		requestCacheMu.RLock()
		item, ok := requestCache[requestID]
		requestCacheMu.RUnlock()
		if ok {
			return json.NewEncoder(w).Encode(item)
		}

		glog.Infof("creating a new item with request ID %s", requestID)
		creq.UserID = userID
		creq.Result = ""
		rb, err = json.Marshal(creq)
		if err != nil {
			return err
		}
		item = etcdqueue.CreateItem(req.URL.Path, 100, string(rb))

		// 2. enqueue(schedule) the job
		qu := ctx.Value(queueKey).(etcdqueue.Queue)
		ch, err := qu.Add(ctx, item)
		if err != nil {
			return json.NewEncoder(w).Encode(&etcdqueue.Item{
				Progress: 0,
				Error:    fmt.Errorf("schedule error %q at %s", err.Error(), time.Now().String()[:29]),
			})
		}

		// 3. watch for changes for later request polling
		requestCacheMu.Lock()
		requestCache[requestID] = item
		requestCacheMu.Unlock()

		go watch(ctx, requestID, item, creq, ch)

	case http.MethodDelete:
		rb, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		req.Body.Close()

		creq := Request{}
		if err = json.Unmarshal(rb, &creq); err != nil {
			return json.NewEncoder(w).Encode(&etcdqueue.Item{
				Progress: 0,
				Error:    fmt.Errorf("JSON parse error %q at %s", err.Error(), time.Now().String()[:29]),
			})
		}
		requestID := fmt.Sprintf("%s-%s-%s", userID, req.URL.Path, hashSha512(creq.RawData))

		glog.Infof("requested to delete %q", requestID)
		requestCacheMu.Lock()
		delete(requestCache, requestID)
		requestCacheMu.Unlock()
		glog.Infof("deleted %q", requestID)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

func watch(ctx context.Context, requestID string, item *etcdqueue.Item, req Request, ch <-chan *etcdqueue.Item) {
	cnt := 0
	for item.Progress < 100 {
		select {
		case <-ctx.Done():
			return
		default:
		}

		requestCacheMu.RLock()
		_, ok := requestCache[requestID]
		requestCacheMu.RUnlock()
		if !ok {
			glog.Infof("%q is deleted; exiting watch routine", requestID)
		}

		// TODO: watch from queue until it's done, feed from queue service
		time.Sleep(time.Second)
		req.Result = fmt.Sprintf("Processing %+v at %s", req, time.Now().String()[:29])
		rb, err := json.Marshal(req)
		if err != nil {
			panic(err)
		}
		item.Value = string(rb)
		item.Progress = (cnt + 1) * 10
		cnt++

		requestCacheMu.Lock()
		requestCache[requestID] = item
		requestCacheMu.Unlock()
		glog.Infof("updated item with request ID %s", requestID)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

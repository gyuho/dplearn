package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	etcdqueue "github.com/gyuho/deephardway/pkg/etcd-queue"
	"github.com/gyuho/deephardway/pkg/lru"

	humanize "github.com/dustin/go-humanize"
	"github.com/golang/glog"
)

// Server warps http.Server.
type Server struct {
	mu         sync.RWMutex
	rootCtx    context.Context
	rootCancel func()
	webURL     url.URL
	httpServer *http.Server
	qu         etcdqueue.Queue

	donec chan struct{}

	requestCacheMu sync.Mutex
	requestCache   map[string]*etcdqueue.Item
}

type key int

const (
	serverKey key = iota
	queueKey
	lruCacheKey
	userKey
)

func with(h ContextHandler, srv *Server, qu etcdqueue.Queue, cache lru.Cache) ContextHandler {
	return ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
		ctx = context.WithValue(ctx, serverKey, srv)
		ctx = context.WithValue(ctx, queueKey, qu)
		ctx = context.WithValue(ctx, lruCacheKey, cache)
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

	webURL := url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", webPort)}
	srv := &Server{
		rootCtx:      rootCtx,
		rootCancel:   rootCancel,
		webURL:       webURL,
		httpServer:   &http.Server{Addr: webURL.Host, Handler: mux},
		qu:           qu,
		donec:        make(chan struct{}),
		requestCache: make(map[string]*etcdqueue.Item),
	}

	cache := lru.NewInMemory(imageCacheSize)
	cache.CreateNamespace(imageCacheBucket)

	mux.Handle("/cats-vs-dogs-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(clientRequestHandler), srv, qu, cache),
	})
	// mux.Handle("/mnist-request", &ContextAdapter{
	// 	ctx:     rootCtx,
	// 	handler: with(ContextHandlerFunc(clientRequestHandler), srv, qu, cache),
	// })
	mux.Handle("/word-predict-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(clientRequestHandler), srv, qu, cache),
	})

	gcPeriod := 5 * time.Minute
	go srv.gcCache(gcPeriod)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				glog.Fatal(err)
				os.Exit(0)
			}
			srv.rootCancel()
		}()

		glog.Infof("starting server %q", srv.webURL.String())
		if err := srv.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			glog.Fatal(err)
		}

		select {
		case <-srv.rootCtx.Done():
			return
		case <-srv.donec:
			return
		default:
			close(srv.donec)
		}
	}()
	return srv, nil
}

// gcCache garbage-collects old items in the cache.
func (srv *Server) gcCache(period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-srv.rootCtx.Done():
			return
		case <-srv.donec:
			return
		case <-ticker.C:
		}

		srv.requestCacheMu.Lock()
		for id, item := range srv.requestCache {
			glog.Warningf("%q should have been requested to delete when user leaves browser (missed DELETE request?)", id)
			if time.Since(item.CreatedAt) > period {
				delete(srv.requestCache, id)
				if item.Progress == 100 {
					glog.Infof("deleted %q because its progress is 100 (created at %s)", id, item.CreatedAt)
				} else {
					glog.Warningf("deleted %q and its progress is %d (created at %s)", id, item.Progress, item.CreatedAt)
				}
			}
		}
		srv.requestCacheMu.Unlock()
	}
}

// Stop stops the server. Useful for testing.
func (srv *Server) Stop() error {
	glog.Infof("stopping server %q", srv.webURL.String())

	srv.mu.Lock()
	srv.qu.Stop()
	if srv.httpServer == nil {
		srv.mu.Unlock()
		glog.Infof("already stopped %q", srv.webURL.String())
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

	glog.Infof("stopped server %q", srv.webURL.String())
	return nil
}

// StopNotify returns receive-only stop channel to notify the server has stopped.
func (srv *Server) StopNotify() <-chan struct{} {
	return srv.donec
}

// Request defines common requests.
type Request struct {
	UserID        string `json:"user_id"`
	RawData       string `json:"raw_data"`
	Result        string `json:"result"`
	DeleteRequest bool   `json:"delete_request"`
}

func clientRequestHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	reqPath := req.URL.Path
	srv := ctx.Value(serverKey).(*Server)
	qu := ctx.Value(queueKey).(etcdqueue.Queue)
	cache := ctx.Value(lruCacheKey).(lru.Cache)
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
			err = fmt.Errorf("JSON parse error %q at %s", err.Error(), time.Now().String()[:29])
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
		}

		// TODO: bug in ngOnDestroy?
		if creq.RawData == "" {
			glog.Info("skipping empty request...")
			return nil
		}

		switch reqPath {
		case "/cats-vs-dogs-request":
			var rawPath string
			var imageBytes []byte
			rawPath, imageBytes, err = fetchImage(cache, creq.RawData)
			if err != nil {
				err = fmt.Errorf("error %q while fetching %q", err.Error(), creq.RawData)
				glog.Warning(err)
				return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
			}

			// TODO: decode or pass to worker
			fmt.Println(rawPath, string(imageBytes[:5]))

		case "/mnist-request":

		case "/word-predict-request":

		default:
			err = fmt.Errorf("unknown request %q at %s", reqPath, time.Now().String()[:29])
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
		}

		requestID := generateRequestID(reqPath, userID, creq.RawData)

		switch creq.DeleteRequest {
		case false:
			srv.requestCacheMu.Lock()
			item, ok := srv.requestCache[requestID]
			srv.requestCacheMu.Unlock()
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
			item = etcdqueue.CreateItem(reqPath, 100, string(rb))
			glog.Infof("created a new item with request ID %s", requestID)

			// 2. enqueue(schedule) the job
			glog.Infof("enqueue-ing a new item with request ID %s", requestID)
			ch, err := qu.Add(ctx, item)
			if err != nil {
				err = fmt.Errorf("schedule error %q at %s", err.Error(), time.Now().String()[:29])
				glog.Warning(err)
				return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
			}
			glog.Infof("enqueue-ed a new item with request ID %s", requestID)

			// 3. watch for changes for later request polling
			srv.requestCacheMu.Lock()
			srv.requestCache[requestID] = item
			srv.requestCacheMu.Unlock()

			go srv.watch(ctx, requestID, item, creq, ch)

		case true:
			glog.Infof("requested to delete %q", requestID)
			srv.requestCacheMu.Lock()
			item, ok := srv.requestCache[requestID]
			if !ok {
				srv.requestCacheMu.Unlock()
				glog.Infof("already deleted %q", requestID)
				return nil
			}
			delete(srv.requestCache, requestID)
			glog.Infof("deleted %q", requestID)
			glog.Infof("canceling the item with request ID %q from the queue", requestID)
			if err = qu.Delete(ctx, item); err != nil {
				err = fmt.Errorf("schedule error %q at %s", err.Error(), time.Now().String()[:29])
				glog.Warning(err)
				srv.requestCacheMu.Unlock()
				return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
			}
			glog.Infof("canceled the item with request ID %q from the queue", requestID)
			srv.requestCacheMu.Unlock()
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

func (srv *Server) watch(ctx context.Context, requestID string, item *etcdqueue.Item, req Request, ch <-chan *etcdqueue.Item) {
	cnt := 0
	for item.Progress < 100 {
		select {
		case <-srv.donec:
			return
		case <-ctx.Done():
			return
		default:
		}

		srv.requestCacheMu.Lock()
		_, ok := srv.requestCache[requestID]
		if !ok {
			glog.Infof("%q is deleted; exiting watch routine", requestID)
			srv.requestCacheMu.Unlock()
			return
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

		srv.requestCache[requestID] = item
		srv.requestCacheMu.Unlock()
		glog.Infof("updated item with request ID %q", requestID)

		select {
		case <-srv.rootCtx.Done():
			return
		case <-srv.donec:
			return
		default:
		}
	}
}

const (
	imageCacheSize      = 100
	imageCacheBucket    = "image-cache"
	imageCacheSizeLimit = 20000000 // 20 MB
)

func fetchImage(cache lru.Cache, ep string) (string, []byte, error) {
	u, err := url.Parse(strings.TrimSpace(ep))
	if err != nil {
		return "", nil, err
	}

	rawPath := strings.TrimSpace(u.String())
	if u.RawQuery != "" {
		rawPath = strings.Replace(rawPath, "?"+u.RawQuery, "", -1)
	}

	var vi interface{}
	vi, err = cache.Get(imageCacheBucket, rawPath)
	if err != nil && err != lru.ErrKeyNotFound {
		return rawPath, nil, err
	}

	var ibt []byte
	if err == lru.ErrKeyNotFound { // not exist in cache, download, and cache it!
		switch filepath.Ext(rawPath) {
		case ".jpg":
		case ".jpeg":
		default:
			return rawPath, nil, fmt.Errorf("does not support image format %q in %q (only support 'jpg', 'jpeg')", filepath.Ext(rawPath), rawPath)
		}
		glog.Infof("downloading %q", u.String())
		var dresp *http.Response
		dresp, err = http.Get(u.String())
		if err != nil {
			return rawPath, nil, err
		}
		ibt, err = ioutil.ReadAll(dresp.Body)
		if err != nil {
			return rawPath, nil, err
		}
		dresp.Body.Close()
		if len(ibt) > imageCacheSizeLimit {
			return rawPath, nil, fmt.Errorf("%q is too large (%s, limit %s)", rawPath, humanize.Bytes(uint64(len(ibt))), humanize.Bytes(uint64(imageCacheSizeLimit)))
		}
		glog.Infof("downloaded %q (%s)", u.String(), humanize.Bytes(uint64(len(ibt))))

		glog.Infof("storing %q into cache", rawPath)
		if err = cache.Put(imageCacheBucket, rawPath, ibt); err != nil {
			return rawPath, nil, err
		}
		glog.Infof("stored %q into cache", rawPath)
	} else { // exist in cache, just use the one from cache
		glog.Infof("fetching %q from cache", rawPath)
		var ok bool
		ibt, ok = vi.([]byte)
		if !ok {
			return rawPath, nil, fmt.Errorf("expected bytes type in 'image-cache' bucket, got %v", reflect.TypeOf(vi))
		}
		glog.Infof("fetched %q from cache", rawPath)
	}

	return rawPath, ibt, nil
}

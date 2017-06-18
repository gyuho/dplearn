package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	etcdqueue "github.com/gyuho/deephardway/pkg/etcd-queue"
	"github.com/gyuho/deephardway/pkg/fileutil"
	"github.com/gyuho/deephardway/pkg/lru"
	"github.com/gyuho/deephardway/pkg/urlutil"

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
	cacheKey
	userKey
)

func with(h ContextHandler, srv *Server, qu etcdqueue.Queue, cache lru.Cache) ContextHandler {
	return ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
		ctx = context.WithValue(ctx, serverKey, srv)
		ctx = context.WithValue(ctx, queueKey, qu)
		ctx = context.WithValue(ctx, cacheKey, cache)
		ctx = context.WithValue(ctx, userKey, generateUserID(req))
		return h.ServeHTTPContext(ctx, w, req)
	})
}

// StartServer starts a backend webserver with stoppable listener.
func StartServer(webPort int, qu etcdqueue.Queue) (*Server, error) {
	rootCtx, rootCancel := context.WithCancel(context.Background())
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
	mux.Handle("/cats-vs-dogs-request/queue", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(queueHandler), srv, qu, cache),
	})
	// mux.Handle("/mnist-request", &ContextAdapter{
	// 	ctx:     rootCtx,
	// 	handler: with(ContextHandlerFunc(clientRequestHandler), srv, qu, cache),
	// })
	mux.Handle("/word-predict-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(clientRequestHandler), srv, qu, cache),
	})
	mux.Handle("/word-predict-request/queue", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(queueHandler), srv, qu, cache),
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

func queueHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	reqPath := req.URL.Path
	bucket := path.Dir(reqPath)
	qu := ctx.Value(queueKey).(etcdqueue.Queue)

	glog.Infof("[%s] client request on %q", req.Method, reqPath)
	switch req.Method {
	case http.MethodGet:
		item, err := qu.Front(ctx, bucket)
		if err != nil {
			return json.NewEncoder(w).Encode(&etcdqueue.Item{Bucket: bucket, Progress: 0, Error: err.Error()})
		}
		if item == nil { // pass empty item
			item = &etcdqueue.Item{}
		}
		return json.NewEncoder(w).Encode(item)

	case http.MethodPost:
		rb, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		io.Copy(ioutil.Discard, req.Body)
		req.Body.Close()

		var item etcdqueue.Item
		if err = json.Unmarshal(rb, &item); err != nil {
			return json.NewEncoder(w).Encode(&etcdqueue.Item{Bucket: bucket, Progress: 0, Error: err.Error()})
		}
		if _, err := qu.Enqueue(ctx, &item); err != nil {
			return json.NewEncoder(w).Encode(&etcdqueue.Item{Bucket: bucket, Progress: 0, Error: err.Error()})
		}
		return json.NewEncoder(w).Encode(&item)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

// Request defines requests from frontend.
type Request struct {
	DataFromFrontend string `json:"data_from_frontend"`
	DeleteRequest    bool   `json:"delete_request"`
}

func clientRequestHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	reqPath := req.URL.Path
	srv := ctx.Value(serverKey).(*Server)
	qu := ctx.Value(queueKey).(etcdqueue.Queue)
	cache := ctx.Value(cacheKey).(lru.Cache)
	userID := ctx.Value(userKey).(string)

	glog.Infof("[%s] client request on %q", req.Method, reqPath)
	switch req.Method {
	case http.MethodPost:
		rb, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		io.Copy(ioutil.Discard, req.Body)
		req.Body.Close()

		creq := Request{}
		if err = json.Unmarshal(rb, &creq); err != nil {
			err = fmt.Errorf("JSON parse error %q", err.Error())
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&etcdqueue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
		}

		if creq.DataFromFrontend == "" { // TODO: bug in ngOnDestroy?
			glog.Warning("skipping empty request...")
			return nil
		}

		switch reqPath {
		case "/cats-vs-dogs-request":
			var fpath string
			fpath, err = cacheImage(cache, creq.DataFromFrontend)
			if err != nil {
				err = fmt.Errorf("error %q while fetching %q", err.Error(), creq.DataFromFrontend)
				glog.Warning(err)
				return json.NewEncoder(w).Encode(&etcdqueue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
			}
			creq.DataFromFrontend = fpath

		case "/mnist-request":

		case "/word-predict-request":

		default:
			err = fmt.Errorf("unknown request %q", reqPath)
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&etcdqueue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
		}

		requestID := generateRequestID(reqPath, userID, creq.DataFromFrontend)

		switch creq.DeleteRequest {
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
			if err = qu.Dequeue(ctx, item); err != nil {
				err = fmt.Errorf("qu.Dequeue error %q", err.Error())
				glog.Warning(err)
				srv.requestCacheMu.Unlock()
				return json.NewEncoder(w).Encode(&etcdqueue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
			}
			srv.requestCacheMu.Unlock()
			glog.Infof("deleted %q", requestID)

		case false:
			srv.requestCacheMu.Lock()
			item, ok := srv.requestCache[requestID]
			if ok {
				srv.requestCacheMu.Unlock()
				return json.NewEncoder(w).Encode(item)
			}

			// enqueue(schedule) the job
			glog.Infof("creating an item with request ID %s", requestID)
			item = etcdqueue.CreateItem(reqPath, 100, creq.DataFromFrontend)
			ch, err := qu.Enqueue(ctx, item)
			if err != nil {
				srv.requestCacheMu.Unlock()
				err = fmt.Errorf("qu.Enqueue error %q", err.Error())
				glog.Warning(err)
				return json.NewEncoder(w).Encode(&etcdqueue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
			}

			// watch for changes from worker, keep the cache up-to-date
			// - waits until the worker processor computes the job
			// - waits until the worker processor writes back to queue
			// - queue watcher gets notified and writes back to 'path.Join(pfxScheduled, bucket)'
			srv.requestCache[requestID] = item
			srv.requestCacheMu.Unlock()
			go srv.watch(ctx, requestID, ch)

			glog.Infof("created an item with request ID %s", requestID)
			copied := *item
			copied.Value = fmt.Sprintf("requested job with %q", copied.Value)
			return json.NewEncoder(w).Encode(&copied)
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

func (srv *Server) watch(ctx context.Context, requestID string, ch <-chan *etcdqueue.Item) {
	item := &etcdqueue.Item{Progress: 0}
	for item.Progress < 100 && !item.Canceled {
		srv.requestCacheMu.Lock()
		_, ok := srv.requestCache[requestID]
		if !ok {
			glog.Infof("watcher: %q is deleted", requestID)
			srv.requestCacheMu.Unlock()
			return
		}
		srv.requestCacheMu.Unlock()

		select {
		case <-srv.donec:
			return
		case <-ctx.Done():
			return
		case item = <-ch:
			srv.requestCacheMu.Lock()
			srv.requestCache[requestID] = item
			srv.requestCacheMu.Unlock()
			glog.Infof("watcher: received an update on %q", requestID)
			if item.Canceled {
				glog.Infof("watcher: %q is canceld", requestID)
			}
		}
	}
}

const (
	imageCacheSize      = 100
	imageCacheBucket    = "image-cache"
	imageCacheSizeLimit = 15000000 // 15 MB
)

func cacheImage(cache lru.Cache, ep string) (string, error) {
	rawPath := urlutil.TrimQuery(ep)

	vi, err := cache.Get(imageCacheBucket, rawPath)
	if err != nil && err != lru.ErrKeyNotFound {
		return "", err
	}

	var fpath string
	if err != lru.ErrKeyNotFound { // exist in cache, just use the one from cache
		glog.Infof("fetching %q from cache", rawPath)
		var ok bool
		fpath, ok = vi.(string)
		if !ok {
			return fpath, fmt.Errorf("expected bytes type in 'image-cache' bucket, got %v", reflect.TypeOf(vi))
		}
		glog.Infof("fetched %q from cache", rawPath)
	} else { // not exist in cache, download, and cache it!
		switch filepath.Ext(rawPath) {
		case ".jpg", ".jpeg":
		case ".png":
		default:
			return "", fmt.Errorf("not support %q in %q (must be jpg, jpeg, png)", filepath.Ext(rawPath), rawPath)
		}

		size, sizet, err := urlutil.GetContentLength(rawPath)
		if err != nil {
			return "", fmt.Errorf("error when fetching %q", rawPath)
		}
		if size > imageCacheSizeLimit {
			return "", fmt.Errorf("%q is too big; %s > %s(limit)", rawPath, sizet, humanize.Bytes(uint64(imageCacheSizeLimit)))
		}

		glog.Infof("downloading %q", rawPath)
		var data []byte
		data, err = urlutil.Get(rawPath)
		if err != nil {
			return "", err
		}
		glog.Infof("downloaded %q (%s)", rawPath, humanize.Bytes(uint64(len(data))))

		fpath = filepath.Join("/tmp", base64.StdEncoding.EncodeToString([]byte(rawPath))+filepath.Ext(rawPath))

		glog.Infof("saving %q to %q", rawPath, fpath)
		if err = fileutil.WriteToFile(fpath, data); err != nil {
			return fpath, err
		}
		glog.Infof("saved %q to %q", rawPath, fpath)

		glog.Infof("storing %q into cache", rawPath)
		if err = cache.Put(imageCacheBucket, rawPath, fpath); err != nil {
			return "", err
		}
		glog.Infof("stored %q into cache", rawPath)
	}

	return fpath, nil
}

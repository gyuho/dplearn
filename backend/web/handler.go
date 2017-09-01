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

	queue "github.com/gyuho/dplearn/pkg/etcd-queue"
	"github.com/gyuho/dplearn/pkg/fileutil"
	"github.com/gyuho/dplearn/pkg/lru"
	"github.com/gyuho/dplearn/pkg/urlutil"

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
	qu         queue.Queue

	donec chan struct{}

	requestCache sync.Map
}

type key int

const (
	serverKey key = iota
	queueKey
	cacheKey
	userKey
)

func with(h ContextHandler, srv *Server, qu queue.Queue, cache lru.Cache) ContextHandler {
	return ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
		ctx = context.WithValue(ctx, serverKey, srv)
		ctx = context.WithValue(ctx, queueKey, qu)
		ctx = context.WithValue(ctx, cacheKey, cache)
		ctx = context.WithValue(ctx, userKey, generateUserID(req))
		return h.ServeHTTPContext(ctx, w, req)
	})
}

const (
	enqueueTTL = 30 * time.Minute

	// RequestIDHeader is the field name for request ID header.
	RequestIDHeader = "Request-Id"
)

// StartServer starts a backend webserver with stoppable listener.
func StartServer(scheme, hostPort string, qu queue.Queue) (*Server, error) {
	rootCtx, rootCancel := context.WithCancel(context.Background())
	mux := http.NewServeMux()
	webURL := url.URL{Scheme: scheme, Host: hostPort}
	srv := &Server{
		rootCtx:    rootCtx,
		rootCancel: rootCancel,
		webURL:     webURL,
		httpServer: &http.Server{Addr: webURL.Host, Handler: mux},
		qu:         qu,
		donec:      make(chan struct{}),
	}

	cache := lru.NewInMemory(imageCacheSize)
	cache.CreateNamespace(imageCacheBucket)

	mux.Handle("/healthz", &ContextAdapter{
		ctx: rootCtx,
		handler: ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			w.WriteHeader(200)
			w.Write([]byte("ok"))
			return nil
		}),
	})
	mux.Handle("/cats-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: with(ContextHandlerFunc(clientRequestHandler), srv, qu, cache),
	})
	mux.Handle("/cats-request/queue", &ContextAdapter{
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

		srv.requestCache.Range(func(k, v interface{}) bool {
			if k == nil || v == nil {
				return false
			}
			id := k.(string)
			item := v.(*queue.Item)

			glog.Warningf("%q should have been requested to delete when user leaves browser (missed DELETE request?)", id)
			if time.Since(item.CreatedAt) > period {
				srv.requestCache.Delete(k)
				if item.Progress == queue.MaxProgress {
					glog.Infof("deleted %q because its progress is %d (created at %s)", id, queue.MaxProgress, item.CreatedAt)
				} else {
					glog.Warningf("deleted %q and its progress is %d (created at %s)", id, item.Progress, item.CreatedAt)
				}
			}
			return true
		})
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
	srv := ctx.Value(serverKey).(*Server)
	qu := ctx.Value(queueKey).(queue.Queue)

	switch req.Method {
	case http.MethodGet:
		return json.NewEncoder(w).Encode(<-qu.Pop(ctx, bucket))

	case http.MethodPost:
		rb, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		// TODO: python gets ('Connection aborted.', BadStatusLine("''",))
		io.Copy(ioutil.Discard, req.Body)
		req.Body.Close()

		var item queue.Item
		if err = json.Unmarshal(rb, &item); err != nil {
			return json.NewEncoder(w).Encode(&queue.Item{Bucket: bucket, Progress: 0, Error: err.Error()})
		}

		if item.Bucket == "" || item.Key == "" || item.Value == "" || item.RequestID == "" {
			return json.NewEncoder(w).Encode(&queue.Item{Bucket: bucket, Progress: 0, Error: fmt.Sprintf("invalid item: %+v", item)})
		}

		_, ok := srv.requestCache.Load(item.RequestID)
		if !ok {
			return json.NewEncoder(w).Encode(&queue.Item{Bucket: bucket, Progress: 0, Error: fmt.Sprintf("unknown request ID %q", item.RequestID)})
		}
		srv.requestCache.Store(item.RequestID, item)

		glog.Infof("queue received POST on %q", item.RequestID)
		return json.NewEncoder(w).Encode(&item)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}
	return nil
}

// Request defines requests from frontend.
type Request struct {
	DataFromFrontend string `json:"data_from_frontend"`
	CreateRequest    bool   `json:"create_request"`
}

func clientRequestHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	reqPath := req.URL.Path
	srv := ctx.Value(serverKey).(*Server)
	qu := ctx.Value(queueKey).(queue.Queue)
	cache := ctx.Value(cacheKey).(lru.Cache)
	userID := ctx.Value(userKey).(string)

	switch req.Method {
	case http.MethodGet: // item status fetch
		requestID := req.Header.Get(RequestIDHeader)
		if requestID == "" {
			err := fmt.Errorf("expected %q from header (got %+v)", RequestIDHeader, req.Header)
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&queue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
		}
		vi, ok := srv.requestCache.Load(requestID)
		if !ok {
			err := fmt.Errorf("cannot find request ID %q", requestID)
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&queue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
		}
		return json.NewEncoder(w).Encode(vi)

	case http.MethodPost: // item creation/cancel
		rb, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		// TODO: python gets ('Connection aborted.', BadStatusLine("''",))
		io.Copy(ioutil.Discard, req.Body)
		req.Body.Close()

		creq := Request{}
		if err = json.Unmarshal(rb, &creq); err != nil {
			err = fmt.Errorf("JSON parse error %q", err.Error())
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&queue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
		}
		if creq.DataFromFrontend == "" {
			glog.Warning("TODO: skipping empty request... bug in frontend ngOnDestroy?")
			return nil
		}

		switch reqPath {
		case "/cats-request":
			var imgFilePath string
			imgFilePath, err = cacheImage(cache, creq.DataFromFrontend)
			if err != nil {
				err = fmt.Errorf("error %q while fetching %q", err.Error(), creq.DataFromFrontend)
				glog.Warning(err)
				return json.NewEncoder(w).Encode(&queue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
			}
			creq.DataFromFrontend = imgFilePath

		default:
			err = fmt.Errorf("unknown request %q", reqPath)
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&queue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
		}

		requestID := generateRequestID(reqPath, userID, creq.DataFromFrontend)

		switch creq.CreateRequest {
		case true:
			glog.Infof("fetching %q before creating item", requestID)
			v, ok := srv.requestCache.Load(requestID)
			if ok {
				glog.Infof("fetched %q before creating item, no need to create", requestID)
				return json.NewEncoder(w).Encode(v)
			}

			item := queue.CreateItem(reqPath, 100, creq.DataFromFrontend)
			item.RequestID = requestID

			if err = qu.Add(ctx, item, queue.WithTTL(enqueueTTL)); err != nil {
				glog.Warning(err)
				return json.NewEncoder(w).Encode(&queue.Item{Bucket: reqPath, Progress: 0, Error: err.Error()})
			}
			srv.requestCache.Store(requestID, item)

			glog.Infof("created an item with request ID %s", requestID)
			copied := *item
			copied.Value = fmt.Sprintf("[BACKEND - ACK] Requested %q (request ID: %s)", copied.Value, requestID)
			return json.NewEncoder(w).Encode(&copied)

		case false:
			glog.Infof("deleting %q", requestID)
			srv.requestCache.Delete(requestID)
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}
	return nil
}

const (
	imageCacheSize      = 100
	imageCacheBucket    = "image-cache"
	imageCacheSizeLimit = 15000000 // 15 MB
)

func cacheImage(cache lru.Cache, ep string) (string, error) {
	originURL := urlutil.TrimQuery(ep)

	vi, err := cache.Get(imageCacheBucket, originURL)
	if err != nil && err != lru.ErrKeyNotFound {
		return "", err
	}

	var imgFilePath string
	if err != lru.ErrKeyNotFound { // exist in cache, just use the one from cache
		glog.Infof("fetching %q from cache", originURL)
		var ok bool
		imgFilePath, ok = vi.(string)
		if !ok {
			return imgFilePath, fmt.Errorf("expected bytes type in 'image-cache' bucket, got %v", reflect.TypeOf(vi))
		}
		glog.Infof("fetched %q from cache", originURL)
	} else { // not exist in cache, download, and cache it!
		switch filepath.Ext(originURL) {
		case ".jpg", ".jpeg":
		case ".png":
		default:
			return "", fmt.Errorf("not support %q in %q (must be jpg, jpeg, png)", filepath.Ext(originURL), originURL)
		}

		size, sizet, err := urlutil.GetContentLength(originURL)
		if err != nil {
			return "", fmt.Errorf("error when fetching %q", originURL)
		}
		if size > imageCacheSizeLimit {
			return "", fmt.Errorf("%q is too big; %s > %s(limit)", originURL, sizet, humanize.Bytes(uint64(imageCacheSizeLimit)))
		}

		glog.Infof("downloading %q", originURL)
		var data []byte
		data, err = urlutil.Get(originURL)
		if err != nil {
			return "", err
		}
		glog.Infof("downloaded %q (%s)", originURL, humanize.Bytes(uint64(len(data))))

		imgFilePath = filepath.Join("/tmp", base64.StdEncoding.EncodeToString([]byte(originURL))+filepath.Ext(originURL))
		glog.Infof("saving %q to %q", originURL, imgFilePath)
		if err = fileutil.WriteToFile(imgFilePath, data); err != nil {
			return imgFilePath, err
		}
		glog.Infof("saved %q to %q", originURL, imgFilePath)

		glog.Infof("storing %q into cache", originURL)
		if err = cache.Put(imageCacheBucket, originURL, imgFilePath); err != nil {
			return "", err
		}
		glog.Infof("stored %q into cache", originURL)
	}

	return imgFilePath, nil
}

package web

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	etcdqueue "github.com/gyuho/deephardway/pkg/etcd-queue"
	"github.com/gyuho/deephardway/pkg/lru"

	"github.com/coreos/etcd/clientv3"
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
	cache := ctx.Value(cacheKey).(lru.Cache)
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
			err = fmt.Errorf("JSON parse error %q", err.Error())
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
		}

		if creq.RawData == "" { // TODO: bug in ngOnDestroy?
			glog.Warning("skipping empty request...")
			return nil
		}

		switch reqPath {
		case "/cats-vs-dogs-request":
			var fpath string
			fpath, err = cacheImage(cache, creq.RawData)
			if err != nil {
				err = fmt.Errorf("error %q while fetching %q", err.Error(), creq.RawData)
				glog.Warning(err)
				return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
			}
			creq.RawData = fpath

			// TODO: pass to worker
			fmt.Println(creq.RawData)

		case "/mnist-request":

		case "/word-predict-request":

		default:
			err = fmt.Errorf("unknown request %q", reqPath)
			glog.Warning(err)
			return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
		}

		requestID := generateRequestID(reqPath, userID, creq.RawData)

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
			if err = qu.Delete(ctx, item); err != nil {
				err = fmt.Errorf("qu.Delete error %q", err.Error())
				glog.Warning(err)
				srv.requestCacheMu.Unlock()
				return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
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

			glog.Infof("creating a item with request ID %s", requestID)
			creq.UserID = userID
			creq.Result = ""
			rb, err = json.Marshal(creq)
			if err != nil {
				srv.requestCacheMu.Unlock()
				err = fmt.Errorf("json.Marshal error %q", err.Error())
				glog.Warning(err)
				return err
			}

			// 2. enqueue(schedule) the job
			item = etcdqueue.CreateItem(reqPath, 100, string(rb))
			ch, err := qu.Add(ctx, item)
			if err != nil {
				srv.requestCacheMu.Unlock()
				err = fmt.Errorf("qu.Add error %q", err.Error())
				glog.Warning(err)
				return json.NewEncoder(w).Encode(&etcdqueue.Item{Progress: 0, Error: err.Error()})
			}

			// 3. watch for changes from worker
			// - now 'item' needs to wait until scheduled in 'path.Join(pfxWorker, bucket)'
			// - waits until the worker processor computes the job
			// - waits until the worker processor writes back to queue
			// - queue watcher gets notified and writes back to 'path.Join(pfxScheduled, bucket)'
			srv.requestCache[requestID] = item
			srv.requestCacheMu.Unlock()
			go srv.watch(ctx, requestID, ch)
			glog.Infof("created a item with request ID %s", requestID)

			// for testing, simulate worker process
			go simulateWorker(qu, item)
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

func (srv *Server) watch(ctx context.Context, requestID string, ch <-chan *etcdqueue.Item) {
	item := &etcdqueue.Item{Progress: 0}
	for item.Progress < 100 {
		srv.requestCacheMu.Lock()
		_, ok := srv.requestCache[requestID]
		if !ok {
			glog.Infof("%q is deleted; exiting watch routine", requestID)
			srv.requestCacheMu.Unlock()
			return
		}
		srv.requestCacheMu.Unlock()

		// watch from queue until it's done, feed from queue service
		select {
		case <-srv.donec:
			return
		case <-ctx.Done():
			return
		case item = <-ch:
			if item.Canceled {
				glog.Infof("%q is canceld; exiting watch routine", item.Key)
				return
			}
			srv.requestCacheMu.Lock()
			srv.requestCache[requestID] = item
			srv.requestCacheMu.Unlock()
			glog.Infof("updated item with request ID %q", requestID)
		}
	}
}

const (
	imageCacheSize      = 100
	imageCacheBucket    = "image-cache"
	imageCacheSizeLimit = 15000000 // 15 MB
)

func cacheImage(cache lru.Cache, ep string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(ep))
	if err != nil {
		return "", err
	}

	rawPath := strings.TrimSpace(u.String())
	if u.RawQuery != "" {
		rawPath = strings.Replace(rawPath, "?"+u.RawQuery, "", -1)
	}

	var vi interface{}
	vi, err = cache.Get(imageCacheBucket, rawPath)
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

		var hresp *http.Response
		hresp, err = http.Head(rawPath)
		if err != nil {
			return "", err
		}
		hresp.Body.Close()
		if hresp.ContentLength > imageCacheSizeLimit {
			return "", fmt.Errorf("%q is too big; %s > %s(limit)", rawPath, humanize.Bytes(uint64(hresp.ContentLength)), humanize.Bytes(uint64(imageCacheSizeLimit)))
		}

		glog.Infof("downloading %q", rawPath)
		var dresp *http.Response
		dresp, err = http.Get(rawPath)
		if err != nil {
			return "", err
		}
		var data []byte
		data, err = ioutil.ReadAll(dresp.Body)
		if err != nil {
			return "", err
		}
		dresp.Body.Close()
		glog.Infof("downloaded %q (%s)", rawPath, humanize.Bytes(uint64(len(data))))

		fpath = filepath.Join("/tmp", base64.StdEncoding.EncodeToString([]byte(rawPath))+filepath.Ext(rawPath))

		glog.Infof("saving %q to %q", rawPath, fpath)
		if err = toFile(data, fpath); err != nil {
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

func toFile(data []byte, fpath string) error {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			glog.Fatal(err)
		}
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		glog.Fatal(err)
	}
	return f.Sync()
}

func simulateWorker(qu etcdqueue.Queue, item *etcdqueue.Item) {
	origItem := item
	time.Sleep(15 * time.Second)

	cli, err := clientv3.New(clientv3.Config{Endpoints: qu.ClientEndpoints()})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	workerKey := path.Join("_wokr", origItem.Bucket)
	scheduleKey := origItem.Key

	scheduleValue, err := json.Marshal(origItem)
	if err != nil {
		panic(err)
	}

	var newValue []byte
	for {
		gresp, err := cli.Get(context.Background(), workerKey)
		if err != nil {
			panic(err)
		}
		if len(gresp.Kvs) != 1 {
			glog.Infof("%q is not yet scheduled for worker", workerKey)
			time.Sleep(time.Second)
			continue
		}
		if bytes.Equal(gresp.Kvs[0].Value, scheduleValue) {
			glog.Infof("%q is now scheduled for worker (value: %q)", workerKey, string(scheduleValue))

			var creq Request
			if err = json.Unmarshal(gresp.Kvs[0].Value, &creq); err != nil {
				panic(err)
			}

			// simulate that worker finishes computation
			glog.Infof("updating %+v", creq)
			creq.Result = "mocked result text at " + time.Now().String()[:26]
			var rb []byte
			rb, err = json.Marshal(creq)
			if err != nil {
				panic(err)
			}
			copied := *origItem
			copied.Progress = 100
			copied.Value = string(rb)
			newValue, err = json.Marshal(copied)
			if err != nil {
				panic(err)
			}
			if _, err = cli.Put(context.Background(), workerKey, string(newValue)); err != nil {
				panic(err)
			}
			glog.Infof("updated %q with %q", workerKey, string(newValue))
			break
		}

		glog.Infof("%q is not yet scheduled; another job %q is in progress", workerKey, string(gresp.Kvs[0].Value))
		time.Sleep(time.Second)
	}
	// scheduler catches this write and writes back to scheduleKey
	for {
		gresp, err := cli.Get(context.Background(), scheduleKey)
		if err != nil {
			panic(err)
		}
		if len(gresp.Kvs) == 0 {
			glog.Infof("%q has not been written to etcd yet", scheduleKey)
			time.Sleep(time.Second)
			continue
		}
		if len(gresp.Kvs) != 1 {
			glog.Fatalf("%q must have 1 KV (got %+v)", scheduleKey, gresp.Kvs)
		}
		if !bytes.Equal(gresp.Kvs[0].Value, newValue) {
			if !bytes.Equal(gresp.Kvs[0].Value, scheduleValue) {
				glog.Fatalf("%q must have old value %q if not new value, but got %q", scheduleKey, scheduleValue, gresp.Kvs[0].Value)
			}
			glog.Infof("%q has not yet received new value, still has %q", scheduleKey, gresp.Kvs[0].Value)
			time.Sleep(time.Second)
			continue
		}

		glog.Infof("%q now has new value (old value %q)", newValue, scheduleValue)
		break
	}
}

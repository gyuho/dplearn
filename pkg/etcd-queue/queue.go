// Package etcdqueue implements queue service backed by etcd.
package etcdqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/etcdserver/api/v3client"
	"github.com/golang/glog"
)

const (
	// all scheduled jobs are namespaced with pfxScheduled
	pfxScheduled = "__ETCD_QUEUE_SCHEDULED"
	pfxTODO      = "__ETCD_QUEUE_TODO"
	pfxCompleted = "__ETCD_QUEUE_COMPLETED"
)

// StatusCode represents the job status.
type StatusCode int8

const (
	// StatusCodeScheduled is the initial status.
	StatusCodeScheduled StatusCode = iota
	// StatusCodeDone is the final status.
	StatusCodeDone
)

// Queue wraps single-node embedded etcd cluster.
type Queue struct {
	mu            sync.RWMutex
	rootDir       string
	cfg           *embed.Config
	srv           *embed.Etcd
	cli           *clientv3.Client
	watchInterval time.Duration

	rootCtx    context.Context
	rootCancel func()
	buckets    map[string]chan error
}

// StartQueue starts a new etcd server.
// cport is the TCP port used for etcd client request serving.
// pport is for etcd peer traffic, and still needed even if it's a single-node cluster.
func StartQueue(cport, pport int) (*Queue, error) {
	cfg := embed.NewConfig()
	cfg.ClusterState = embed.ClusterStateFlagNew

	rootDir, err := ioutil.TempDir(os.TempDir(), "etcd-queue")
	if err != nil {
		return nil, err
	}

	cfg.Name = "etcd-queue"
	cfg.Dir = filepath.Join(rootDir, cfg.Name+".data-dir-etcd")
	cfg.WalDir = filepath.Join(rootDir, cfg.Name+".data-dir-etcd", "wal")

	curl := url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", cport)}
	cfg.ACUrls = []url.URL{curl}
	cfg.LCUrls = []url.URL{curl}

	purl := url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", pport)}
	cfg.APUrls = []url.URL{purl}
	cfg.LPUrls = []url.URL{purl}

	cfg.InitialCluster = fmt.Sprintf("%s=%s", cfg.Name, cfg.APUrls[0].String())

	// auto-compaction every hour
	cfg.AutoCompactionRetention = 1
	// single-node, so aggressively snapshot/discard Raft log entries
	cfg.SnapCount = 1000

	glog.Infof("starting %q with endpoint %q", cfg.Name, curl.String())
	var srv *embed.Etcd
	srv, err = embed.StartEtcd(cfg)
	if err != nil {
		return nil, err
	}
	select {
	case <-srv.Server.ReadyNotify():
		err = nil
	case err = <-srv.Err():
	case <-srv.Server.StopNotify():
		err = fmt.Errorf("received from etcdserver.Server.StopNotify")
	}
	if err != nil {
		return nil, err
	}
	glog.Infof("started %q with endpoint %q", cfg.Name, curl.String())

	cli := v3client.New(srv.Server)

	// issue linearized read to ensure leader election
	glog.Infof("GET request to endpoint %q", curl.String())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = cli.Get(ctx, "foo")
	cancel()
	glog.Infof("GET request succeeded on endpoint %q", curl.String())

	ctx, cancel = context.WithCancel(context.Background())
	return &Queue{
		rootDir:       rootDir,
		cfg:           cfg,
		srv:           srv,
		cli:           cli,
		rootCtx:       ctx,
		rootCancel:    cancel,
		buckets:       make(map[string]chan error),
		watchInterval: time.Second,
	}, err
}

// ClientEndpoints returns the client endpoints.
func (qu *Queue) ClientEndpoints() []string {
	qu.mu.RLock()
	defer qu.mu.RUnlock()
	return []string{qu.cfg.LCUrls[0].String()}
}

// Client returns the embedded client.
func (qu *Queue) Client() *clientv3.Client {
	qu.mu.RLock()
	defer qu.mu.RUnlock()
	return qu.cli
}

// SetWatchInterval upates watch interval.
func (qu *Queue) SetWatchInterval(dur time.Duration) {
	qu.mu.Lock()
	qu.watchInterval = dur
	qu.mu.Unlock()
}

// Stop stops the etcd server.
func (qu *Queue) Stop() {
	glog.Infof("stopping %q with endpoint %q", qu.cfg.Name, qu.cfg.LCUrls[0].String())

	qu.mu.Lock()
	qu.rootCancel()
	for bucket, errc := range qu.buckets {
		glog.Infof("stopping bucket %q", bucket)
		err := <-errc
		if err != nil && err != context.Canceled {
			glog.Warningf("watch error: %v", err)
		}
		glog.Infof("stopped bucket %q", bucket)
	}
	qu.cli.Close()
	qu.srv.Close()
	os.RemoveAll(qu.rootDir)

	qu.mu.Unlock()

	glog.Infof("stopped %q with endpoint %q", qu.cfg.Name, qu.cfg.LCUrls[0].String())
}

// Add adds an item to the queue.
func (qu *Queue) Add(ctx context.Context, it *Item) (<-chan *Item, error) {
	qu.mu.Lock()
	defer qu.mu.Unlock()

	key := it.Key
	val, err := json.Marshal(it.Value)
	if err != nil {
		return nil, err
	}

	err = qu.put(ctx, key, val)
	if err != nil {
		return nil, err
	}

	if _, ok := qu.buckets[it.Bucket]; !ok {
		if err = qu.put(ctx, path.Join(pfxTODO, it.Bucket), val); err != nil {
			return nil, err
		}
		qu.buckets[it.Bucket] = make(chan error, 1)

		go qu.watch(qu.rootCtx, it.Bucket, qu.buckets[it.Bucket])
	}

	wch := qu.cli.Watch(ctx, key)

	copied := *it
	ch := make(chan *Item, 1)

	// assume first event is always 'update' event; ignore delete events afterwards
	go func() {
		select {
		case wresp := <-wch:
			if len(wresp.Events) != 1 {
				copied.Error = fmt.Errorf("%q should receive 1 watch event, got %+v", key, wresp.Events)
				ch <- &copied
				close(ch)
				return
			}

			valBytes := wresp.Events[0].Kv.Value
			var val Value
			if err := json.Unmarshal(valBytes, &val); err != nil {
				copied.Error = fmt.Errorf("cannot parse %s", string(valBytes))
				ch <- &copied
				close(ch)
				return
			}

			glog.Infof("%q has been updated", key)
			copied.Value = val
			ch <- &copied
			close(ch)
			return

		case <-ctx.Done():
			copied.Error = ctx.Err()
			ch <- &copied
			close(ch)
			return
		}
	}()
	return ch, nil
}

// watch watches on the queue and schedules the jobs.
// Point is never miss events, thus one routine must always watch path.Join(pfxTODO, bucket)
// 1. blocks until TODO job is done, notified via watch events
// 2. notify the client back with the new results on the key (Key field in Item)
// 3. delete the DONE key from the queue, and move to pfxCompleted + Key
// 4. fetch one new job from path.Join(pfxScheduled, bucket)
// 5. skip if there is no job to schedule
// 6. write this job to path.Join(pfxTODO, bucket)
// 7. drain watch events for this wrtie
// repeat!
func (qu *Queue) watch(ctx context.Context, bucket string, errc chan error) {
	defer close(errc)

	for {
		if err := qu._watch(ctx, bucket); err != nil {
			if err == context.Canceled {
				errc <- err
				return
			}
			glog.Warning(err)
		}

		// below is implemented for failure tolerance; retry logic
		keyToWatch := path.Join(pfxTODO, bucket)
		pfxToFetch := path.Join(pfxScheduled, bucket)

		// resetting current TODO job
		resp, err := qu.cli.Get(ctx, keyToWatch)
		if err != nil {
			errc <- err
			return
		}
		if len(resp.Kvs) != 1 {
			errc <- fmt.Errorf("len(resp.Kvs) expected 1, got %+v", resp.Kvs)
			return
		}
		valBytes := resp.Kvs[0].Value
		var val Value
		if err = json.Unmarshal(valBytes, &val); err != nil {
			errc <- err
			return
		}
		if val.StatusCode == StatusCodeDone {
			glog.Warningf("watch might have failed after %q is finished", val.Key)

			// 2. notify the client back with the new results on the key (ID field in Item)
			glog.Infof("%q is done", val.Key)
			if err := qu.put(ctx, val.Key, valBytes); err != nil {
				errc <- err
				return
			}

			// 3. delete the DONE key from the queue, and move to pfxCompleted + Key
			glog.Infof("%q is deleted", val.Key)
			if err := qu.delete(ctx, val.Key); err != nil {
				errc <- err
				return
			}
			cKey := path.Join(pfxCompleted, val.Key)
			if err := qu.put(ctx, cKey, valBytes); err != nil {
				errc <- err
				return
			}
			glog.Infof("%q is written", cKey)

			// 4. fetch one new job from path.Join(pfxScheduled, bucket)
			resp, err := qu.cli.Get(ctx, pfxToFetch, append(clientv3.WithFirstKey(), clientv3.WithPrefix())...)
			if err != nil {
				errc <- err
				return
			}

			// 5. skip if there is no job to schedule
			if len(resp.Kvs) == 0 {
				glog.Infof("no job to schedule on the bucket %q", bucket)
				continue
			}
			if len(resp.Kvs) != 1 {
				errc <- fmt.Errorf("%q should return only one key-value pair (got %+v)", pfxToFetch, resp.Kvs)
				return
			}
			fetchBytes := resp.Kvs[0].Value
			var fval Value
			if err := json.Unmarshal(fetchBytes, &fval); err != nil {
				errc <- fmt.Errorf("%q has wrong JSON %q", resp.Kvs[0].Key, string(fetchBytes))
				return
			}
			if fval.StatusCode != StatusCodeScheduled {
				errc <- fmt.Errorf("%q must have status code scheduled, got %d", fval.Key, fval.StatusCode)
				return
			}

			// 6. write this job to path.Join(pfxTODO, bucket)
			glog.Infof("%q is scheduled", string(resp.Kvs[0].Key))
			if err := qu.put(ctx, keyToWatch, resp.Kvs[0].Value); err != nil {
				errc <- err
				return
			}

			// continue to '_watch'
		}
	}
}

func (qu *Queue) _watch(ctx context.Context, bucket string) error {
	keyToWatch := path.Join(pfxTODO, bucket)
	pfxToFetch := path.Join(pfxScheduled, bucket)

	wch := qu.cli.Watch(ctx, keyToWatch)
	glog.Infof("watching %q", keyToWatch)

	for {
		// 1. blocks until TODO job is done, notified via watch events
		select {
		case wresp := <-wch:
			if len(wresp.Events) != 1 {
				return fmt.Errorf("no watch events on %q (%+v, %v)", keyToWatch, wresp, wresp.Err())
			}
			valBytes := wresp.Events[0].Kv.Value
			var val Value
			if err := json.Unmarshal(valBytes, &val); err != nil {
				return fmt.Errorf("%q returned wrong value %q (%v)", keyToWatch, string(valBytes), err)
			}
			if val.StatusCode != StatusCodeDone {
				return fmt.Errorf("wrong status code %d (expected %d)", val.StatusCode, StatusCodeDone)
			}

			// 2. notify the client back with the new results on the key (ID field in Item)
			glog.Infof("%q is done", val.Key)
			if err := qu.put(ctx, val.Key, valBytes); err != nil {
				return err
			}

			// 3. delete the DONE key from the queue, and move to pfxCompleted + Key
			glog.Infof("%q is deleted", val.Key)
			if err := qu.delete(ctx, val.Key); err != nil {
				return err
			}
			cKey := path.Join(pfxCompleted, val.Key)
			if err := qu.put(ctx, cKey, valBytes); err != nil {
				return err
			}
			glog.Infof("%q is written", cKey)

			// 4. fetch one new job from path.Join(pfxScheduled, bucket)
			resp, err := qu.cli.Get(ctx, pfxToFetch, append(clientv3.WithFirstKey(), clientv3.WithPrefix())...)
			if err != nil {
				return err
			}

			// 5. skip if there is no job to schedule
			if len(resp.Kvs) == 0 {
				glog.Infof("no job to schedule on the bucket %q", bucket)
				continue
			}
			if len(resp.Kvs) != 1 {
				return fmt.Errorf("%q should return only one key-value pair (got %+v)", pfxToFetch, resp.Kvs)
			}
			fetchBytes := resp.Kvs[0].Value
			var fval Value
			if err := json.Unmarshal(fetchBytes, &fval); err != nil {
				return fmt.Errorf("%q has wrong JSON %q", resp.Kvs[0].Key, string(fetchBytes))
			}
			if fval.StatusCode != StatusCodeScheduled {
				return fmt.Errorf("%q must have status code scheduled, got %d", fval.Key, fval.StatusCode)
			}

			// 6. write this job to path.Join(pfxTODO, bucket)
			glog.Infof("%q is scheduled", string(resp.Kvs[0].Key))
			if err := qu.put(ctx, keyToWatch, resp.Kvs[0].Value); err != nil {
				return err
			}

			// 7. drain watch events for this wrtie
			select {
			case wresp = <-wch:
				if len(wresp.Events) != 1 {
					return fmt.Errorf("no watch events on %q after schedule (%+v, %v)", keyToWatch, wresp, wresp.Err())
				}
				valBytes := wresp.Events[0].Kv.Value
				var val Value
				if err := json.Unmarshal(valBytes, &val); err != nil {
					return fmt.Errorf("%q returned wrong value %q (%v)", keyToWatch, string(valBytes), err)
				}
				if val.StatusCode != StatusCodeScheduled {
					return fmt.Errorf("wrong status code %d (expected %d)", val.StatusCode, StatusCodeScheduled)
				}
				if !bytes.Equal(resp.Kvs[0].Value, valBytes) {
					return fmt.Errorf("scheduled value expected %q, got %q", string(resp.Kvs[0].Value), string(valBytes))
				}

			case <-ctx.Done():
				return ctx.Err()
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (qu *Queue) put(ctx context.Context, key string, val []byte) error {
	_, err := qu.cli.Put(ctx, key, string(val))
	return err
}

func (qu *Queue) delete(ctx context.Context, key string) error {
	_, err := qu.cli.Delete(ctx, key)
	return err
}

// Get fetches an item from the queue. Use when watch has failed.
func (qu *Queue) Get(ctx context.Context, it *Item) (*Item, error) {
	qu.mu.RLock()
	defer qu.mu.RUnlock()

	key := it.Key
	resp, err := qu.cli.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	switch {
	case len(resp.Kvs) == 1:
	default:
		return nil, fmt.Errorf("%q should have 1 key (%+v)", key, resp.Kvs)
	}

	val := resp.Kvs[0].Value
	var v Value
	if err = json.Unmarshal(val, &v); err != nil {
		return nil, fmt.Errorf("cannot parse key: %q, val: %q (%v)", key, string(val), err)
	}

	it2 := *it
	it2.Value = v
	return &it2, nil
}

// Value contains status and any data.
type Value struct {
	Key        string     `json:"key"`
	StatusCode StatusCode `json:"status-code"`
	Error      string     `json:"error"`
	Data       []byte     `json:"value"`
}

// Item is a job item.
type Item struct {
	// Bucket is the name or job category for namespacing.
	// All keys will be prefixed with this bucket name.
	Bucket string

	// Weight is the priority of an item.
	// The maximum weight(priority) is 99999.
	Weight uint64

	// UnixNano is the unix nanosecond when the item is created.
	UnixNano int64

	// Key is the item ID.
	// This is the key when stored in etcd.
	Key string

	// Value is the data to be stored in etcd.
	Value Value

	// Error contains any error.
	Error error
}

// CreateItem creates an item, generating an ID.
// The maximum weight(priority) is 99999.
func CreateItem(bucket string, weight uint64, data []byte) (*Item, error) {
	if weight > 99999 {
		weight = 99999
	}
	unixNano := time.Now().UnixNano()
	id := path.Join(pfxScheduled, bucket, fmt.Sprintf("%05d%035X", weight, unixNano))
	return &Item{
		Bucket:   bucket,
		Weight:   weight,
		UnixNano: unixNano,
		Key:      id,
		Value:    Value{Key: id, Error: "", Data: data},
	}, nil
}

// ParseItem parses the ID.
func ParseItem(key string, val []byte) (*Item, error) {
	oid := key
	if !strings.HasPrefix(key, pfxScheduled) {
		return nil, fmt.Errorf("%q does not have schedule-prefix %q", key, pfxScheduled)
	}
	key = strings.Replace(key, pfxScheduled+"/", "", 1)
	bucket := path.Dir(key)

	key = strings.Replace(key, bucket+"/", "", 1)
	weight, err := strconv.ParseUint(key[:5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse weight %q (given %q)", key[:5], oid)
	}

	key = strings.Replace(key, key[:5], "", 1)
	unixNano, err := strconv.ParseInt(key, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse unix nano %q (given %q)", key, oid)
	}

	var v Value
	if err = json.Unmarshal(val, &v); err != nil {
		return nil, err
	}
	return &Item{
		Bucket:   bucket,
		Weight:   weight,
		UnixNano: unixNano,
		Key:      oid,
		Value:    v,
	}, nil
}

// Items is a list of Item.
type Items []*Item

func (s Items) Len() int      { return len(s) }
func (s Items) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s Items) Less(i, j int) bool {
	// highest weight first, then unix-nano ascending order
	return s[i].Weight > s[j].Weight ||
		(s[i].Weight == s[j].Weight && s[i].UnixNano < s[j].UnixNano)
}

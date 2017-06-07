// Package etcdqueue implements queue service backed by etcd.
package etcdqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/etcdserver/api/v3client"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/golang/glog"
)

// Queue is the queue service backed by etcd.
type Queue interface {
	// ClientEndpoints returns the client endpoints.
	ClientEndpoints() []string

	// Client returns the client.
	Client() *clientv3.Client

	// Stop stops the queue and its clients.
	Stop()

	// Add adds an item to the queue.
	// Updates are sent to the returned channel.
	// And the channel is closed after key deletion,
	// which is the last event when the job is completed.
	Add(ctx context.Context, it *Item) (<-chan *Item, error)
}

const (
	pfxScheduled = "queue_scheduled" // requested by client, added on queue
	pfxWorker    = "queue_worker"    // ready/in-progress in worker process
	pfxCompleted = "queue_completed" // finished by worker

	// progress value 100 means that the job is done!
	maxProgress = 100
)

type queue struct {
	cli        *clientv3.Client
	rootCtx    context.Context
	rootCancel func()
	buckets    map[string]chan error
}

// NewQueue creates a new queue from given etcd client.
func NewQueue(cli *clientv3.Client) Queue {
	ctx, cancel := context.WithCancel(context.Background())
	return &queue{
		cli:        cli,
		rootCtx:    ctx,
		rootCancel: cancel,
		buckets:    make(map[string]chan error),
	}
}

func (qu *queue) ClientEndpoints() []string { return qu.cli.Endpoints() }
func (qu *queue) Client() *clientv3.Client  { return qu.cli }

func (qu *queue) Stop() {
	glog.Info("stopping queue")

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

	glog.Info("stopped queue")
}

func (qu *queue) Add(ctx context.Context, it *Item) (<-chan *Item, error) {
	key := it.Key
	val, err := json.Marshal(it)
	if err != nil {
		return nil, err
	}

	err = qu.put(ctx, key, val)
	if err != nil {
		return nil, err
	}

	if _, ok := qu.buckets[it.Bucket]; !ok { // first job in the bucket, so schedule right away
		if err = qu.put(ctx, path.Join(pfxWorker, it.Bucket), val); err != nil {
			return nil, err
		}
		qu.buckets[it.Bucket] = make(chan error, 1)

		go qu.run(qu.rootCtx, it.Bucket, qu.buckets[it.Bucket])
	}

	wch := qu.cli.Watch(ctx, key, clientv3.WithPrevKV())

	// TODO: configurable?
	ch := make(chan *Item, 100)

	item := *it

	// watch until it's done, close on delete/error event at the end
	go func() {
		for {
			select {
			case wresp := <-wch:
				if len(wresp.Events) != 1 {
					item.Error = fmt.Errorf("%q expects 1 event from watch, got %+v", key, wresp.Events)
					ch <- &item
					close(ch)
					return
				}
				if wresp.Events[0].Type == mvccpb.DELETE {
					glog.Infof("%q has been deleted, thus completed", key)
					if wresp.Events[0].PrevKv != nil {
						item.Value = wresp.Events[0].PrevKv.Value
					}
					ch <- &item
					close(ch)
					return
				}
				if err := json.Unmarshal(wresp.Events[0].Kv.Value, &item); err != nil {
					item.Error = fmt.Errorf("cannot parse %s", string(wresp.Events[0].Kv.Value))
					ch <- &item
					close(ch)
					return
				}

				ch <- &item
				if item.Error != nil {
					glog.Warningf("watched item contains error %v", item.Error)
					close(ch)
					return
				}
				glog.Infof("%q has been updated", key)

			case <-ctx.Done():
				item.Error = ctx.Err()
				ch <- &item
				close(ch)
				return
			}
		}
	}()
	return ch, nil
}

// watchWorker watches on the queue and schedules the jobs.
// Point is never miss events, thus one routine must always watch path.Join(pfxWorker, bucket)
// 1. blocks until TODO job is done, notified via watch events
// 2. notify the client back with the new results on the key (Key field in Item)
// 3. delete the DONE key from the queue, and move to pfxCompleted + Key for logging
// 4. fetch one new job from path.Join(pfxScheduled, bucket)
// 5. skip if there is no job to schedule
// 6. write this job to path.Join(pfxWorker, bucket)
// 7. drain watch events for this wrtie
// repeat!
func (qu *queue) watchWorker(ctx context.Context, bucket string) error {
	keyToWatch := path.Join(pfxWorker, bucket)
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
			var item Item
			if err := json.Unmarshal(valBytes, &item); err != nil {
				return fmt.Errorf("%q returned wrong JSON value %q (%v)", keyToWatch, string(valBytes), err)
			}
			if item.Progress < maxProgress {
				glog.Infof("%q is in progress %d / %d (continue)", item.Key, item.Progress, maxProgress)
				continue
			}

			// 2. notify the client back with the new results on the key (ID field in Item)
			glog.Infof("%q is done", item.Key)
			if err := qu.put(ctx, item.Key, valBytes); err != nil {
				return err
			}

			// 3. delete the DONE key from the queue, and move to pfxCompleted + Key for logging
			glog.Infof("%q is deleted", item.Key)
			if err := qu.delete(ctx, item.Key); err != nil {
				return err
			}
			cKey := path.Join(pfxCompleted, item.Key)
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
			var newItem Item
			if err := json.Unmarshal(fetchBytes, &newItem); err != nil {
				return fmt.Errorf("%q has wrong JSON %q", resp.Kvs[0].Key, string(fetchBytes))
			}
			if newItem.Progress != 0 {
				return fmt.Errorf("%q must have initial progress, got %d", newItem.Key, newItem.Progress)
			}

			// 6. write this job to path.Join(pfxWorker, bucket)
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
				var item Item
				if err := json.Unmarshal(valBytes, &item); err != nil {
					return fmt.Errorf("%q returned wrong JSON value %q (%v)", keyToWatch, string(valBytes), err)
				}
				if item.Progress != 0 {
					return fmt.Errorf("%q has wrong progress %d, expected initial progress 0", keyToWatch, item.Progress)
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

func (qu *queue) run(ctx context.Context, bucket string, errc chan error) {
	defer close(errc)

	for {
		if err := qu.watchWorker(ctx, bucket); err != nil {
			if err == context.Canceled {
				errc <- err
				return
			}
			glog.Warning(err)
		}
		// below is implemented for failure tolerance; retry logic

		keyToWatch := path.Join(pfxWorker, bucket)
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
		var item Item
		if err = json.Unmarshal(valBytes, &item); err != nil {
			errc <- err
			return
		}
		if item.Progress == maxProgress {
			glog.Warningf("watch might have failed after %q is finished", item.Key)

			// 2. notify the client back with the new results on the key (ID field in Item)
			glog.Infof("%q is done", item.Key)
			if err := qu.put(ctx, item.Key, valBytes); err != nil {
				errc <- err
				return
			}

			// 3. delete the DONE key from the queue, and move to pfxCompleted + Key for logging
			glog.Infof("%q is deleted", item.Key)
			if err := qu.delete(ctx, item.Key); err != nil {
				errc <- err
				return
			}
			cKey := path.Join(pfxCompleted, item.Key)
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
			var newItem Item
			if err := json.Unmarshal(fetchBytes, &newItem); err != nil {
				errc <- fmt.Errorf("%q has wrong JSON %q", resp.Kvs[0].Key, string(fetchBytes))
				return
			}
			if newItem.Progress != 0 {
				errc <- fmt.Errorf("%q must have initial progress, got %d", newItem.Key, newItem.Progress)
				return
			}

			// 6. write this job to path.Join(pfxWorker, bucket)
			glog.Infof("%q is scheduled", string(resp.Kvs[0].Key))
			if err := qu.put(ctx, keyToWatch, resp.Kvs[0].Value); err != nil {
				errc <- err
				return
			}

			// continue to 'watchWorker'
		}
	}
}

func (qu *queue) put(ctx context.Context, key string, val []byte) error {
	_, err := qu.cli.Put(ctx, key, string(val))
	return err
}

func (qu *queue) delete(ctx context.Context, key string) error {
	_, err := qu.cli.Delete(ctx, key)
	return err
}

// embeddedQueue implements Queue interface with a single-node embedded etcd cluster.
type embeddedQueue struct {
	srv *embed.Etcd
	Queue
}

// NewEmbeddedQueue starts a new embedded etcd server.
// cport is the TCP port used for etcd client request serving.
// pport is for etcd peer traffic, and still needed even if it's a single-node cluster.
func NewEmbeddedQueue(cport, pport int, dataDir string) (Queue, error) {
	cfg := embed.NewConfig()
	cfg.ClusterState = embed.ClusterStateFlagNew

	cfg.Name = "etcd-queue"
	cfg.Dir = dataDir

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
	srv, err := embed.StartEtcd(cfg)
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
	return &embeddedQueue{
		srv: srv,
		Queue: &queue{
			cli:        cli,
			rootCtx:    ctx,
			rootCancel: cancel,
			buckets:    make(map[string]chan error),
		},
	}, err
}

func (qu *embeddedQueue) ClientEndpoints() []string {
	eps := make([]string, len(qu.srv.Config().LCUrls))
	for i := range qu.srv.Config().LCUrls {
		eps = append(eps, qu.srv.Config().LCUrls[i].String())
	}
	return eps
}

func (qu *embeddedQueue) Stop() {
	glog.Info("stopping queue with an embedded etcd server")
	qu.Queue.Stop()
	qu.srv.Close()
	glog.Info("stopped queue with an embedded etcd server")
}

// Item is a job item.
// Key is used as a key in etcd.
// Marshalled JSON struct data as a value.
type Item struct {
	// Bucket is the name or job category for namespacing.
	// All keys will be prefixed with this bucket name.
	Bucket string `json:"bucket"`

	// Key is autogenerated and used as a key when written to etcd.
	Key   string `json:"key"`
	Value []byte `json:"value"`

	// Progress is the progress status value.
	// 100 means it's done.
	Progress int   `json:"progress"`
	Error    error `json:"error"`
}

// CreateItem creates an item with auto-generated ID. The ID uses unix
// nano seconds, so that items created later are added in order.
// The maximum weight(priority) is 99999.
func CreateItem(bucket string, weight uint64, value []byte) *Item {
	if weight > 99999 {
		weight = 99999
	}
	return &Item{
		Bucket:   bucket,
		Key:      path.Join(pfxScheduled, bucket, fmt.Sprintf("%05d%035X", weight, time.Now().UnixNano())),
		Value:    value,
		Progress: 0,
		Error:    nil,
	}
}

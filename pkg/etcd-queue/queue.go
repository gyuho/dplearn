// Package etcdqueue implements queue service backed by etcd.
package etcdqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/golang/glog"
)

// Queue is the queue service backed by etcd.
type Queue interface {
	// Enqueue adds/overwrites an item in the queue. Updates are to be
	// done by other external worker services. The worker first fetches
	// the first item via 'Front' method, and writes back with 'Enqueue'
	// method. Enqueue returns a channel that notifies any events on the
	// item. The channel is closed when the job is completed or canceled.
	Enqueue(ctx context.Context, it *Item, opts ...OpOption) ItemWatcher

	// Front returns ItemWatcher that returns the first item in the queue.
	// It blocks until there is at least one item to return.
	Front(ctx context.Context, bucket string) ItemWatcher

	// Dequeue deletes the item in the queue, whether the item is completed
	// or in progress. The item needs not be the first one in the queue.
	Dequeue(ctx context.Context, it *Item) error

	// Watch creates a item watcher, assuming that the job is already scheduled
	// by 'Enqueue' method. The returned channel is never closed until the
	// context is canceled.
	Watch(ctx context.Context, key string) ItemWatcher

	// Stop stops the queue service and any embedded clients.
	Stop()

	// Client returns the client.
	Client() *clientv3.Client

	// ClientEndpoints returns the client endpoints.
	ClientEndpoints() []string
}

const (
	pfxScheduled = "_schd" // requested by client, added to queue
	pfxCompleted = "_cmpl" // finished by worker
)

type queue struct {
	mu         sync.RWMutex
	cli        *clientv3.Client
	rootCtx    context.Context
	rootCancel func()
}

// NewQueue creates a new queue from given etcd client.
func NewQueue(cli *clientv3.Client) (Queue, error) {
	// issue linearized read to ensure leader election
	glog.Infof("GET request to endpoint %v", cli.Endpoints())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := cli.Get(ctx, "foo")
	cancel()
	glog.Infof("GET request succeeded on endpoint %v", cli.Endpoints())
	if err != nil {
		return nil, err
	}

	ctx, cancel = context.WithCancel(context.Background())
	return &queue{
		cli:        cli,
		rootCtx:    ctx,
		rootCancel: cancel,
	}, nil
}

func (qu *queue) Enqueue(ctx context.Context, item *Item, opts ...OpOption) ItemWatcher {
	ret := Op{}
	ret.applyOpts(opts)

	// TODO: make this configurable
	ch := make(chan *Item, 100)

	if item == nil {
		ch <- &Item{Error: "received <nil> Item"}
		close(ch)
		return ch
	}

	cur := *item
	key := path.Join(pfxScheduled, cur.Key)

	data, err := json.Marshal(&cur)
	if err != nil {
		cur.Error = err.Error()
		ch <- &cur
		close(ch)
		return ch
	}
	val := string(data)

	qu.mu.Lock()
	defer qu.mu.Unlock()

	if err = qu.put(ctx, key, val, ret.ttl); err != nil {
		cur.Error = err.Error()
		ch <- &cur
		close(ch)
		return ch
	}
	glog.Infof("enqueue: wrote %q", item.Key)

	if cur.Progress == MaxProgress {
		if err = qu.delete(ctx, key); err != nil {
			cur.Error = err.Error()
			ch <- &cur
			close(ch)
			return ch
		}

		if err := qu.put(ctx, path.Join(pfxCompleted, cur.Key), val, 0); err != nil {
			cur.Error = err.Error()
			ch <- &cur
			close(ch)
			return ch
		}

		glog.Infof("enqueue: %q is finished", cur.Key)
		ch <- &cur
		close(ch)
		return ch
	}

	wch := qu.cli.Watch(ctx, key, clientv3.WithPrevKV())
	go func() {
		defer close(ch)

		for {
			select {
			case wresp := <-wch:
				if len(wresp.Events) != 1 {
					cur.Error = fmt.Sprintf("enqueue-watcher: %q expects 1 event from watch, got %+v", cur.Key, wresp.Events)
					ch <- &cur
					return
				}
				if wresp.Err() != nil {
					cur.Error = fmt.Sprintf("enqueue-watcher: %q returned error %v", cur.Key, wresp.Err())
					ch <- &cur
					return
				}

				if wresp.Canceled || wresp.Events[0].Type == mvccpb.DELETE {
					glog.Infof("enqueue-watcher: %q has been deleted; either completed or canceled", cur.Key)
					var prev Item
					if err := json.Unmarshal(wresp.Events[0].PrevKv.Value, &prev); err != nil {
						prev.Error = fmt.Sprintf("enqueue-watcher: cannot parse %q", string(wresp.Events[0].PrevKv.Value))
						ch <- &prev
						return
					}

					if prev.Progress != 100 {
						prev.Canceled = true
						glog.Infof("enqueue-watcher: found %q progress is only %d (canceled)", prev.Key, prev.Progress)
					}

					ch <- &prev
					return
				}

				if err := json.Unmarshal(wresp.Events[0].Kv.Value, &cur); err != nil {
					cur.Error = fmt.Sprintf("enqueue-watcher: cannot parse %q (%v)", string(wresp.Events[0].Kv.Value), err)
					ch <- &cur
					return
				}

				ch <- &cur
				if cur.Error != "" {
					glog.Warningf("enqueue-watcher: %q contains error %v", cur.Key, cur.Error)
					return
				}
				if cur.Progress == 100 {
					glog.Infof("enqueue-watcher: %q is finished", cur.Key)
					return
				}
				glog.Infof("enqueue-watcher: %q has been updated (waiting for next updates)", cur.Key)

			case <-ctx.Done():
				cur.Error = ctx.Err().Error()
				ch <- &cur
				return
			}
		}
	}()
	return ch
}

func (qu *queue) Front(ctx context.Context, bucket string) ItemWatcher {
	scheduledKey := path.Join(pfxScheduled, bucket)
	ch := make(chan *Item, 1)

	resp, err := qu.cli.Get(ctx, scheduledKey, clientv3.WithFirstKey()...)
	if err != nil {
		ch <- &Item{Error: err.Error()}
		close(ch)
		return ch
	}

	if len(resp.Kvs) == 0 {
		wch := qu.cli.Watch(ctx, scheduledKey, clientv3.WithPrefix())
		go func() {
			defer close(ch)

			select {
			case wresp := <-wch:
				if len(wresp.Events) != 1 {
					ch <- &Item{Error: fmt.Sprintf("%q did not return 1 event via watch (got %+v)", scheduledKey, wresp)}
					return
				}
				if wresp.Err() != nil {
					ch <- &Item{Error: fmt.Sprintf("%q returned error %v", scheduledKey, wresp.Err())}
					return
				}
				if wresp.Canceled || wresp.Events[0].Type == mvccpb.DELETE {
					ch <- &Item{Error: fmt.Sprintf("%q watch has been canceled or deleted", scheduledKey)}
					return
				}

				v := wresp.Events[0].Kv.Value
				var item Item
				if err := json.Unmarshal(v, &item); err != nil {
					ch <- &Item{Error: fmt.Sprintf("%q returned wrong JSON value %q (%v)", scheduledKey, string(v), err)}
					return
				}
				ch <- &item

			case <-ctx.Done():
				ch <- &Item{Error: ctx.Err().Error()}
			}
		}()
		return ch
	}

	if len(resp.Kvs) != 1 {
		ch <- &Item{Error: fmt.Sprintf("%q returned more than 1 key", scheduledKey)}
		close(ch)
		return ch
	}
	v := resp.Kvs[0].Value
	var item Item
	if err := json.Unmarshal(v, &item); err != nil {
		ch <- &Item{Error: fmt.Sprintf("%q returned wrong JSON value %q (%v)", scheduledKey, string(v), err)}
		close(ch)
	} else {
		ch <- &item
	}
	return ch
}

func (qu *queue) Dequeue(ctx context.Context, it *Item) error {
	key := path.Join(pfxScheduled, it.Key)

	qu.mu.Lock()
	defer qu.mu.Unlock()

	glog.Infof("dequeue-ing %q", key)
	if err := qu.delete(ctx, key); err != nil {
		return err
	}
	glog.Infof("dequeue-ed %q", key)
	return nil
}

func (qu *queue) Watch(ctx context.Context, key string) ItemWatcher {
	glog.Infof("watch: started watching on %q", key)

	key = path.Join(pfxScheduled, key)
	ch := make(chan *Item, 100)

	wch := qu.cli.Watch(ctx, key)
	go func() {
		for {
			select {
			case wresp := <-wch:
				if len(wresp.Events) != 1 {
					ch <- &Item{Error: fmt.Sprintf("watch: %q did not return 1 event via watch (got %+v)", key, wresp)}
					continue
				}
				if wresp.Err() != nil {
					ch <- &Item{Error: fmt.Sprintf("watch: %q returned error %v", key, wresp.Err())}
					return
				}
				if wresp.Canceled || wresp.Events[0].Type == mvccpb.DELETE {
					ch <- &Item{Error: fmt.Sprintf("watch: %q has been canceled or deleted", key)}
					return
				}

				glog.Infof("watch: received event on %q", key)
				v := wresp.Events[0].Kv.Value
				var item Item
				if err := json.Unmarshal(v, &item); err != nil {
					ch <- &Item{Error: fmt.Sprintf("watch: %q returned wrong JSON value %q (%v)", key, string(v), err)}
				} else {
					ch <- &item
					glog.Infof("watch: sent event on %q", key)
				}

			case <-ctx.Done():
				glog.Infof("watch: canceled on %q (closing channel)", key)
				close(ch)
				return
			}
		}
	}()

	return ch
}

func (qu *queue) Stop() {
	qu.mu.Lock()
	defer qu.mu.Unlock()

	glog.Info("stopping queue")
	qu.rootCancel()
	qu.cli.Close()
	glog.Info("stopped queue")
}

func (qu *queue) Client() *clientv3.Client {
	return qu.cli
}

func (qu *queue) ClientEndpoints() []string {
	return qu.cli.Endpoints()
}

func (qu *queue) put(ctx context.Context, key, val string, ttl int64) error {
	var opts []clientv3.OpOption
	if ttl > 5 {
		resp, err := qu.cli.Grant(ctx, ttl)
		if err != nil {
			return err
		}
		leaseID := resp.ID
		opts = append(opts, clientv3.WithLease(leaseID))
	}
	_, err := qu.cli.Put(ctx, key, val, opts...)
	return err
}

func (qu *queue) delete(ctx context.Context, key string) error {
	_, err := qu.cli.Delete(ctx, key)
	return err
}

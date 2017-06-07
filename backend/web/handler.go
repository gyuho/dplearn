package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"sync"

	etcdqueue "github.com/gyuho/deephardway/pkg/etcd-queue"
)

var (
	wordPredictMu       sync.RWMutex
	wordPredictRequests = make(map[string]*etcdqueue.Item)
)

func wordPredictHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodPost:
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

		userID := ctx.Value(userKey).(string)
		requestID := userID + req.URL.Path

		wordPredictMu.RLock()
		item, ok := wordPredictRequests[requestID]
		wordPredictMu.RUnlock()
		if ok {
			return json.NewEncoder(w).Encode(item)
		}

		// 1. create a new Item
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
				Error:    fmt.Errorf("Queue.Add error %q at %s", err.Error(), time.Now().String()[:29]),
			})
		}

		// 3. watch for changes for later request polling
		wordPredictMu.Lock()
		wordPredictRequests[requestID] = item
		wordPredictMu.Unlock()

		go watch(ctx, requestID, item, creq, ch)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

var (
	catsVsDogsMu       sync.RWMutex
	catsVsDogsRequests = make(map[string]*etcdqueue.Item)
)

func catsVsDogsHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodPost:
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

		userID := ctx.Value(userKey).(string)
		requestID := userID + req.URL.Path

		catsVsDogsMu.RLock()
		item, ok := catsVsDogsRequests[requestID]
		catsVsDogsMu.RUnlock()
		if ok {
			return json.NewEncoder(w).Encode(item)
		}

		// 1. create a new Item
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
				Error:    fmt.Errorf("Queue.Add error %q at %s", err.Error(), time.Now().String()[:29]),
			})
		}

		// 3. watch for changes for later request polling
		catsVsDogsMu.Lock()
		catsVsDogsRequests[requestID] = item
		catsVsDogsMu.Unlock()

		go watch(ctx, requestID, item, creq, ch)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

var (
	mnistMu       sync.RWMutex
	mnistRequests = make(map[string]*etcdqueue.Item)
)

func mnistHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodPost:
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

		userID := ctx.Value(userKey).(string)
		requestID := userID + req.URL.Path

		mnistMu.RLock()
		item, ok := mnistRequests[requestID]
		mnistMu.RUnlock()
		if ok {
			return json.NewEncoder(w).Encode(item)
		}

		// 1. create a new Item
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
				Error:    fmt.Errorf("Queue.Add error %q at %s", err.Error(), time.Now().String()[:29]),
			})
		}

		// 3. watch for changes for later request polling
		mnistMu.Lock()
		mnistRequests[requestID] = item
		mnistMu.Unlock()

		go watch(ctx, requestID, item, creq, ch)

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

		mnistMu.Lock()
		mnistRequests[requestID] = item
		mnistMu.Unlock()

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

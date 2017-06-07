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

		qu := ctx.Value(queueKey).(etcdqueue.Queue)
		userID := ctx.Value(userKey).(string)

		creq.UserID = userID
		creq.Result = ""
		rb, err = json.Marshal(creq)
		if err != nil {
			return err
		}
		item := etcdqueue.CreateItem("cats-vs-dogs", 100, string(rb))

		// TODO: write to queue
		fmt.Println("queue:", qu.ClientEndpoints())

		cnt := 0
		for item.Progress < 100 {
			// TODO: watch from queue until it's done
			time.Sleep(time.Second)
			creq.Result = fmt.Sprintf("Processing %+v at %s", creq, time.Now().String()[:29])
			rb, err = json.Marshal(creq)
			if err != nil {
				return err
			}
			item.Value = string(rb)
			item.Progress = (cnt + 1) * 10
			cnt++

			// received progress report from queue service
			if err := json.NewEncoder(w).Encode(item); err != nil {
				return err
			}
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

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

		qu := ctx.Value(queueKey).(etcdqueue.Queue)
		userID := ctx.Value(userKey).(string)

		creq.UserID = userID
		creq.Result = ""
		rb, err = json.Marshal(creq)
		if err != nil {
			return err
		}
		item := etcdqueue.CreateItem("cats-vs-dogs", 100, string(rb))

		// TODO: write to queue
		fmt.Println("queue:", qu.ClientEndpoints())

		cnt := 0
		for item.Progress < 100 {
			// TODO: watch from queue until it's done
			time.Sleep(time.Second)
			creq.Result = fmt.Sprintf("Processing %+v at %s", creq, time.Now().String()[:29])
			rb, err = json.Marshal(creq)
			if err != nil {
				return err
			}
			item.Value = string(rb)
			item.Progress = (cnt + 1) * 10
			cnt++

			// received progress report from queue service
			if err := json.NewEncoder(w).Encode(item); err != nil {
				return err
			}
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

var (
	rmu      sync.RWMutex
	requests = make(map[string]*etcdqueue.Item)
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

		qu := ctx.Value(queueKey).(etcdqueue.Queue)
		userID := ctx.Value(userKey).(string)
		requestID := userID + req.URL.Path

		rmu.RLock()
		item, ok := requests[requestID]
		if !ok {
			creq.UserID = userID
			creq.Result = ""
			rb, err = json.Marshal(creq)
			if err != nil {
				rmu.RUnlock()
				return err
			}
			item = etcdqueue.CreateItem("cats-vs-dogs", 100, string(rb))
		}
		rmu.RUnlock()

		go func(ctx context.Context, id string, item *etcdqueue.Item, req Request) {
			cnt := 0
			for item.Progress < 100 {
				// TODO: watch from queue until it's done, feed from queue service
				time.Sleep(time.Second)
				req.Result = fmt.Sprintf("Processing %+v at %s", req, time.Now().String()[:29])
				rb, err = json.Marshal(req)
				if err != nil {
					panic(err)
				}
				item.Value = string(rb)
				item.Progress = (cnt + 1) * 10
				cnt++

				rmu.Lock()
				requests[requestID] = item
				rmu.Unlock()

				select {
				case <-ctx.Done():
					return
				}
			}
		}(ctx, requestID, item, creq)

		// TODO: write to queue
		fmt.Println("queue:", requestID, qu.ClientEndpoints())
		fmt.Printf("WRITING: %+v\n", item)
		if err := json.NewEncoder(w).Encode(item); err != nil {
			return err
		}
		fmt.Printf("WROTE: %+v\n", item)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

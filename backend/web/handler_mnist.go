package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	etcdqueue "github.com/gyuho/deephardway/pkg/etcd-queue"
)

func mnistHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodPost:
		rb, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		req.Body.Close()

		qu := ctx.Value(queueKey).(etcdqueue.Queue)
		userID := ctx.Value(userKey).(string)

		creq := Request{}
		if err = json.Unmarshal(rb, &creq); err != nil {
			return json.NewEncoder(w).Encode(&etcdqueue.Item{
				Progress: 0,
				Error:    fmt.Errorf("JSON parse error %q at %s", err.Error(), time.Now().String()[:29]),
			})
		}
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
			fmt.Printf("WRITING: %+v\n", item)
			if err := json.NewEncoder(w).Encode(item); err != nil {
				return err
			}
			fmt.Printf("WROTE: %+v\n", item)
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

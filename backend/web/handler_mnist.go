package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// MNISTRequest defines 'mnist' requests.
type MNISTRequest struct {
	URL     int    `json:"url"`
	RawData string `json:"raw-data"`
}

// MNISTResponse is the response from server.
type MNISTResponse struct {
	Result string `json:"result"`
}

func mnistHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodPost:
		cresp := MNISTResponse{Result: ""}

		creq := MNISTRequest{}
		if err := json.NewDecoder(req.Body).Decode(&creq); err != nil {
			cresp.Result = err.Error()
			return json.NewEncoder(w).Encode(cresp)
		}
		defer req.Body.Close()

		cresp.Result = "Response at " + time.Now().String()[:29]
		if err := json.NewEncoder(w).Encode(cresp); err != nil {
			return err
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}
	return nil
}

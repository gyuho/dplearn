package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// WordPredictRequest defines 'word-predict' requests.
type WordPredictRequest struct {
	Type int    `json:"type"`
	Text string `json:"text"`
}

// WordPredictResponse is the response from server.
type WordPredictResponse struct {
	Result string `json:"result"`
}

func wordPredictHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodPost:
		cresp := WordPredictResponse{Result: ""}

		creq := WordPredictRequest{}
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

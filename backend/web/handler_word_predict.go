package web

import (
	"context"
	"encoding/json"
	"fmt"
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
		resp := WordPredictResponse{Result: ""}

		rreq := WordPredictRequest{}
		if err := json.NewDecoder(req.Body).Decode(&rreq); err != nil {
			resp.Result = fmt.Sprintf("JSON parse error %q at %s", err.Error(), time.Now().String()[:29])
			return json.NewEncoder(w).Encode(resp)
		}
		defer req.Body.Close()

		resp.Result = fmt.Sprintf("Received %+v at %s", rreq, time.Now().String()[:29])
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			return err
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

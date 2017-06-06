package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// CatsVsDogsRequest defines 'cats-vs-dogs' requests.
type CatsVsDogsRequest struct {
	URL     int    `json:"url"`
	RawData string `json:"raw-data"`
}

// CatsVsDogsResponse is the response from server.
type CatsVsDogsResponse struct {
	Result string `json:"result"`
}

func catsVsDogsHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodPost:
		cresp := CatsVsDogsResponse{Result: ""}

		creq := CatsVsDogsRequest{}
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

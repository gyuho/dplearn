package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// CatsAndDogsRequest defines 'cats-and-dogs' requests.
type CatsAndDogsRequest struct {
	URL     int    `json:"url"`
	RawData string `json:"raw-data"`
}

// CatsAndDogsResponse is the response from server.
type CatsAndDogsResponse struct {
	Result string `json:"result"`
}

func catsAndDogsHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodPost:
		cresp := CatsAndDogsResponse{Result: ""}

		creq := CatsAndDogsRequest{}
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

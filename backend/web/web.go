package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
)

// Server warps http.Server.
type Server struct {
	mu         sync.RWMutex
	rootCtx    context.Context
	rootCancel func()
	addrURL    url.URL
	httpServer *http.Server

	donec chan struct{}
}

// StartServer starts a backend webserver with stoppable listener.
func StartServer(port int) (*Server, error) {
	rootCtx, rootCancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()
	mux.Handle("/client-request", &ContextAdapter{
		ctx:     rootCtx,
		handler: ContextHandlerFunc(spellCheckHandler),
	})

	addrURL := url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", port)}
	srv := &Server{
		rootCtx:    rootCtx,
		rootCancel: rootCancel,
		addrURL:    addrURL,
		httpServer: &http.Server{Addr: addrURL.Host, Handler: mux},
		donec:      make(chan struct{}),
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				glog.Fatal(err)
				os.Exit(0)
			}
			srv.rootCancel()
		}()

		glog.Infof("starting server %q", srv.addrURL.String())
		if err := srv.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			glog.Fatal(err)
		}

		select {
		case <-srv.donec:
		default:
			close(srv.donec)
		}
	}()
	return srv, nil
}

// Stop stops the server. Useful for testing.
func (srv *Server) Stop() error {
	glog.Infof("stopping server %q", srv.addrURL.String())

	srv.mu.Lock()
	if srv.httpServer == nil {
		srv.mu.Unlock()
		glog.Infof("already stopped %q", srv.addrURL.String())
		return nil
	}

	ctx, cancel := context.WithTimeout(srv.rootCtx, 5*time.Second)
	err := srv.httpServer.Shutdown(ctx)
	cancel()
	if err != nil {
		return err
	}

	srv.httpServer = nil
	srv.mu.Unlock()

	glog.Infof("stopped server %q", srv.addrURL.String())
	return nil
}

// StopNotify returns receive-only stop channel to notify the server has stopped.
func (srv *Server) StopNotify() <-chan struct{} {
	return srv.donec
}

// SpellCheckRequest defines client requests.
type SpellCheckRequest struct {
	Text string `json:"text"`
}

// SpellCheckResponse is the response from server.
type SpellCheckResponse struct {
	Text   string `json:"text"`
	Result string `json:"result"`
}

func spellCheckHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodPost:
		cresp := SpellCheckResponse{Text: "", Result: ""}

		creq := SpellCheckRequest{}
		if err := json.NewDecoder(req.Body).Decode(&creq); err != nil {
			cresp.Text = ""
			cresp.Result = err.Error()
			return json.NewEncoder(w).Encode(cresp)
		}
		defer req.Body.Close()

		cresp.Text = creq.Text
		cresp.Result = "Response at " + time.Now().String()[:29]
		if err := json.NewEncoder(w).Encode(cresp); err != nil {
			return err
		}

	default:
		http.Error(w, "Method Not Allowed", 405)
	}
	return nil
}

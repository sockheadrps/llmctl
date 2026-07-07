package statusserver

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
)

// Server is a small HTTP server that serves GET /status as JSON.
// The caller updates the snapshot via SetStatus; the server always
// serves the latest snapshot it has been given.
type Server struct {
	mu     sync.RWMutex
	status Status
	srv    *http.Server
}

// NewServer creates a Server. Call Start to bind and begin serving.
func NewServer() *Server {
	return &Server{}
}

// SetStatus replaces the current snapshot atomically.
func (s *Server) SetStatus(st Status) {
	s.mu.Lock()
	s.status = st
	s.mu.Unlock()
}

// Start binds to host:port and serves in the background.
// Returns an error if the listener cannot be created.
func (s *Server) Start(host string, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		st := s.status
		s.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st)
	})

	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}

	s.srv = &http.Server{Handler: mux}
	go s.srv.Serve(ln) //nolint:errcheck
	return nil
}

// Stop shuts the server down gracefully.
func (s *Server) Stop() {
	if s.srv != nil {
		_ = s.srv.Shutdown(context.Background())
	}
}

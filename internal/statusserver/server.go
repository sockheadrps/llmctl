package statusserver

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Server is a small HTTP server that serves GET /status as JSON.
// The caller updates the snapshot via SetStatus; the server always
// serves the latest snapshot it has been given.
type Server struct {
	mu      sync.RWMutex
	status  Status
	srv     *http.Server
	clients map[string]time.Time // remote IP → last seen
}

// NewServer creates a Server. Call Start to bind and begin serving.
func NewServer() *Server {
	return &Server{clients: make(map[string]time.Time)}
}

// RecentClientCount returns the number of distinct remote IPs that have
// polled /status within the given window.
func (s *Server) RecentClientCount(window time.Duration) int {
	cutoff := time.Now().Add(-window)
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, t := range s.clients {
		if t.After(cutoff) {
			n++
		}
	}
	return n
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
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		s.mu.Lock()
		st := s.status
		if host != "" && host != "127.0.0.1" && host != "::1" {
			s.clients[host] = time.Now()
		}
		s.mu.Unlock()
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

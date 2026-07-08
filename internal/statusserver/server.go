package statusserver

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Server serves the local llmctl status snapshot and accepts pushed status
// snapshots from llmctl instances running as RPC clients.
type Server struct {
	mu            sync.RWMutex
	status        Status
	srv           *http.Server
	clientUpdates map[string]ClientInfo
	upgrader      websocket.Upgrader
}

// NewServer creates a Server. Call Start to bind and begin serving.
func NewServer() *Server {
	return &Server{
		clientUpdates: make(map[string]ClientInfo),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// SetStatus replaces the current local snapshot atomically.
func (s *Server) SetStatus(st Status) {
	s.mu.Lock()
	st.Clients = s.clientSnapshotsLocked(45 * time.Second)
	s.status = st
	s.mu.Unlock()
}

// ClientStatuses returns recently pushed RPC client snapshots.
func (s *Server) ClientStatuses(window time.Duration) []ClientInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clientSnapshotsLocked(window)
}

func (s *Server) clientSnapshotsLocked(window time.Duration) []ClientInfo {
	cutoff := time.Now().Add(-window).Unix()
	out := make([]ClientInfo, 0, len(s.clientUpdates))
	for _, client := range s.clientUpdates {
		if client.LastSeen >= cutoff {
			out = append(out, client)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].ID < out[j].ID
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Start binds to host:port and serves in the background.
func (s *Server) Start(host string, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.status.Clients = s.clientSnapshotsLocked(45 * time.Second)
		st := s.status
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st)
	})
	mux.HandleFunc("/ws/client-status", s.handleClientStatusWS)

	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}

	s.srv = &http.Server{Handler: mux}
	go s.srv.Serve(ln) //nolint:errcheck
	return nil
}

type clientUpdate struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Status Status `json:"status"`
}

func (s *Server) handleClientStatusWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	for {
		var update clientUpdate
		if err := conn.ReadJSON(&update); err != nil {
			return
		}
		if update.ID == "" {
			update.ID = host
		}
		info := ClientInfo{
			ID:       update.ID,
			Name:     update.Name,
			Addr:     host,
			LastSeen: time.Now().Unix(),
			Running:  update.Status.Running,
			GPU:      update.Status.GPU,
		}
		s.mu.Lock()
		s.clientUpdates[update.ID] = info
		s.mu.Unlock()
	}
}

// Stop shuts the server down gracefully.
func (s *Server) Stop() {
	if s.srv != nil {
		_ = s.srv.Shutdown(context.Background())
	}
}

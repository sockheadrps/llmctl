package statusserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Server serves the local llmctl status snapshot and accepts pushed status
// snapshots from llmctl instances running as RPC clients.
type Server struct {
	mu                sync.RWMutex
	status            Status
	history           []HistorySample
	historyLimit      int
	historyPath       string
	historyPersist    bool
	historyLoadedPath string
	srv               *http.Server
	clientUpdates     map[string]ClientInfo
	dashboardEnabled  bool
	upgrader          websocket.Upgrader
}

const defaultHistoryLimit = 240

// NewServer creates a Server. Call Start to bind and begin serving.
func NewServer() *Server {
	return &Server{
		clientUpdates:    make(map[string]ClientInfo),
		historyLimit:     defaultHistoryLimit,
		dashboardEnabled: true,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// ConfigureDashboard controls whether the browser dashboard is served.
func (s *Server) ConfigureDashboard(enabled bool) {
	s.mu.Lock()
	s.dashboardEnabled = enabled
	s.mu.Unlock()
}

// SetStatus replaces the current local snapshot atomically.
func (s *Server) SetStatus(st Status) {
	s.mu.Lock()
	s.recordStatusLocked(st)
	persist := s.historyPersist && s.historyPath != ""
	path := s.historyPath
	snapshot := History{Samples: append([]HistorySample(nil), s.history...)}
	s.mu.Unlock()
	if persist {
		_ = saveHistoryFile(path, snapshot)
	}
}

// ConfigureHistoryPersistence loads any saved history and sets the
// persistence destination for future updates.
func (s *Server) ConfigureHistoryPersistence(path string, enabled bool) error {
	path = strings.TrimSpace(path)
	if path == "" || !enabled {
		s.mu.Lock()
		s.historyPath = path
		s.historyPersist = false
		s.mu.Unlock()
		return nil
	}

	loaded, err := loadHistoryFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	s.mu.Lock()
	if s.historyLoadedPath != path {
		if len(loaded.Samples) > 0 {
			merged := append(append([]HistorySample(nil), loaded.Samples...), s.history...)
			if s.historyLimit > 0 && len(merged) > s.historyLimit {
				merged = merged[len(merged)-s.historyLimit:]
			}
			s.history = merged
		}
		s.historyLoadedPath = path
	}
	s.historyPath = path
	s.historyPersist = true
	snapshot := History{Samples: append([]HistorySample(nil), s.history...)}
	s.mu.Unlock()

	return saveHistoryFile(path, snapshot)
}

// History returns a copy of the most recent captured status snapshots.
func (s *Server) History() []HistorySample {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HistorySample, len(s.history))
	copy(out, s.history)
	return out
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

func (s *Server) recordStatusLocked(st Status) {
	st.Clients = s.clientSnapshotsLocked(45 * time.Second)
	s.status = st
	s.history = append(s.history, HistorySample{
		SampledAtMs: time.Now().UnixMilli(),
		Status:      st,
	})
	if s.historyLimit > 0 && len(s.history) > s.historyLimit {
		s.history = append([]HistorySample(nil), s.history[len(s.history)-s.historyLimit:]...)
	}
}

func (s *Server) statusLocked() Status {
	st := s.status
	st.Clients = s.clientSnapshotsLocked(45 * time.Second)
	return st
}

func (s *Server) historyLocked() History {
	out := make([]HistorySample, len(s.history))
	copy(out, s.history)
	return History{Samples: out}
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		s.mu.RLock()
		enabled := s.dashboardEnabled
		s.mu.RUnlock()
		if enabled {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/status", http.StatusFound)
	})
	mux.HandleFunc("/dashboard", s.handleDashboard)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/history", s.handleHistory)
	mux.HandleFunc("/ws/client-status", s.handleClientStatusWS)
	return mux
}

// Start binds to host:port and serves in the background.
func (s *Server) Start(host string, port int) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}

	s.srv = &http.Server{Handler: s.handler()}
	go s.srv.Serve(ln) //nolint:errcheck
	return nil
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	st := s.statusLocked()
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	history := s.historyLocked()
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(history)
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

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	enabled := s.dashboardEnabled
	s.mu.RUnlock()
	if !enabled {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, dashboardHTML)
}

func loadHistoryFile(path string) (History, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return History{}, err
	}
	var history History
	if err := json.Unmarshal(data, &history); err != nil {
		return History{}, err
	}
	if history.Samples == nil {
		history.Samples = []HistorySample{}
	}
	return history, nil
}

func saveHistoryFile(path string, history History) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if history.Samples == nil {
		history.Samples = []HistorySample{}
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Stop shuts the server down gracefully.
func (s *Server) Stop() {
	if s.srv != nil {
		_ = s.srv.Shutdown(context.Background())
	}
}

// Package runtime tracks running llama-server instances across separate
// CLI invocations by persisting them to a JSON state file, reconciling
// against actual OS process liveness on every read.
package runtime

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/process"
	"github.com/sockheadrps/llmctl/internal/util"
)

// Manager coordinates starting/stopping llama-server processes and
// persisting their state to disk.
type Manager struct {
	statePath    string
	rpcStatePath string
}

// NewManager creates a Manager backed by the default state file location.
func NewManager() (*Manager, error) {
	statePath, err := util.StateFile()
	if err != nil {
		return nil, err
	}
	rpcStatePath, err := util.RPCStateFile()
	if err != nil {
		return nil, err
	}
	return &Manager{statePath: statePath, rpcStatePath: rpcStatePath}, nil
}

// LogPath returns the deterministic log file location for a model+profile
// pair, matching where Start writes its output. Exported so callers (e.g.
// the TUI's log viewer) can find a profile's log without having to run it
// first.
func LogPath(modelKey, profileKey string) (string, error) {
	logDir, err := util.LogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(logDir, fmt.Sprintf("%s-%s.log", modelKey, profileKey)), nil
}

// List returns all instances currently believed to be running, pruning any
// whose PID is no longer alive.
func (mgr *Manager) List() ([]models.Running, error) {
	running, err := loadState(mgr.statePath)
	if err != nil {
		return nil, err
	}

	alive := make([]models.Running, 0, len(running))
	for _, r := range running {
		if process.IsAlive(r.PID) {
			alive = append(alive, r)
		}
	}

	if len(alive) != len(running) {
		if err := saveState(mgr.statePath, alive); err != nil {
			return nil, err
		}
	}

	return alive, nil
}

// Start launches modelKey/profileKey from cfg, records it in the state
// file, and returns the resulting Running entry. It refuses to start a
// model+profile pair that is already running.
// rpcEndpointOverride, when non-empty, is used as the --rpc endpoint instead
// of cfg.RPCEndpoint (e.g. an auto-discovered addr).
func (mgr *Manager) Start(cfg *config.Config, modelKey, profileKey string, rpcEndpointOverride string) (models.Running, error) {
	m, p, err := cfg.FindProfile(modelKey, profileKey)
	if err != nil {
		return models.Running{}, err
	}

	running, err := mgr.List()
	if err != nil {
		return models.Running{}, err
	}
	for _, r := range running {
		if r.ModelKey == modelKey && r.ProfileKey == profileKey {
			return models.Running{}, fmt.Errorf("%s / %s is already running on port %d", m.Name, p.Name, r.Port)
		}
	}

	logPath, err := LogPath(modelKey, profileKey)
	if err != nil {
		return models.Running{}, err
	}

	if resolvedPort, err := resolveLaunchPort(p.Host, p.Port); err != nil {
		return models.Running{}, err
	} else {
		p.Port = resolvedPort
	}

	rpcEndpoint := ""
	useRPC := cfg.RPCEnabled
	if p.RPCEnabled != nil {
		useRPC = *p.RPCEnabled
	}
	if useRPC {
		if strings.TrimSpace(rpcEndpointOverride) != "" {
			rpcEndpoint = strings.TrimSpace(rpcEndpointOverride)
		} else if strings.TrimSpace(cfg.RPCEndpoint) != "" {
			rpcEndpoint = strings.TrimSpace(cfg.RPCEndpoint)
		}
	}

	pid, err := process.Start(cfg.LlamaServerBin, m, p, logPath, rpcEndpoint)
	if err != nil {
		return models.Running{}, err
	}

	if err := awaitStableStart(pid, logPath); err != nil {
		return models.Running{}, fmt.Errorf("%s / %s failed to start: %w", m.Name, p.Name, err)
	}

	entry := models.Running{
		ModelKey:    modelKey,
		ModelName:   m.Name,
		ProfileKey:  profileKey,
		ProfileName: p.Name,
		Host:        p.Host,
		Port:        p.Port,
		PID:         pid,
		LogFile:     logPath,
		StartedAt:   time.Now().Unix(),
	}

	running = append(running, entry)
	if err := saveState(mgr.statePath, running); err != nil {
		return models.Running{}, err
	}

	recordRecent(modelKey, profileKey)

	return entry, nil
}

func resolveLaunchPort(host string, configured int) (int, error) {
	if configured > 0 && portAvailable(host, configured) {
		return configured, nil
	}
	return freePort(host)
}

func portAvailable(host string, port int) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort(listenHost(host), strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func freePort(host string) (int, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort(listenHost(host), "0"))
	if err != nil {
		return 0, fmt.Errorf("find free port: %w", err)
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("find free port: unexpected address %s", ln.Addr())
	}
	return addr.Port, nil
}

func listenHost(host string) string {
	if host == "" {
		return "127.0.0.1"
	}
	return host
}

// startGracePeriod is how long Start waits to confirm the process didn't
// die immediately (e.g. from a bad CLI argument) before declaring success.
// It doesn't wait for the model to finish loading, just for the process to
// survive past its own startup argument parsing.
const startGracePeriod = 2500 * time.Millisecond

// awaitStableStart polls pid for startGracePeriod. If the process exits
// during that window, it returns an error containing the tail of its log
// so the caller can see why (e.g. an invalid flag or missing model file).
func awaitStableStart(pid int, logPath string) error {
	deadline := time.Now().Add(startGracePeriod)
	for time.Now().Before(deadline) {
		if !process.IsAlive(pid) {
			if tail, err := process.TailLog(logPath, 8); err == nil && tail != "" {
				return fmt.Errorf("process exited immediately:\n%s", tail)
			}
			return fmt.Errorf("process exited immediately")
		}
		time.Sleep(150 * time.Millisecond)
	}
	return nil
}

// Stop terminates the running instance for modelKey/profileKey and removes
// it from the state file.
func (mgr *Manager) Stop(modelKey, profileKey string) error {
	running, err := mgr.List()
	if err != nil {
		return err
	}

	kept := make([]models.Running, 0, len(running))
	var target *models.Running
	for _, r := range running {
		if r.ModelKey == modelKey && r.ProfileKey == profileKey {
			r := r
			target = &r
			continue
		}
		kept = append(kept, r)
	}

	if target == nil {
		return fmt.Errorf("%s / %s is not running", modelKey, profileKey)
	}

	if err := process.Stop(target.PID); err != nil {
		return err
	}

	return saveState(mgr.statePath, kept)
}

// Find returns the Running entry for modelKey/profileKey, if any.
func (mgr *Manager) Find(modelKey, profileKey string) (models.Running, bool, error) {
	running, err := mgr.List()
	if err != nil {
		return models.Running{}, false, err
	}
	for _, r := range running {
		if r.ModelKey == modelKey && r.ProfileKey == profileKey {
			return r, true, nil
		}
	}
	return models.Running{}, false, nil
}

// StartRPCServer launches the ggml-rpc-server using cfg settings, saves
// its state, and returns the resulting RPCServerState.
func (mgr *Manager) StartRPCServer(cfg *config.Config) error {
	host := cfg.RPCServerHost
	port := cfg.RPCServerPort
	bin := cfg.RPCServerBin

	if pid, found, err := process.FindRPCServerPID(bin, host, port); err == nil && found {
		logDir, err := util.LogDir()
		if err != nil {
			return err
		}
		logPath := filepath.Join(logDir, "rpc-server.log")
		state := &RPCServerState{
			PID:       pid,
			Host:      host,
			Port:      port,
			LogFile:   logPath,
			StartedAt: time.Now(),
		}
		return SaveRPCState(mgr.rpcStatePath, state)
	}

	if !portAvailable(host, port) {
		return fmt.Errorf("port %d is already in use", port)
	}

	logDir, err := util.LogDir()
	if err != nil {
		return err
	}
	logPath := filepath.Join(logDir, "rpc-server.log")

	pid, err := process.StartRPC(bin, host, port, logPath)
	if err != nil {
		return err
	}

	state := &RPCServerState{
		PID:       pid,
		Host:      host,
		Port:      port,
		LogFile:   logPath,
		StartedAt: time.Now(),
	}

	if err := SaveRPCState(mgr.rpcStatePath, state); err != nil {
		return err
	}

	return nil
}

// StopRPCServer terminates the ggml-rpc-server process and clears its state.
func (mgr *Manager) StopRPCServer() error {
	state, err := LoadRPCState(mgr.rpcStatePath)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("rpc-server is not running")
	}

	if err := process.Stop(state.PID); err != nil {
		return err
	}

	return ClearRPCState(mgr.rpcStatePath)
}

// HasRPCStateFile reports whether a state file exists for the RPC server,
// regardless of whether the stored PID is still alive. Use this to
// distinguish "crashed/errored" (file exists, dead PID) from "not started"
// (no file at all).
func (mgr *Manager) HasRPCStateFile() bool {
	raw, _ := LoadRPCStateRaw(mgr.rpcStatePath)
	return raw != nil
}

// ClearRPCServer removes a stale RPC server state file without attempting
// to kill any process. Use this to reset the "down/errored" state when the
// process has already died on its own.
func (mgr *Manager) ClearRPCServer() error {
	return ClearRPCState(mgr.rpcStatePath)
}

// RPCServerLogPath returns the canonical log path for the RPC server,
// regardless of whether it has been started yet.
func RPCServerLogPath() (string, error) {
	logDir, err := util.LogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(logDir, "rpc-server.log"), nil
}

// RPCServerStatus returns the current RPC server state and whether it is
// believed to be running. Returns zero state and false if no state file
// exists or the PID is dead.
func (mgr *Manager) RPCServerStatus() (RPCServerState, bool) {
	state, err := LoadRPCState(mgr.rpcStatePath)
	if err != nil || state == nil {
		return RPCServerState{}, false
	}
	return *state, true
}

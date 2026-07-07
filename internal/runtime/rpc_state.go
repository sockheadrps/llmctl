package runtime

import (
	"encoding/json"
	"os"
	"time"

	"github.com/sockheadrps/llmctl/internal/process"
)

// RPCServerState tracks the lifecycle of the ggml-rpc-server process.
type RPCServerState struct {
	PID       int       `json:"pid"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	LogFile   string    `json:"log_file"`
	StartedAt time.Time `json:"started_at"`
}

// LoadRPCStateRaw reads the RPC server state from disk without checking
// whether the PID is still alive. Returns nil if no state file exists.
// Use this to distinguish "never started" (no file) from "crashed" (file
// with dead PID) — LoadRPCState returns nil for both.
func LoadRPCStateRaw(path string) (*RPCServerState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state RPCServerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// LoadRPCState reads the RPC server state from disk.
// A missing file is treated as no state (nil returned), not an error.
// Dead PIDs are pruned — if the stored PID is no longer alive, nil is returned.
func LoadRPCState(path string) (*RPCServerState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state RPCServerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if !process.IsAlive(state.PID) {
		return nil, nil
	}
	return &state, nil
}

// SaveRPCState writes the RPC server state to disk.
func SaveRPCState(path string, s *RPCServerState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ClearRPCState removes the RPC server state file.
func ClearRPCState(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

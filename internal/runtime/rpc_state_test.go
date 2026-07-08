package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRPCStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc_state.json")
	want := &RPCServerState{
		PID:       os.Getpid(), // current process — guaranteed alive
		Host:      "0.0.0.0",
		Port:      50052,
		LogFile:   "/tmp/rpc-server.log",
		StartedAt: time.Now().Truncate(time.Second),
	}

	if err := SaveRPCState(path, want); err != nil {
		t.Fatalf("SaveRPCState: %v", err)
	}

	got, err := LoadRPCState(path)
	if err != nil {
		t.Fatalf("LoadRPCState: %v", err)
	}
	if got == nil {
		t.Fatal("LoadRPCState returned nil for a live PID")
	}
	if got.PID != want.PID {
		t.Errorf("PID: got %d, want %d", got.PID, want.PID)
	}
	if got.Host != want.Host {
		t.Errorf("Host: got %q, want %q", got.Host, want.Host)
	}
	if got.Port != want.Port {
		t.Errorf("Port: got %d, want %d", got.Port, want.Port)
	}
}

func TestLoadRPCStateMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc_state.json")
	got, err := LoadRPCState(path)
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for missing file, got %+v", got)
	}
}

func TestLoadRPCStatePrunesDeadPID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc_state.json")

	// PID 0 is never a valid live process.
	dead := &RPCServerState{PID: 0, Host: "0.0.0.0", Port: 50052}
	data, _ := json.MarshalIndent(dead, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadRPCState(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for dead PID, got %+v", got)
	}
}

func TestClearRPCState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc_state.json")
	s := &RPCServerState{PID: os.Getpid(), Host: "0.0.0.0", Port: 50052}
	if err := SaveRPCState(path, s); err != nil {
		t.Fatal(err)
	}

	if err := ClearRPCState(path); err != nil {
		t.Fatalf("ClearRPCState: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected state file to be deleted")
	}
}

func TestClearRPCStateIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc_state.json")
	// File does not exist — ClearRPCState should not error.
	if err := ClearRPCState(path); err != nil {
		t.Fatalf("ClearRPCState on missing file: %v", err)
	}
}

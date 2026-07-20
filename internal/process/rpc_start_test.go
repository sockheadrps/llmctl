package process

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartRPCMissingBinaryReturnsHint(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "rpc.log")

	_, err := StartRPC("llmctl-definitely-missing-rpc-server", "0.0.0.0", 50052, logPath, nil)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}

	msg := err.Error()
	for _, want := range []string{"ggml-rpc-server binary", "rpc_server_bin", "PATH"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not contain %q", msg, want)
		}
	}
}

func TestStartRPCEmptyBinFallsBackToDefaultName(t *testing.T) {
	if _, err := exec.LookPath("ggml-rpc-server"); err == nil {
		t.Skip("ggml-rpc-server is present on PATH — skipping not-found test")
	}

	logPath := filepath.Join(t.TempDir(), "rpc.log")

	// Empty bin should fall back to "ggml-rpc-server" and fail with the
	// standard not-found message (not a generic exec error).
	_, err := StartRPC("", "0.0.0.0", 50052, logPath, nil)
	if err == nil {
		t.Fatal("expected error for missing fallback binary")
	}

	if !strings.Contains(err.Error(), "ggml-rpc-server") {
		t.Errorf("expected ggml-rpc-server in error, got: %v", err)
	}
}

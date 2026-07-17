package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

func setupTestController(t *testing.T) *Controller {
	t.Helper()
	// Controller requires a real runtime.Manager, so we create one
	// For unit tests, we'll test methods that don't require a running manager
	return New(nil)
}

func TestNewController(t *testing.T) {
	ctrl := setupTestController(t)
	if ctrl == nil {
		t.Fatal("controller should not be nil")
	}
}

func TestTailLog_Success(t *testing.T) {
	ctrl := setupTestController(t)

	// Create a test log file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Tail last 3 lines
	output, err := ctrl.TailLog(logPath, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "line 3") {
		t.Error("output should contain 'line 3'")
	}
	if !strings.Contains(output, "line 4") {
		t.Error("output should contain 'line 4'")
	}
	if !strings.Contains(output, "line 5") {
		t.Error("output should contain 'line 5'")
	}
}

func TestTailLog_FileNotFound(t *testing.T) {
	ctrl := setupTestController(t)

	output, err := ctrl.TailLog("/nonexistent/file.log", 10)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if output != "" {
		t.Error("output should be empty on error")
	}
}

func TestBuildProfileArgs(t *testing.T) {
	ctrl := setupTestController(t)

	port := 8080
	batch := 512
	profile := models.Profile{
		Name:      "test",
		Port:      port,
		BatchSize: &batch,
	}

	args := ctrl.BuildProfileArgs(&profile)
	if len(args) == 0 {
		t.Error("args should not be empty")
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "8080") {
		t.Errorf("args should contain port 8080, got: %s", argsStr)
	}
}

func TestGetRSSMiB_ZeroPID(t *testing.T) {
	ctrl := setupTestController(t)

	// PID 0 should return 0 RSS
	rss := ctrl.GetRSSMiB(0)
	if rss != 0 {
		t.Errorf("RSS for PID 0 should be 0, got: %d", rss)
	}
}

func TestParseModelLoadSlices_EmptyLog(t *testing.T) {
	ctrl := setupTestController(t)

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "empty.log")
	if err := os.WriteFile(logPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	devices, loads, err := ctrl.ParseModelLoadSlices(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("devices should be empty, got %d", len(devices))
	}
	if len(loads) != 0 {
		t.Errorf("loads should be empty, got %d", len(loads))
	}
}

func TestParseModelLoadSlices_NoData(t *testing.T) {
	ctrl := setupTestController(t)

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "no-gpu.log")
	if err := os.WriteFile(logPath, []byte("random log output\n"), 0644); err != nil {
		t.Fatal(err)
	}

	devices, loads, err := ctrl.ParseModelLoadSlices(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("devices should be empty, got %d", len(devices))
	}
	if len(loads) != 0 {
		t.Errorf("loads should be empty, got %d", len(loads))
	}
}

func TestLogPath(t *testing.T) {
	ctrl := setupTestController(t)

	path, err := ctrl.LogPath("test-model", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(path, "test-model") {
		t.Errorf("path should contain 'test-model', got: %s", path)
	}
	if !strings.Contains(path, "default") {
		t.Errorf("path should contain 'default', got: %s", path)
	}
}

func TestRPCServerLogPath(t *testing.T) {
	ctrl := setupTestController(t)

	path, err := ctrl.RPCServerLogPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(path, "rpc-server") {
		t.Errorf("path should contain 'rpc-server', got: %s", path)
	}
}

func TestStatusServer(t *testing.T) {
	ctrl := setupTestController(t)

	srv := ctrl.StatusServer()
	// May be nil if not initialized, just check it doesn't panic
	_ = srv
}

func TestNewStatusServer(t *testing.T) {
	ctrl := setupTestController(t)

	srv := ctrl.NewStatusServer()
	if srv == nil {
		t.Error("StatusServer should not be nil")
	}
}

func TestNewPublisher(t *testing.T) {
	ctrl := setupTestController(t)

	pub := ctrl.NewPublisher("client-id", "Client Name")
	if pub == nil {
		t.Error("Publisher should not be nil")
	}
}

func TestStartModel_MissingProfile(t *testing.T) {
	ctrl := setupTestController(t)
	cfg := &config.Config{}

	_, err := ctrl.StartModel(cfg, "nonexistent", "default", "")
	if err == nil {
		t.Error("expected error for missing model/profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestPollRemoteStatus_InvalidAddress(t *testing.T) {
	ctrl := setupTestController(t)

	_, err := ctrl.PollRemoteStatus("invalid-address")
	// Should fail with connection error
	if err == nil {
		t.Error("expected error for invalid address")
	}
}

// TestNilManager tests that methods handle nil manager gracefully where applicable
func TestListRunning_NilManager(t *testing.T) {
	ctrl := New(nil)

	// ListRunning calls mgr.List(), will panic with nil manager
	// This test documents the behavior
	defer func() {
		if r := recover(); r == nil {
			t.Log("ListRunning handled nil manager gracefully")
		} else {
			t.Logf("ListRunning panicked with nil manager (expected): %v", r)
		}
	}()

	_, _ = ctrl.ListRunning()
}

func TestStopModel_NilManager(t *testing.T) {
	ctrl := New(nil)

	// StopModel calls mgr.Stop(), will panic with nil manager
	defer func() {
		if r := recover(); r == nil {
			t.Log("StopModel handled nil manager gracefully")
		} else {
			t.Logf("StopModel panicked with nil manager (expected): %v", r)
		}
	}()

	_ = ctrl.StopModel("model", "profile")
}

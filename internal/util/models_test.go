package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHomeSupportsEnvVars(t *testing.T) {
	original := os.Getenv("LLMCTL_TEST_HOME")
	if err := os.Setenv("LLMCTL_TEST_HOME", "custom-home"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	defer func() {
		if original == "" {
			_ = os.Unsetenv("LLMCTL_TEST_HOME")
		} else {
			_ = os.Setenv("LLMCTL_TEST_HOME", original)
		}
	}()

	expanded, err := ExpandHome(filepath.Join("~", "subdir", "${LLMCTL_TEST_HOME}"))
	if err != nil {
		t.Fatalf("ExpandHome failed: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir failed: %v", err)
	}

	expected := filepath.Join(home, "subdir", "custom-home")
	if expanded != expected {
		t.Fatalf("expected %q, got %q", expected, expanded)
	}
}

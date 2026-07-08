// Package util holds small, dependency-free helpers shared across llmctl
// packages: filesystem locations and network port probing.
package util

import (
	"os"
	"path/filepath"
)

// HomeDir returns the llmctl state directory, ~/.llmctl, creating it if needed.
func HomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".llmctl")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// LogDir returns ~/.llmctl/logs, creating it if needed.
func LogDir() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// StateFile returns the path to the JSON file tracking running instances.
func StateFile() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "state.json"), nil
}

// RecentFile returns the path to the JSON file tracking recently run profiles.
func RecentFile() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "recent.json"), nil
}

// RPCStateFile returns the path to the JSON file tracking the RPC server state.
func RPCStateFile() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "rpc_state.json"), nil
}

// DefaultConfigPath returns the default location for config.yaml:
// ~/.llmctl/config.yaml.
func DefaultConfigPath() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "config.yaml"), nil
}

// LoadTimesFile returns the path to the JSON file tracking historical load times.
func LoadTimesFile() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "load_times.json"), nil
}

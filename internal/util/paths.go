// Package util holds small, dependency-free helpers shared across llmctl
// packages: filesystem locations, path expansion, and network port probing.
package util

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// TokRatesFile returns the path to the JSON file tracking persisted tok/s history.
func TokRatesFile() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "tok_rates.json"), nil
}

// StatusHistoryFile returns the path to the JSON file tracking status server
// history samples used by the browser dashboard.
func StatusHistoryFile() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "status_history.json"), nil
}

var percentEnvPattern = regexp.MustCompile(`%([A-Za-z0-9_]+)%`)

// ExpandHome replaces a leading "~" in path with the user's home directory
// and expands environment variables like ${VAR} or %VAR%.
func ExpandHome(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	expanded := os.ExpandEnv(path)
	expanded = percentEnvPattern.ReplaceAllStringFunc(expanded, func(match string) string {
		key := strings.Trim(match, "%")
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		return match
	})

	if expanded == "" || expanded[0] != '~' {
		return expanded, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, strings.TrimPrefix(expanded, "~")), nil
}

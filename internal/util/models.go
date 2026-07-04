package util

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

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

// ScanGGUF returns the full paths of all .gguf files directly under dir,
// sorted alphabetically.
func ScanGGUF(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".gguf") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

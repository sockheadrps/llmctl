package models

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanGGUF returns the full paths of all .gguf files directly under dir,
// sorted alphabetically. Subdirectories are skipped.
//
// This is model-domain knowledge: GGUF is the file format llama.cpp uses
// for model weights, so discovering them belongs with model types rather
// than in a generic utilities package.
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

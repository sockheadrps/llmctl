package models

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestScanGGUFReturnsOnlyGgufSorted(t *testing.T) {
	t.Helper()
	dir := t.TempDir()

	// Two gguf files out of alphabetical order plus a non-gguf file.
	write := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("b-model.gguf")
	write("a-model.GGUF")        // uppercase variant must also match
	write("notes.txt")           // ignored
	if err := os.Mkdir(filepath.Join(dir, "subdir.gguf"), 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	} // directories named .gguf must be skipped

	got, err := ScanGGUF(dir)
	if err != nil {
		t.Fatalf("ScanGGUF: %v", err)
	}

	want := []string{
		filepath.Join(dir, "a-model.GGUF"),
		filepath.Join(dir, "b-model.gguf"),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ScanGGUF = %v; want %v", got, want)
	}
}

func TestScanGGUFEmptyDir(t *testing.T) {
	got, err := ScanGGUF(t.TempDir())
	if err != nil {
		t.Fatalf("ScanGGUF: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result for empty dir, got %v", got)
	}
}

func TestScanGGUFMissingDir(t *testing.T) {
	_, err := ScanGGUF(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing dir, got nil")
	}
}

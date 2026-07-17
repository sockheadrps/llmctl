package models

import (
	"path/filepath"
	"testing"
)

func TestModelKeyByPath(t *testing.T) {
	existing := map[string]Model{
		"llama2": {Path: "/models/llama2.gguf"},
		"mistral": {Path: "/models/mistral.gguf"},
	}

	if got := ModelKeyByPath(existing, "/models/llama2.gguf"); got != "llama2" {
		t.Errorf("ModelKeyByPath(llama2) = %q; want llama2", got)
	}
	if got := ModelKeyByPath(existing, "/does/not/exist.gguf"); got != "" {
		t.Errorf("ModelKeyByPath(unknown) = %q; want \"\"", got)
	}
}

func TestModelKeyFromPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/models/Qwen2-7B-Instruct.gguf", "qwen2-7b-instruct"},
		{"/models/mistral 7b.GGUF", "mistral-7b"},
		{"/path/to/my_model.gguf", "my-model"},
		{"", "model"},
		{"///", "model"},
	}

	for _, tc := range cases {
		if got := ModelKeyFromPath(tc.path); got != tc.want {
			t.Errorf("ModelKeyFromPath(%q) = %q; want %q", tc.path, got, tc.want)
		}
	}
}

func TestModelNameFromPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/models/llama2.gguf", "llama2"},
		{"/models/Mistral-7B.GGUF", "Mistral-7B"},
		{"single.gguf", "single"},
	}

	for _, tc := range cases {
		got := filepath.Base(tc.path)
		got = got[:len(got)-len(filepath.Ext(got))]
		if got != tc.want {
			t.Errorf("ModelNameFromPath(%q) = %q; want %q", tc.path, got, tc.want)
		}
	}
}

func TestUniqueModelKey(t *testing.T) {
	existing := map[string]Model{
		"model": {},
		"model-2": {},
	}

	if got := UniqueModelKey(existing, "model"); got != "model-3" {
		t.Errorf("UniqueModelKey(occupied) = %q; want model-3", got)
	}
	if got := UniqueModelKey(existing, "newone"); got != "newone" {
		t.Errorf("UniqueModelKey(free) = %q; want newone", got)
	}
}

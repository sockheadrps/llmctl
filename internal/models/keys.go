package models

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ModelKeyByPath returns the key of the model whose Path matches path, or
// "" if none does.
func ModelKeyByPath(existing map[string]Model, path string) string {
	for k, mdl := range existing {
		if mdl.Path == path {
			return k
		}
	}
	return ""
}

func ModelNameFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// ModelKeyFromPath derives a URL-safe, lowercase key from a GGUF file path
// by stripping the extension, lowercasing, and replacing non-alphanumeric
// characters (except '.' and '-') with '-'. Empty result → "model".
func ModelKeyFromPath(path string) string {
	name := strings.ToLower(ModelNameFromPath(path))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		case r == '_', r == ' ':
			b.WriteRune('-')
		}
	}
	key := b.String()
	if key == "" {
		key = "model"
	}
	return key
}

// UniqueModelKey returns base if unused, else appends "-2", "-3", ... until
// a free key is found.
func UniqueModelKey(existing map[string]Model, base string) string {
	if _, ok := existing[base]; !ok {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := existing[candidate]; !ok {
			return candidate
		}
	}
}

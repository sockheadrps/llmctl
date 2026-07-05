// Package config loads and saves llmctl's config.yaml: the set of known
// Models and the launch Profiles each one owns.
package config

import (
	"fmt"
	"os"

	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/util"
	"gopkg.in/yaml.v3"
)

// Config is the root of config.yaml.
type Config struct {
	LlamaServerBin string `yaml:"llama_server_bin,omitempty"`
	// ModelsDir is the deprecated single-directory predecessor of
	// ModelsDirs. Load migrates it in and Save never writes it back out.
	ModelsDir   string                  `yaml:"models_dir,omitempty"`
	ModelsDirs  []string                `yaml:"models_dirs,omitempty"`
	RPCEnabled   bool   `yaml:"rpc_enabled,omitempty"`
	RPCEndpoint  string `yaml:"rpc_endpoint,omitempty"`
	RPCServerBin string `yaml:"rpc_server_bin,omitempty"`
	Models      map[string]models.Model `yaml:"models"`
}

// Load reads and parses the config file at path. If the file doesn't exist,
// it creates one with sensible defaults. Missing model/profile keys are
// populated onto the Key/Name fields so callers don't need the map key.
func Load(path string) (*Config, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		cfg := &Config{
			LlamaServerBin: "llama-server",
			ModelsDirs:     []string{},
			Models:         map[string]models.Model{},
		}
		if err := Save(path, cfg); err != nil {
			return nil, fmt.Errorf("create config %s: %w", path, err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.LlamaServerBin == "" {
		cfg.LlamaServerBin = "llama-server"
	}

	// Migrate the old single-directory field in, then drop it — the next
	// Save writes only models_dirs (omitempty leaves ModelsDir out once
	// it's blank again).
	if cfg.ModelsDir != "" {
		cfg.ModelsDirs = append([]string{cfg.ModelsDir}, cfg.ModelsDirs...)
		cfg.ModelsDir = ""
	}
	cfg.ModelsDirs = dedupeStrings(cfg.ModelsDirs)

	if cfg.Models == nil {
		cfg.Models = map[string]models.Model{}
	}

	for key, m := range cfg.Models {
		m.Key = key
		if m.Profiles == nil {
			m.Profiles = map[string]models.Profile{}
		}
		for pKey, p := range m.Profiles {
			p.Name = pKey
			m.Profiles[pKey] = p
		}
		cfg.Models[key] = m
	}

	return cfg, nil
}

// ResolvedModelsDirs expands ~ in each ModelsDirs entry, returning the
// directories llmctl scans for importable .gguf files.
func (c *Config) ResolvedModelsDirs() ([]string, error) {
	resolved := make([]string, 0, len(c.ModelsDirs))
	for _, dir := range c.ModelsDirs {
		expanded, err := util.ExpandHome(dir)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, expanded)
	}
	return resolved, nil
}

// AddModelsDir appends dir to ModelsDirs if it isn't already present.
// Returns false if it was already there.
func (c *Config) AddModelsDir(dir string) bool {
	for _, d := range c.ModelsDirs {
		if d == dir {
			return false
		}
	}
	c.ModelsDirs = append(c.ModelsDirs, dir)
	return true
}

// RemoveModelsDir removes dir from ModelsDirs, if present.
func (c *Config) RemoveModelsDir(dir string) {
	kept := make([]string, 0, len(c.ModelsDirs))
	for _, d := range c.ModelsDirs {
		if d != dir {
			kept = append(kept, d)
		}
	}
	c.ModelsDirs = kept
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// saveHeader is written above the generated YAML on every Save. yaml.Marshal
// doesn't round-trip comments, so hand-written notes in config.yaml would
// otherwise be silently dropped the first time the TUI persists a change.
const saveHeader = `# llmctl configuration.
# llama_server_bin is the llama-server binary to launch.
# models_dirs are scanned for .gguf files when adding a new model from the
# TUI — manage the list from the "Model Directories" screen, or edit here.
# This file is rewritten by the TUI when models/profiles are added — freeform
# comments beyond this header will not survive a save from within llmctl.
`

// Save writes cfg to path as YAML, creating parent directories as needed.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	out := append([]byte(saveHeader), data...)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// FindProfile looks up a model+profile pair by key.
func (c *Config) FindProfile(modelKey, profileKey string) (models.Model, models.Profile, error) {
	m, ok := c.Models[modelKey]
	if !ok {
		return models.Model{}, models.Profile{}, fmt.Errorf("model %q not found", modelKey)
	}
	p, ok := m.Profiles[profileKey]
	if !ok {
		return models.Model{}, models.Profile{}, fmt.Errorf("profile %q not found on model %q", profileKey, modelKey)
	}
	return m, p, nil
}

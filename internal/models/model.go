// Package models defines the core domain types used throughout llmctl:
// Models, the launch Profiles they own, and Running instances of a
// model+profile pair.
package models

// Profile is a saved launch configuration for a Model.
//
// The sampling fields (Temp, TopP, TopK, MinP, PresencePenalty,
// RepetitionPenalty) are pointers so an explicitly-set value of 0 (e.g.
// MinP=0.0 to disable min-p filtering) can be told apart from "not set at
// all". llama-server's own defaults differ from their "disabled" values
// for several of these (min-p defaults to 0.05 but 0.0 disables it; top-k
// defaults to 40 but 0 disables it), so omitting the flag and passing an
// explicit disabling value are not the same thing.
type Profile struct {
	Name              string   `yaml:"-"`
	Port              int      `yaml:"port"`
	CtxSize           int      `yaml:"ctx_size,omitempty"`
	Temp              *float64 `yaml:"temp,omitempty"`
	TopP              *float64 `yaml:"top_p,omitempty"`
	TopK              *int     `yaml:"top_k,omitempty"`
	MinP              *float64 `yaml:"min_p,omitempty"`
	PresencePenalty   *float64 `yaml:"presence_penalty,omitempty"`
	RepetitionPenalty *float64 `yaml:"repetition_penalty,omitempty"`
	FlashAttn         bool     `yaml:"flash_attention,omitempty"`
	GPULayers         int      `yaml:"gpu_layers,omitempty"`
	CacheTypeK        string   `yaml:"cache_type_k,omitempty"`
	CacheTypeV        string   `yaml:"cache_type_v,omitempty"`
	ExtraArgs         []string `yaml:"extra_args,omitempty"`
	Notes             string   `yaml:"notes,omitempty"`
}

// Model is a GGUF model along with the reusable profiles it can be run with.
// Exactly one of Path (a local GGUF file) or HFRepo (a HuggingFace repo,
// optionally suffixed ":quant", resolved and cached by llama-server itself
// via -hf) should be set.
type Model struct {
	Key      string             `yaml:"-"`
	Name     string             `yaml:"name"`
	Path     string             `yaml:"path,omitempty"`
	HFRepo   string             `yaml:"hf_repo,omitempty"`
	CacheDir string             `yaml:"cache_dir,omitempty"`
	Notes    string             `yaml:"notes,omitempty"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// IsRemote reports whether this model is fetched from HuggingFace rather
// than loaded from a local file.
func (m Model) IsRemote() bool {
	return m.HFRepo != ""
}

// Running describes a currently running llama-server instance.
type Running struct {
	ModelKey    string `json:"model_key"`
	ModelName   string `json:"model_name"`
	ProfileKey  string `json:"profile_key"`
	ProfileName string `json:"profile_name"`
	Port        int    `json:"port"`
	PID         int    `json:"pid"`
	LogFile     string `json:"log_file"`
	StartedAt   int64  `json:"started_at"`
}

// Label returns the "Model / Profile" display label used across the CLI and TUI.
func (r Running) Label() string {
	return r.ModelName + " / " + r.ProfileName
}

// RecentLimit is how many entries the rolling "recently run" history keeps.
const RecentLimit = 6

// RecentRun is one entry in the rolling history of recently started
// model+profile pairs, used to power the TUI's Recents tab.
type RecentRun struct {
	ModelKey   string `json:"model_key"`
	ProfileKey string `json:"profile_key"`
	RanAt      int64  `json:"ran_at"`
}

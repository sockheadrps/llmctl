// Package statusserver provides a lightweight HTTP server that exposes
// llmctl's current runtime state as JSON, and a client for polling remote
// instances. Used to share model/VRAM/tok-rate data across machines over LAN.
package statusserver

// Status is the JSON payload served at GET /status.
type Status struct {
	Version   string        `json:"version"`
	Running   []RunningInfo `json:"running"`
	RPCServer *RPCInfo      `json:"rpc_server,omitempty"`
	GPU       *GPUInfo      `json:"gpu,omitempty"`
	Clients   []ClientInfo  `json:"clients,omitempty"`
}

// RunningInfo describes one active llama-server instance.
type RunningInfo struct {
	Model          string    `json:"model"`
	Profile        string    `json:"profile"`
	Alias          string    `json:"alias,omitempty"`
	Port           int       `json:"port"`
	Health         string    `json:"health,omitempty"` // "loading", "up", or "down"
	TokS           float64   `json:"tok_s,omitempty"`
	TokPeak        float64   `json:"tok_peak,omitempty"`
	TokAvg         float64   `json:"tok_avg,omitempty"`
	TokHistory     []float64 `json:"tok_history,omitempty"` // last N in-session tok/s samples for sparkline
	VRAMMiB        int64     `json:"vram_mib,omitempty"`
	RAMMiB         int64     `json:"ram_mib,omitempty"`
	ModelSizeBytes int64     `json:"model_size_bytes,omitempty"`
}

// RPCInfo describes the local ggml-rpc-server state.
type RPCInfo struct {
	Up      bool   `json:"up"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	VRAMMiB int64  `json:"vram_mib,omitempty"`
}

// GPUInfo describes the local GPU.
type GPUInfo struct {
	Name     string `json:"name"`
	TotalMiB int64  `json:"total_mib"`
	UsedMiB  int64  `json:"used_mib"`
}

// ClientInfo is a status snapshot pushed by an llmctl running in RPC client
// mode to the llmctl running in RPC server mode.
type ClientInfo struct {
	ID       string        `json:"id"`
	Name     string        `json:"name,omitempty"`
	Addr     string        `json:"addr,omitempty"`
	LastSeen int64         `json:"last_seen"`
	Running  []RunningInfo `json:"running,omitempty"`
	GPU      *GPUInfo      `json:"gpu,omitempty"`
}

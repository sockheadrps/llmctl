package models

import "github.com/sockheadrps/llmctl/internal/util"

// SuggestPort returns a free port above the highest port currently in use,
// falling back to 8080 if any scan error occurs. Callers extract the used
// port list from their config layer; we take raw ints to keep models/ as
// a leaf package (breaking the natural cycle with config/).
func SuggestPort(used []int) int {
	maxPort := 8079
	for _, p := range used {
		if p > maxPort {
			maxPort = p
		}
	}
	start := maxPort + 1
	if free, err := util.FindFreePort(start); err == nil {
		return free
	}
	return start
}

// DefaultProfile returns a Profile populated with sensible starting
// defaults for most inference parameters. The Port field is left at 8080;
// callers should override it with a fresh SuggestPort result to avoid
// collisions.
func DefaultProfile() Profile {
	temp, topP, minP, topK := 0.6, 0.95, 0.0, 20
	return Profile{
		Name:      "default",
		Port:      8080,
		CtxSize:   8192,
		Temp:      &temp,
		TopP:      &topP,
		TopK:      &topK,
		MinP:      &minP,
		FlashAttn: true,
		GPULayers: 999,
	}
}

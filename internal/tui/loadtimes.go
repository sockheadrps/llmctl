package tui

import (
	"encoding/json"
	"os"
)

const maxLoadTimeHistory = 4

type profileHistory struct {
	RPC   []float64 `json:"rpc,omitempty"`
	NoRPC []float64 `json:"norpc,omitempty"`
}

type loadTimeStore map[string]profileHistory // key: "modelKey/profileKey"

func (s loadTimeStore) record(key string, durSecs float64, rpc bool) {
	h := s[key]
	if rpc {
		h.RPC = appendCapped(h.RPC, durSecs, maxLoadTimeHistory)
	} else {
		h.NoRPC = appendCapped(h.NoRPC, durSecs, maxLoadTimeHistory)
	}
	s[key] = h
}

func (s loadTimeStore) average(key string, rpc bool) (float64, bool) {
	h, ok := s[key]
	if !ok {
		return 0, false
	}
	times := h.NoRPC
	if rpc {
		times = h.RPC
	}
	if len(times) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, t := range times {
		sum += t
	}
	return sum / float64(len(times)), true
}

func appendCapped(s []float64, v float64, max int) []float64 {
	s = append(s, v)
	if len(s) > max {
		s = s[len(s)-max:]
	}
	return s
}

func loadLoadTimes(path string) loadTimeStore {
	data, err := os.ReadFile(path)
	if err != nil {
		return loadTimeStore{}
	}
	var store loadTimeStore
	if err := json.Unmarshal(data, &store); err != nil {
		return loadTimeStore{}
	}
	return store
}

func saveLoadTimes(path string, store loadTimeStore) error {
	data, err := json.Marshal(store)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

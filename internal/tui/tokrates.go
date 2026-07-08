package tui

import (
	"encoding/json"
	"os"
)

// maxTokRateHistory caps how many per-session averages are kept per profile.
const maxTokRateHistory = 20

// tokRateStore persists per-session average tok/s for each "modelKey/profileKey".
// Each element is the average tok/s measured during one run session.
type tokRateStore map[string][]float64

func (s tokRateStore) record(key string, sessionAvg float64) {
	s[key] = appendCapped(s[key], sessionAvg, maxTokRateHistory)
}

func (s tokRateStore) average(key string) (float64, bool) {
	rates := s[key]
	if len(rates) == 0 {
		return 0, false
	}
	var sum float64
	for _, r := range rates {
		sum += r
	}
	return sum / float64(len(rates)), true
}

func loadTokRates(path string) tokRateStore {
	data, err := os.ReadFile(path)
	if err != nil {
		return tokRateStore{}
	}
	var store tokRateStore
	if err := json.Unmarshal(data, &store); err != nil {
		return tokRateStore{}
	}
	return store
}

func saveTokRates(path string, store tokRateStore) error {
	data, err := json.Marshal(store)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

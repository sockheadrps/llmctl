package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Slot is one generation slot's status from llama-server's /slots endpoint
// (enabled by default, no server flags required). next_token is a
// single-element array in the actual API response (not a bare object), and
// it's only present while the slot is actively generating — callers should
// only trust Decoded() when IsProcessing is true.
type Slot struct {
	IsProcessing bool `json:"is_processing"`
	NextToken    []struct {
		NDecoded int `json:"n_decoded"`
	} `json:"next_token"`
}

// Decoded returns the slot's current decoded-token count, or 0 if it
// isn't generating (next_token is only populated while processing).
func (s Slot) Decoded() int {
	if len(s.NextToken) == 0 {
		return 0
	}
	return s.NextToken[0].NDecoded
}

// Slots fetches the current slot states for the llama-server instance on
// host:port. Errors (connection refused, non-200, bad JSON) are returned as-is
// so callers can decide how to treat an unreachable instance.
func Slots(host string, port int) ([]Slot, error) {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/slots", probeHost(host), port))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("slots endpoint returned %d", resp.StatusCode)
	}

	var slots []Slot
	if err := json.NewDecoder(resp.Body).Decode(&slots); err != nil {
		return nil, err
	}
	return slots, nil
}

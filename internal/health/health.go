// Package health probes a running llama-server instance to determine
// whether it has finished loading and is serving requests.
package health

import (
	"fmt"
	"net/http"
	"time"
)

// Status is the health state of a running instance.
type Status string

const (
	StatusUp      Status = "up"
	StatusDown    Status = "down"
	StatusLoading Status = "loading"
)

// Check hits the llama-server /health endpoint on host:port and
// classifies the result. A refused connection means the process hasn't
// opened its listener yet (still loading, or dead).
//
// llama-server returns 200 when idle and 503 when all slots are busy
// processing requests - both mean the server is alive and healthy.
// Connection refused or timeout means it's still loading or dead.
func Check(host string, port int) Status {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/health", probeHost(host), port))
	if err != nil {
		return StatusDown
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusServiceUnavailable:
		return StatusUp
	default:
		return StatusDown
	}
}

// Await polls Check until StatusUp, StatusDown, or timeout elapses.
func Await(host string, port int, timeout time.Duration) Status {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := Check(host, port); s == StatusUp {
			return s
		}
		time.Sleep(500 * time.Millisecond)
	}
	return Check(host, port)
}

func probeHost(host string) string {
	if host == "" {
		return "127.0.0.1"
	}
	return host
}

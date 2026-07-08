// Package health probes a running llama-server instance to determine
// whether it has finished loading and is serving requests.
package health

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/sockheadrps/llmctl/internal/process"
)

// Status is the health state of a running instance.
type Status string

const (
	StatusUp         Status = "up"
	StatusDown       Status = "down"
	StatusLoading    Status = "loading"
	StatusNotStarted Status = "not_started"
)

// Check hits the llama-server /health endpoint on host:port and
// classifies the result.
//
// llama-server status field meanings:
//   - 200 "ok"                → StatusUp (idle, fully loaded)
//   - 503 "loading model"     → StatusLoading (still initialising)
//   - 503 "no slot available" → StatusUp (busy but fully loaded)
//
// Connection refused or timeout means the HTTP listener isn't up yet.
func Check(host string, port int) Status {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/health", probeHost(host), port))
	if err != nil {
		return StatusDown
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return StatusUp
	case http.StatusServiceUnavailable:
		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return StatusUp
		}
		if body.Status == "loading model" {
			return StatusLoading
		}
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
	if host == "" || host == "0.0.0.0" || host == "::" {
		return "127.0.0.1"
	}
	return host
}

// ProbeRPCPort dials host:port via TCP and returns true if the connection
// succeeds. Use this as the primary "is the RPC server up" signal —
// independent of any PID bookkeeping.
func ProbeRPCPort(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(probeHost(host), strconv.Itoa(port)), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// CheckRPCServer checks whether the ggml-rpc-server process with the given
// PID is alive and its host:port is reachable via TCP. Returns StatusUp
// only when both checks pass; StatusDown otherwise.
func CheckRPCServer(host string, port int, pid int) Status {
	if !process.IsAlive(pid) {
		return StatusDown
	}
	if !ProbeRPCPort(host, port) {
		return StatusDown
	}
	return StatusUp
}

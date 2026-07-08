package tui

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/statusserver"
)

func TestRPCServerModeOwnsStatusServer(t *testing.T) {
	port := freeTCPPort(t)
	m := Model{
		cfg: &config.Config{
			RPCEnabled:          true,
			RPCMode:             "server",
			StatusServerEnabled: false,
			StatusServerHost:    "127.0.0.1",
			StatusServerPort:    port,
		},
	}

	if !m.shouldRunStatusServer() {
		t.Fatal("expected RPC server mode to own the status server")
	}
	if err := m.reconcileStatusServer(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.statusServer.Stop)

	if _, err := statusserver.PollAddr(fmt.Sprintf("127.0.0.1:%d", port)); err != nil {
		t.Fatalf("expected server-mode status server to respond: %v", err)
	}
}

func TestRPCClientModePublishesToRemoteStatusServer(t *testing.T) {
	port := freeTCPPort(t)
	server := Model{
		cfg: &config.Config{
			RPCEnabled:       true,
			RPCMode:          "server",
			StatusServerHost: "127.0.0.1",
			StatusServerPort: port,
		},
	}
	if err := server.reconcileStatusServer(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(server.statusServer.Stop)
	server.pushStatusServer()

	client := Model{
		cfg: &config.Config{
			RPCEnabled:          true,
			RPCMode:             "client",
			RemoteStatusAddr:    fmt.Sprintf("127.0.0.1:%d", port),
			StatusServerEnabled: true,
			StatusServerHost:    "127.0.0.1",
			StatusServerPort:    freeTCPPort(t),
		},
		statusPublisher: statusserver.NewPublisher("client-a", "Client A"),
		running: []models.Running{
			{
				ModelKey:    "model",
				ModelName:   "Client Model",
				ProfileKey:  "profile",
				ProfileName: "Default",
				Port:        8080,
			},
		},
	}
	if client.shouldRunStatusServer() {
		t.Fatal("expected RPC client mode not to run its own status server")
	}
	client.reconcileStatusPublisher()
	t.Cleanup(client.statusPublisher.Stop)
	client.pushStatusServer()

	st := pollStatusEventually(t, fmt.Sprintf("127.0.0.1:%d", port), func(st statusserver.Status) bool {
		return len(st.Clients) == 1 && len(st.Clients[0].Running) == 1
	})
	if got := st.Clients[0].Running[0].Model; got != "Client Model" {
		t.Fatalf("expected pushed client model name, got %q", got)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func pollStatusEventually(t *testing.T, addr string, ready func(statusserver.Status) bool) statusserver.Status {
	t.Helper()
	var lastErr error
	for i := 0; i < 40; i++ {
		st, err := statusserver.PollAddr(addr)
		if err == nil && ready(st) {
			return st
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("status server did not reach expected state: %v", lastErr)
	return statusserver.Status{}
}

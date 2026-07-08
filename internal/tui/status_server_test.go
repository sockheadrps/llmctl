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

func TestStatusServerRunsInRPCClientMode(t *testing.T) {
	port := freeTCPPort(t)
	m := Model{
		cfg: &config.Config{
			RPCEnabled:          true,
			RPCMode:             "client",
			StatusServerEnabled: false,
			StatusServerHost:    "127.0.0.1",
			StatusServerPort:    port,
		},
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

	if !m.shouldRunStatusServer() {
		t.Fatal("expected RPC client mode to require a status server")
	}
	if err := m.reconcileStatusServer(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.statusServer.Stop)

	st := pollStatusEventually(t, fmt.Sprintf("127.0.0.1:%d", port))
	if len(st.Running) != 1 {
		t.Fatalf("expected one running model in status payload, got %+v", st.Running)
	}
	if got := st.Running[0].Model; got != "Client Model" {
		t.Fatalf("expected client model name, got %q", got)
	}
}

func TestStatusServerNotRequiredWhenDisabledOutsideClientMode(t *testing.T) {
	m := Model{cfg: &config.Config{RPCEnabled: true, RPCMode: "server", StatusServerEnabled: false}}
	if m.shouldRunStatusServer() {
		t.Fatal("expected disabled server-mode status server to stay off")
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

func pollStatusEventually(t *testing.T, addr string) statusserver.Status {
	t.Helper()
	var lastErr error
	for i := 0; i < 20; i++ {
		st, err := statusserver.PollAddr(addr)
		if err == nil {
			return st
		}
		lastErr = err
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("status server did not respond: %v", lastErr)
	return statusserver.Status{}
}

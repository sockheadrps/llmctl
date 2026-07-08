package tui

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/health"
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

func TestStatusServerRequiresRPCServerMode(t *testing.T) {
	cases := []struct {
		name string
		cfg  *config.Config
		want bool
	}{
		{name: "rpc disabled with stale flag", cfg: &config.Config{StatusServerEnabled: true}, want: false},
		{name: "rpc client with stale flag", cfg: &config.Config{RPCEnabled: true, RPCMode: "client", StatusServerEnabled: true}, want: false},
		{name: "rpc server", cfg: &config.Config{RPCEnabled: true, RPCMode: "server"}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := (Model{cfg: tc.cfg}).shouldRunStatusServer(); got != tc.want {
				t.Fatalf("shouldRunStatusServer() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSelectingRPCServerStartsStatusServer(t *testing.T) {
	port := freeTCPPort(t)
	m := Model{
		cfg: &config.Config{
			RPCEnabled:       true,
			StatusServerHost: "127.0.0.1",
			StatusServerPort: port,
			Models:           map[string]models.Model{},
		},
		cfgPath: filepath.Join(t.TempDir(), "config.yaml"),
		settings: settingsState{rpc: rpcContentState{
			cursor: 2,
		}},
		health: healthMsg{"rpc-server": health.StatusUp},
	}

	next, _ := m.activateRPCRow()
	got := next.(Model)
	if got.statusServer == nil {
		t.Fatal("expected selecting RPC server mode to start the status server")
	}
	t.Cleanup(got.statusServer.Stop)
	if !got.cfg.StatusServerEnabled {
		t.Fatal("expected selecting RPC server mode to mark status server enabled")
	}
	if _, err := statusserver.PollAddr(fmt.Sprintf("127.0.0.1:%d", port)); err != nil {
		t.Fatalf("expected selected server-mode status server to respond: %v", err)
	}
}

func TestTickPublishesStatusOutsideMainScreen(t *testing.T) {
	port := freeTCPPort(t)
	m := Model{
		cfg: &config.Config{
			RPCEnabled:       true,
			RPCMode:          "server",
			StatusServerHost: "127.0.0.1",
			StatusServerPort: port,
			Models:           map[string]models.Model{},
		},
		screen: screenLogs,
		running: []models.Running{{
			ModelKey:    "model",
			ModelName:   "Model",
			ProfileKey:  "profile",
			ProfileName: "Profile",
			Port:        8080,
		}},
	}
	if err := m.reconcileStatusServer(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.statusServer.Stop)

	next, _ := m.Update(tickMsg(time.Now()))
	gotModel := next.(Model)
	st := pollStatusEventually(t, fmt.Sprintf("127.0.0.1:%d", port), func(st statusserver.Status) bool {
		return len(st.Running) == 1
	})
	if got := st.Running[0].Model; got != "Model" {
		t.Fatalf("expected tick to publish running model outside main screen, got %q", got)
	}
	if gotModel.screen != screenLogs {
		t.Fatalf("expected to remain on logs screen, got %v", gotModel.screen)
	}
}

func TestRPCClientModePublishesToRemoteStatusServer(t *testing.T) {
	port := freeTCPPort(t)
	modelPath := filepath.Join(t.TempDir(), "client-model.gguf")
	if err := os.WriteFile(modelPath, make([]byte, 4096), 0o600); err != nil {
		t.Fatal(err)
	}
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
			Models: map[string]models.Model{
				"model": {Name: "Client Model", Path: modelPath},
			},
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
	if got := st.Clients[0].Running[0].ModelSizeBytes; got != 4096 {
		t.Fatalf("expected pushed client model size, got %d", got)
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

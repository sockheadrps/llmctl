package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultsDashboardOffForNewConfigs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StatusDashboardEnabled() {
		t.Fatal("expected new configs to default the dashboard off")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); !strings.Contains(got, "status_dashboard_enabled: false") {
		t.Fatalf("saved config missing dashboard default, got:\n%s", got)
	}
}

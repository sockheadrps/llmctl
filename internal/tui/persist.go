package tui

import "github.com/sockheadrps/llmctl/internal/config"

// saveConfig writes the in-memory config back to disk at cfgPath.
func (m *Model) saveConfig() error {
	return config.Save(m.cfgPath, m.cfg)
}

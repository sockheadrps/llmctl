package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type rpcLayerSplitState struct {
	modelKey     string
	profileKey   string
	label        string
	totalLayers  int
	clientLayers int // layers on the local GPU; serverLayers = total - client
}

// rpcSplitApplies reports whether the RPC layer split slider should be shown
// for a given row: RPC must be enabled, the profile must have GPU layers, and
// there must be an actual RPC endpoint to connect to.
func (m Model) rpcSplitApplies(r row) bool {
	mdl, ok := m.cfg.Models[r.modelKey]
	if !ok {
		return false
	}
	p, ok := mdl.Profiles[r.profileKey]
	if !ok {
		return false
	}
	if p.CPUOnly || p.GPULayers <= 0 {
		return false
	}
	useRPC := m.cfg.RPCEnabled
	if p.RPCEnabled != nil {
		useRPC = *p.RPCEnabled
	}
	if !useRPC {
		return false
	}
	return strings.TrimSpace(m.discoveredRPCEndpoint) != "" ||
		strings.TrimSpace(m.cfg.RPCEndpoint) != ""
}

func (m Model) openRPCLayerSplit(r row) (tea.Model, tea.Cmd) {
	mdl := m.cfg.Models[r.modelKey]
	p := mdl.Profiles[r.profileKey]
	total := p.GPULayers
	// Default: split evenly, rounding client up.
	client := (total + 1) / 2
	m.rpcLayerSplit = rpcLayerSplitState{
		modelKey:     r.modelKey,
		profileKey:   r.profileKey,
		label:        r.label,
		totalLayers:  total,
		clientLayers: client,
	}
	m.screen = screenRPCLayerSplit
	return m, nil
}

func (m Model) updateRPCLayerSplit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := &m.rpcLayerSplit
	switch msg.String() {
	case "esc":
		m.screen = screenMain
	case "left", "h", "a":
		if s.clientLayers > 0 {
			s.clientLayers--
		}
	case "right", "l", "d":
		if s.clientLayers < s.totalLayers {
			s.clientLayers++
		}
	case "shift+left", "H", "A":
		s.clientLayers -= 5
		if s.clientLayers < 0 {
			s.clientLayers = 0
		}
	case "shift+right", "L", "D":
		s.clientLayers += 5
		if s.clientLayers > s.totalLayers {
			s.clientLayers = s.totalLayers
		}
	case "enter", " ":
		m.screen = screenMain
		tensorSplit := fmt.Sprintf("%d,%d", s.clientLayers, s.totalLayers-s.clientLayers)
		r := row{
			kind:       rowProfile,
			modelKey:   s.modelKey,
			profileKey: s.profileKey,
			label:      s.label,
		}
		m.starting = true
		m.startingLabel = s.label
		m.clearError()
		return m, m.startProfileCmdWithSplit(r, tensorSplit)
	}
	return m, nil
}

package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/build"
	"github.com/sockheadrps/llmctl/internal/health"
)

// renderHeaderLine renders the Models/Recents/Settings tab strip above the
// two boxes, centered over totalWidth (their combined rendered width) with
// dashes filling the rest, like a notebook tab strip sitting on the boxes'
// border.
func (m Model) renderHeaderLine(totalWidth int) string {
	return dashWrap(totalWidth, m.renderTabBarLabels(), m.focus == focusTabs)
}

// dashWrap centers content over totalWidth, padding both sides with a
// horizontal rule. The rule brightens when focused, matching how the
// boxes below signal focus via border color.
func dashWrap(totalWidth int, content string, focused bool) string {
	contentW := lipgloss.Width(content)
	// Mirror the Overview top border: 1 dash + space before content, space after,
	// remaining dashes fill to the right edge.
	right := totalWidth - 1 - 1 - contentW - 1
	if right < 0 {
		right = 0
	}

	dashColor := lipgloss.Color("30")
	if focused {
		dashColor = lipgloss.Color("240")
	}
	dashStyle := lipgloss.NewStyle().Foreground(dashColor)
	return dashStyle.Render("─") + " " + content + " " + dashStyle.Render(strings.Repeat("─", right))
}

// renderTabBarLabels shows the Overview/Models/Settings/Running tabs (Recents
// is a sub-tab within the Models pane), with the active tab styled as a chip.
func (m Model) renderTabBarLabels() string {
	tabs := []struct {
		mode  leftMode
		label string
	}{
		{modeOverview, "Overview"},
		{modeModels, "Models"},
		{modeSettings, "Settings"},
		{modeRunning, "Running"},
	}
	if m.networkTabVisible() {
		tabs = append(tabs, struct {
			mode  leftMode
			label string
		}{modeNetwork, "Network"})
	}
	if m.cfg.RPCEnabled {
		rpcTabLabel := "RPC Server"
		if m.cfg.RPCMode == "client" {
			rpcTabLabel = "RPC Connection"
		}
		tabs = append(tabs, struct {
			mode  leftMode
			label string
		}{modeRPCServer, rpcTabLabel})
	}

	tabFocused := m.focus == focusTabs
	rendered := make([]string, len(tabs))
	for i, t := range tabs {
		// modeRecents is a sub-tab of Models; show Models tab as active for both.
		isActive := m.leftMode == t.mode || (t.mode == modeModels && m.leftMode == modeRecents)
		if isActive {
			color := lipgloss.Color("39")
			if !tabFocused {
				color = lipgloss.Color("24")
			}
			rendered[i] = lipgloss.NewStyle().
				Foreground(color).
				Bold(true).
				Underline(true).
				Render(t.label)
		} else {
			rendered[i] = profileStyle.Render(t.label)
		}
	}

	return strings.Join(rendered, "  ")
}

// renderModelsSubTab renders the "Models  Recents" sub-tab header inside the
// left pane when the Models top-level tab is active.
func (m Model) renderModelsSubTab() string {
	subFocused := m.focus == focusLeft && m.modelSubTabFocused

	renderLabel := func(label string, isActive bool) string {
		if isActive && subFocused {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true).
				Underline(true).
				Render(label)
		}
		if isActive {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("24")).
				Bold(true).
				Render(label)
		}
		return profileStyle.Render(label)
	}

	return renderLabel("Models", m.leftMode == modeModels) + "  " + renderLabel("Recents", m.leftMode == modeRecents)
}

// renderSettingsList shows the Settings tab's menu of configuration
// sub-pages (currently just Model Directories).
func (m Model) renderSettingsList(width int) string {
	var b strings.Builder
	rows := m.buildSettingsRows()
	textWidth := formRowTextWidth(width)
	inSettings := m.focus == focusLeft || m.focus == focusSettingsContent
	for i, r := range rows {
		selected := i == m.settingsCursor
		cursor := "  "
		style := profileStyle
		if selected && inSettings {
			if m.focus == focusLeft {
				cursor = cursorStyle.Render("> ")
			}
			style = activeModelStyle
		}
		label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)))
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(label)))
	}
	b.WriteString("\n")
	b.WriteString(detailMutedStyle.Render("llmctl " + build.Version))
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderModelsTree(width int) string {
	var b strings.Builder
	textWidth := formRowTextWidth(width)

	if len(m.cfg.ModelsDirs) == 0 {
		b.WriteString(modelStyle.Render("No models folders set."))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("Please navigate to Settings > Model Directories"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("to add a directory to load models from."))
		return b.String()
	}

	if m.modelSearch != "" || m.searchEditing {
		query := m.modelSearch
		if query == "" {
			query = "type to filter..."
		}
		prefix := "/ "
		if m.searchEditing {
			prefix = cursorStyle.Render("/ ")
		}
		b.WriteString(prefix)
		b.WriteString(detailMutedStyle.Render(truncateText(query, max(1, textWidth-lipgloss.Width(prefix)))))
		b.WriteString("\n\n")
	}

	if len(m.rows) == 0 {
		empty := "(no models configured)"
		if m.modelSearch != "" {
			empty = "(no matches)"
		}
		b.WriteString(profileStyle.Render(empty))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("press A to add a model from your configured folders"))
		return b.String()
	}

	focused := m.focus == focusLeft
	for i, r := range m.rows {
		selected := i == m.cursor
		active := selected && focused
		cursor := "  "
		if active {
			cursor = cursorStyle.Render("> ")
		}

		switch r.kind {
		case rowModel:
			// A blank line ahead of every model but the first gives the
			// (now profile-less by default) list some breathing room
			// instead of a wall of names packed edge to edge.
			if i > 0 {
				b.WriteString("\n")
			}
			// The selected model stands out; every other one dims back
			// so it doesn't compete for attention.
			style := m.modelRowStyle(r, active)
			dot := ""
			if isRunning, status := m.modelRunningStatus(r.modelKey); isRunning {
				switch status {
				case health.StatusUp:
					dot = runningStyle.Render("● ")
				case health.StatusDown:
					dot = downStyle.Render("● ")
				default:
					dot = loadingStyle.Render("● ")
				}
			}
			label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)-lipgloss.Width(dot)))
			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, dot, style.Render(label)))

		case rowProfile:
			label := r.label
			style := profileStyle
			switch {
			case r.modelKey == m.pendingDeleteModel && r.profileKey == m.pendingDeleteProfile:
				style = pendingDeleteStyle
				label += " (del again to confirm)"
			case active:
				style = selectedProfileStyle
			}
			avgSuffix := ""
			if avg, ok := m.loadHistory.average(r.modelKey+"/"+r.profileKey, m.cfg.RPCEnabled); ok {
				avgSuffix = detailMutedStyle.Render("  (avg " + fmtLoadDur(time.Duration(avg*float64(time.Second))) + ")")
			}
			label = truncateText(label, max(1, textWidth-lipgloss.Width(cursor)-2-lipgloss.Width(avgSuffix)))
			fmt.Fprintf(&b, "%s  %s%s\n", cursor, style.Render(label), avgSuffix)

		case rowAddProfile:
			style := addStyle
			if active {
				style = selectedAddStyle
			}
			label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)-2))
			b.WriteString(fmt.Sprintf("%s  %s\n", cursor, style.Render(label)))

		case rowAddModel:
			b.WriteString("\n")
			style := addStyle
			if active {
				style = selectedAddStyle
			}
			label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)))
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(label)))
		}
	}
	return b.String()
}

func (m Model) modelRowStyle(r row, active bool) lipgloss.Style {
	switch {
	case active:
		return selectedProfileStyle
	case m.modelProfilesMode && r.modelKey == m.expandedModelKey:
		return activeModelStyle
	default:
		return profileStyle
	}
}

// renderRecentsList shows up to models.RecentLimit most recently run
// profiles, most recent first, for quick re-selection.
func (m Model) renderRecentsList(width int) string {
	var b strings.Builder

	if len(m.recentRows) == 0 {
		b.WriteString(profileStyle.Render("(nothing run yet)"))
		return b.String()
	}

	textWidth := formRowTextWidth(width)
	focused := m.focus == focusLeft
	for i, r := range m.recentRows {
		selected := i == m.recentCursor
		active := selected && focused
		cursor := "  "
		style := profileStyle
		if active {
			cursor = cursorStyle.Render("> ")
			style = selectedProfileStyle
		}
		label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)))
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(label)))
	}
	return b.String()
}

// tabTitle and tabInstructions describe each tab for the Details panel
// while focus is still at the outer tab bar — before arrowing down into a
// tab, there's no row selected yet to show details for, so explain what's
// in there instead of falling back to a generic empty-state message.
func (m Model) tabTitle(mode leftMode) string {
	switch mode {
	case modeOverview:
		return "Overview"
	case modeRecents:
		return "Recents"
	case modeSettings:
		return "Settings"
	case modeRunning:
		return "Running"
	case modeNetwork:
		return "Network"
	case modeRPCServer:
		if m.cfg.RPCMode == "client" {
			return "RPC Connection"
		}
		return "RPC Server"
	default:
		return "Models"
	}
}

func (m Model) tabInstructions(mode leftMode) string {
	switch mode {
	case modeRecents:
		return "Select from your most recently run profiles to quickly re-run one."
	case modeSettings:
		return "Select a settings category to configure, like where llmctl looks for model files."
	case modeRunning:
		return "Select a running instance to preview its output. Enter to stop it or view the full output."
	case modeNetwork:
		return "Switch between the RPC and internet network profiles, and view link status."
	case modeRPCServer:
		if m.cfg.RPCMode == "client" {
			return "View the health of your RPC connection and remote llmctl status."
		}
		return "Enter to start or stop the RPC server. Press e to view its output log."
	default:
		return "Select from saved model profiles, or add new model profiles."
	}
}

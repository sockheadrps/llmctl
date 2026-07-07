package tui

import (
	"fmt"
	"strings"
)

// renderSettingsDetail shows the given settings category's content in the
// Details pane — no separate screen, just like a model's profiles preview
// in place when it's focused. The header names the category itself instead
// of a generic "Details" label, same as the model/profile headers do.
func (m Model) renderSettingsDetail(categoryID string) string {
	label := categoryID
	for _, c := range settingsCategories {
		if c.id == categoryID {
			label = c.label
			break
		}
	}

	var b strings.Builder
	b.WriteString(modelStyle.Render(label))
	b.WriteString("\n\n")

	switch categoryID {
	case "model_dirs":
		b.WriteString(m.renderDirsContent())
	case "llama_bin":
		b.WriteString(m.renderBinContent())
	case "rpc":
		b.WriteString(m.renderRPCContent())
	}

	if categoryID == "model_dirs" && m.settings.dirs.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("error: " + m.settings.dirs.err))
	}
	if categoryID == "llama_bin" && m.settings.bin.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("error: " + m.settings.bin.err))
	}
	if categoryID == "rpc" && m.settings.rpc.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("error: " + m.settings.rpc.err))
	}

	return b.String()
}

func (m Model) renderBinContent() string {
	var b strings.Builder
	focused := m.focus == focusSettingsContent

	cursor := "  "
	style := profileStyle
	if focused {
		cursor = cursorStyle.Render("> ")
		style = selectedProfileStyle
	}

	value := m.cfg.LlamaServerBin
	if strings.TrimSpace(value) == "" {
		value = "llama-server"
	}
	fmt.Fprintf(&b, "%s%s\n\n", cursor, style.Render("Edit Binary"))
	b.WriteString(profileStyle.Render("Current: " + value))
	b.WriteString("\n")
	b.WriteString(detailMutedStyle.Render("Use a command on PATH or a full path to llama-server.exe."))
	b.WriteString("\n")

	if m.settings.bin.editing {
		b.WriteString("\n")
		b.WriteString(formLabelStyle.Render("Binary:") + " " + m.settings.bin.input.View())
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderRPCContent() string {
	var b strings.Builder
	focused := m.focus == focusSettingsContent

	enabledLabel := "Disabled"
	if m.cfg.RPCEnabled {
		enabledLabel = "Enabled"
	}

	rows := []string{"Toggle RPC (" + enabledLabel + ")", "Endpoint", "RPC Binary"}
	if m.netSupported && m.cfg.RPCEnabled {
		netTabLabel := "Network Tab (Disabled)"
		if m.cfg.NetworkTabEnabled {
			netTabLabel = "Network Tab (Enabled)"
		}
		rows = append(rows, netTabLabel)
	}

	for i, label := range rows {
		cursor := "  "
		style := profileStyle
		if focused && m.settings.rpc.cursor == i {
			cursor = cursorStyle.Render("> ")
			style = selectedProfileStyle
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, style.Render(label))
	}

	endpoint := m.cfg.RPCEndpoint
	if endpoint == "" {
		endpoint = "(not set)"
	}
	rpcBin := m.cfg.RPCServerBin
	if rpcBin == "" {
		rpcBin = "(uses default binary)"
	}
	b.WriteString("\n")
	b.WriteString(profileStyle.Render("Endpoint: " + endpoint))
	b.WriteString("\n")
	b.WriteString(profileStyle.Render("Binary:   " + rpcBin))
	b.WriteString("\n")

	if focused && m.settings.rpc.cursor == 3 && m.netSupported && m.cfg.RPCEnabled {
		b.WriteString("\n")
		b.WriteString(sectionTitleStyle.Render("Network Tab") + "\n")
		b.WriteString(profileStyle.Render(
			"Adds a Network tab to the TUI for managing nmcli connection\n"+
				"profiles without leaving llmctl. Use it to switch between your\n"+
				"internet and RPC ethernet connections when offloading model\n"+
				"layers to a Windows GPU over direct ethernet.\n\n"+
				"Requires: nmcli (NetworkManager) and polkit authorization.\n"+
				"Optional: ethtool for link speed and carrier detection.\n\n"+
				"Disable this if you manage network switching yourself and\n"+
				"don't need llmctl to control NetworkManager.",
		) + "\n")
	} else {
		b.WriteString(detailMutedStyle.Render("When RPC is enabled, the RPC binary is used instead of the default."))
		b.WriteString("\n")
	}

	if m.settings.rpc.editing {
		b.WriteString("\n")
		b.WriteString(formLabelStyle.Render("Endpoint:") + " " + m.settings.rpc.input.View())
		b.WriteString("\n")
	}
	if m.settings.rpc.binEditing {
		b.WriteString("\n")
		b.WriteString(formLabelStyle.Render("Binary:") + " " + m.settings.rpc.binInput.View())
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderDirsContent() string {
	var b strings.Builder

	focused := m.focus == focusSettingsContent

	// When not actively editing this category, read straight from config so
	// the preview is always current without requiring an Enter first.
	dirs := m.settings.dirs.list
	if !focused {
		dirs = m.cfg.ModelsDirs
	}

	addCursor, addRowStyle := "  ", addStyle
	switch {
	case m.settings.dirs.cursor == 0 && focused:
		addCursor = cursorStyle.Render("> ")
		addRowStyle = selectedAddStyle
	case m.settings.dirs.cursor == 0:
		addCursor = profileStyle.Render("> ")
	}
	fmt.Fprintf(&b, "%s%s\n", addCursor, addRowStyle.Render("+ Add Directory"))

	if len(dirs) == 0 {
		b.WriteString(profileStyle.Render("(no directories configured)"))
		b.WriteString("\n")
	}
	for i, d := range dirs {
		cursor := "  "
		style := profileStyle
		label := d
		switch {
		case d == m.settings.dirs.pendingDel:
			style = pendingDeleteStyle
			label += " (del again to confirm)"
		case m.settings.dirs.cursor == i+1 && focused:
			cursor = cursorStyle.Render("> ")
			style = selectedProfileStyle
		case m.settings.dirs.cursor == i+1:
			cursor = profileStyle.Render("> ")
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, style.Render(label))
	}

	if m.settings.dirs.editing {
		b.WriteString("\n")
		label := "New Directory:"
		if m.settings.dirs.editingIdx >= 0 {
			label = "Edit Directory:"
		}
		b.WriteString(formLabelStyle.Render(label) + " " + m.settings.dirs.input.View())
		b.WriteString("\n")
	}

	return b.String()
}

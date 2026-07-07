package tui

import (
	"fmt"
	"runtime"
	"strconv"
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
	case "status_server":
		b.WriteString(m.renderStatusServerContent())
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
	if categoryID == "status_server" && m.settings.statusSrv.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("error: " + m.settings.statusSrv.err))
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
		b.WriteString(formLabelStyle.Render("Binary:"));b.WriteString(" ");b.WriteString(m.settings.bin.input.View())
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderRPCContent() string {
	var b strings.Builder
	focused := m.focus == focusSettingsContent

	row := func(idx int, label string) {
		cursor := "  "
		style := profileStyle
		if focused && m.settings.rpc.cursor == idx {
			cursor = cursorStyle.Render("> ")
			style = selectedProfileStyle
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, style.Render(label))
	}

	enabledLabel := "Disabled"
	if m.cfg.RPCEnabled {
		enabledLabel = "Enabled"
	}
	row(0, "RPC ("+enabledLabel+")")

	if !m.cfg.RPCEnabled {
		b.WriteString("\n")
		b.WriteString(detailMutedStyle.Render("Enable RPC to choose client or server mode."))
		b.WriteString("\n")
		return b.String()
	}

	// Mode selector rows
	clientStyle, serverStyle := profileStyle, profileStyle
	if m.cfg.RPCMode == "client" {
		clientStyle = runningStyle
	}
	if m.cfg.RPCMode == "server" {
		serverStyle = runningStyle
	}

	clientCursor, serverCursor := "  ", "  "
	if focused && m.settings.rpc.cursor == 1 {
		clientCursor = cursorStyle.Render("> ")
	}
	if focused && m.settings.rpc.cursor == 2 {
		serverCursor = cursorStyle.Render("> ")
	}
	clientLabel := "[ Client ]"
	serverLabel := "[ Server ]"
	if m.cfg.RPCMode == "client" {
		clientLabel = "[✓ Client ]"
	}
	if m.cfg.RPCMode == "server" {
		serverLabel = "[✓ Server ]"
	}
	fmt.Fprintf(&b, "%s%s\n", clientCursor, clientStyle.Render(clientLabel))
	fmt.Fprintf(&b, "%s%s\n", serverCursor, serverStyle.Render(serverLabel))

	b.WriteString("\n")

	switch m.cfg.RPCMode {
	case "client":
		b.WriteString(sectionTitleStyle.Render("Client"))
		b.WriteString("\n")

		remoteAddrLabel := "Remote Status Address"
		if m.cfg.RemoteStatusAddr != "" {
			remoteAddrLabel += " (" + m.cfg.RemoteStatusAddr + ")"
		}
		row(3, remoteAddrLabel)
		if m.settings.rpc.remoteAddrEditing {
			fmt.Fprintf(&b, "  %s %s\n", formLabelStyle.Render("Address:"), m.settings.rpc.remoteAddrInput.View())
		}
		if focused && m.settings.rpc.cursor == 3 && !m.settings.rpc.remoteAddrEditing {
			b.WriteString(detailMutedStyle.Render("  The status server address of the remote llmctl (host:port).\n  RPC endpoint is auto-discovered from the status poll."))
			b.WriteString("\n")
			if m.cfg.RemoteStatusAddr != "" {
				if m.discoveredRPCEndpoint != "" {
					b.WriteString(runningStyle.Render("  Discovered: " + m.discoveredRPCEndpoint))
				} else {
					b.WriteString(detailMutedStyle.Render("  Discovered: (polling…)"))
				}
				b.WriteString("\n")
			}
		}

		endpointLabel := "Manual RPC Endpoint"
		if m.cfg.RPCEndpoint != "" {
			endpointLabel += " (" + m.cfg.RPCEndpoint + ")"
		}
		row(4, endpointLabel)
		if m.settings.rpc.editing {
			fmt.Fprintf(&b, "  %s %s\n", formLabelStyle.Render("Endpoint:"), m.settings.rpc.input.View())
		}
		if focused && m.settings.rpc.cursor == 4 && !m.settings.rpc.editing {
			b.WriteString(detailMutedStyle.Render("  Optional: used only if auto-discovery hasn't resolved yet."))
			b.WriteString("\n")
		}

	case "server":
		b.WriteString(sectionTitleStyle.Render("Server"))
		b.WriteString("\n")
		if runtime.GOOS == "windows" {
			binLabel := "Binary"
			if m.cfg.RPCServerBin != "" {
				binLabel += " (" + m.cfg.RPCServerBin + ")"
			}
			row(3, binLabel)
			if m.settings.rpc.rpcBinEditing {
				fmt.Fprintf(&b, "  %s %s\n", formLabelStyle.Render("Binary:"), m.settings.rpc.rpcBinInput.View())
			}
			row(4, "Port ("+strconv.Itoa(m.cfg.RPCServerPort)+")")
			if m.settings.rpc.portEditing {
				fmt.Fprintf(&b, "  %s %s\n", formLabelStyle.Render("Port:"), m.settings.rpc.portInput.View())
			}
		} else if m.netSupported {
			netTabLabel := "Network Tab (Disabled)"
			if m.cfg.NetworkTabEnabled {
				netTabLabel = "Network Tab (Enabled)"
			}
			row(3, netTabLabel)
			if focused && m.settings.rpc.cursor == 3 {
				b.WriteString(profileStyle.Render(
					"  Adds a Network tab for switching nmcli connections\n" +
						"  between internet and RPC ethernet without leaving llmctl.\n\n" +
						"  Requires: nmcli (NetworkManager) and polkit authorization.\n" +
						"  Optional: ethtool for link speed and carrier detection."))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(detailMutedStyle.Render("ggml-rpc-server will listen on " +
				m.cfg.RPCServerHost + ":" + strconv.Itoa(m.cfg.RPCServerPort)))
			b.WriteString("\n")
		}

	default:
		b.WriteString(detailMutedStyle.Render("Select Client or Server above."))
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
		b.WriteString(formLabelStyle.Render(label));b.WriteString(" ");b.WriteString(m.settings.dirs.input.View())
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderStatusServerContent() string {
	var b strings.Builder
	focused := m.focus == focusSettingsContent

	enabledLabel := "Disabled"
	if m.cfg.StatusServerEnabled {
		enabledLabel = "Enabled"
	}

	host := m.cfg.StatusServerHost
	if host == "" {
		host = "0.0.0.0"
	}

	rows := []string{
		"Toggle Status Server (" + enabledLabel + ")",
		"Host (" + host + ")",
		"Port (" + strconv.Itoa(m.cfg.StatusServerPort) + ")",
	}
	for i, label := range rows {
		cursor := "  "
		style := profileStyle
		if focused && m.settings.statusSrv.cursor == i {
			cursor = cursorStyle.Render("> ")
			style = selectedProfileStyle
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, style.Render(label))
	}

	b.WriteString("\n")
	b.WriteString(detailMutedStyle.Render(
		"Serves GET /status as JSON so other llmctl instances\n" +
			"on the same LAN can poll model name, VRAM and tok/s.\n" +
			"Default: 0.0.0.0:11435 (accessible from other machines)."))
	b.WriteString("\n")

	if m.settings.statusSrv.hostEditing {
		b.WriteString("\n")
		b.WriteString(formLabelStyle.Render("Host:"))
		b.WriteString(" ")
		b.WriteString(m.settings.statusSrv.hostInput.View())
		b.WriteString("\n")
	}
	if m.settings.statusSrv.portEditing {
		b.WriteString("\n")
		b.WriteString(formLabelStyle.Render("Port:"))
		b.WriteString(" ")
		b.WriteString(m.settings.statusSrv.portInput.View())
		b.WriteString("\n")
	}

	return b.String()
}

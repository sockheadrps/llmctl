package tui

import (
	"fmt"
	"net"
	"os/exec"
	"os/user"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// netStatusMsg carries the result of a periodic network status poll.
type netStatusMsg struct {
	activeConn string // name of the detected active connection, "" if unknown
	isRPC      bool   // activeConn matches the configured RPC conn name
	isInternet bool   // activeConn matches the configured Internet conn name
	linkState  string // "UP", "DOWN", or "UNKNOWN"
	speed      string // e.g. "10000Mb/s" or "unknown"
	carrier    bool   // true when ethtool reports "Link detected: yes"
	checkErr   string // non-empty when nmcli failed
}

// netSwitchResultMsg carries the result of a network profile switch.
type netSwitchResultMsg struct {
	toRPC bool
	err   error
}

// netSwitchVerifyMsg carries the result of the post-switch verification poll.
type netSwitchVerifyMsg struct {
	toRPC            bool
	actualIsRPC      bool
	actualIsInternet bool
}

// netSwitchState backs the pending switch confirmation modal.
type netSwitchState struct {
	toRPC  bool
	cursor int // 0 = Switch (confirm), 1 = Cancel
}

// netPickerRole identifies which connection role the picker is assigning.
type netPickerRole int

const (
	netPickerRoleInternet netPickerRole = iota
	netPickerRoleRPC
)

// netConnection is one entry from nmcli connection show.
type netConnection struct {
	name   string // nmcli connection profile name
	device string // associated device, "--" if inactive
	active bool   // device != "--"
}

// netPickerState backs the connection-picker modal.
type netPickerState struct {
	role        netPickerRole
	connections []netConnection
	cursor      int
	loading     bool
}

// netConnectionsMsg carries the fetched connection list back to Update.
type netConnectionsMsg struct {
	role        netPickerRole
	connections []netConnection
}

// networkTabVisible reports whether the Network tab should be shown and polled.
// Requires Linux, RPC enabled, and the network-management toggle also enabled.
func (m Model) networkTabVisible() bool {
	return m.netSupported && m.cfg.RPCEnabled && m.cfg.NetworkTabEnabled
}

// firstNonEmpty returns the first non-empty string from the arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// isNetworkAuthError reports whether err is a polkit "not authorized" failure
// from nmcli, which requires a system-level fix rather than a retry.
func isNetworkAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not authorized") || strings.Contains(msg, "authorization failed")
}

// networkFixCommand returns the shell command that grants the current user
// network control without a password prompt via the netdev group.
func networkFixCommand() string {
	username := "$USER"
	if u, err := user.Current(); err == nil && u.Username != "" {
		username = u.Username
	}
	return "sudo usermod -aG netdev " + username
}

// copyNetworkFix copies the polkit fix command to the clipboard.
func (m Model) copyNetworkFix() (tea.Model, tea.Cmd) {
	if !isNetworkAuthError(m.err) {
		return m, nil
	}
	if err := writeClipboard(networkFixCommand()); err != nil {
		m.setError(fmt.Errorf("copy: %w", err), "")
		return m, nil
	}
	m.clearError()
	return m, nil
}

// netRowSwitchRPC … netRowCount are the fixed cursor positions in the
// Network tab's left pane.
const (
	netRowSwitchRPC      = 0
	netRowSwitchInternet = 1
	netRowSetInternet    = 2
	netRowSetRPC         = 3
	netRowCount          = 4
)

// checkNetworkStatusCmd polls nmcli for the active connection, the Go
// stdlib net package for link state, and ethtool for speed/carrier.
// ip link show is replaced by net.InterfaceByName so there is no
// dependency on the ip binary.
func checkNetworkStatusCmd(iface, internetConn, rpcConn string) tea.Cmd {
	return func() tea.Msg {
		msg := netStatusMsg{linkState: "UNKNOWN", speed: "unknown"}

		out, err := exec.Command("nmcli", "-t", "-f", "NAME", "connection", "show", "--active").Output()
		if err != nil {
			msg.checkErr = "nmcli: " + err.Error()
		} else {
			for _, line := range strings.Split(string(out), "\n") {
				name := strings.TrimSpace(line)
				if name == "" {
					continue
				}
				if rpcConn != "" && name == rpcConn {
					msg.activeConn = name
					msg.isRPC = true
				} else if internetConn != "" && name == internetConn {
					msg.activeConn = name
					msg.isInternet = true
				}
			}
		}

		// Use stdlib net package — no dependency on the ip binary.
		if iface != "" {
			if netIface, err := net.InterfaceByName(iface); err == nil {
				if netIface.Flags&net.FlagUp != 0 {
					msg.linkState = "UP"
				} else {
					msg.linkState = "DOWN"
				}
			}
		}

		if iface != "" {
			if _, err := exec.LookPath("ethtool"); err == nil {
				ethOut, _ := exec.Command("ethtool", iface).CombinedOutput()
				for _, line := range strings.Split(string(ethOut), "\n") {
					line = strings.TrimSpace(line)
					switch {
					case strings.HasPrefix(line, "Speed:"):
						msg.speed = strings.TrimSpace(strings.TrimPrefix(line, "Speed:"))
					case line == "Link detected: yes":
						msg.carrier = true
					}
				}
			}
		}

		return msg
	}
}

// switchNetworkCmd brings down the current profile and brings up the target
// one via nmcli, matching the behaviour of cmd/network.go.
func switchNetworkCmd(toRPC bool, internetConn, rpcConn string) tea.Cmd {
	return func() tea.Msg {
		var upConn string
		if toRPC {
			upConn = rpcConn
		} else {
			upConn = internetConn
		}

		if strings.TrimSpace(upConn) == "" {
			return netSwitchResultMsg{toRPC: toRPC, err: fmt.Errorf("missing connection name")}
		}

		out, err := exec.Command("nmcli", "connection", "up", upConn).CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			return netSwitchResultMsg{toRPC: toRPC, err: fmt.Errorf("nmcli: %s", msg)}
		}
		return netSwitchResultMsg{toRPC: toRPC}
	}
}

// verifyNetworkSwitchCmd waits briefly then polls nmcli to confirm the
// expected connection came up after a switch. If nmcli cannot be reached,
// the msg leaves both flags false so the caller can decide whether to surface
// an error (it should not — regular polling will reflect reality on its own).
func verifyNetworkSwitchCmd(toRPC bool, internetConn, rpcConn string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1500 * time.Millisecond)
		out, err := exec.Command("nmcli", "-t", "-f", "NAME", "connection", "show", "--active").Output()
		if err != nil {
			return netSwitchVerifyMsg{toRPC: toRPC}
		}
		var isRPC, isInternet bool
		for _, line := range strings.Split(string(out), "\n") {
			name := strings.TrimSpace(line)
			if rpcConn != "" && name == rpcConn {
				isRPC = true
			}
			if internetConn != "" && name == internetConn {
				isInternet = true
			}
		}
		return netSwitchVerifyMsg{toRPC: toRPC, actualIsRPC: isRPC, actualIsInternet: isInternet}
	}
}

// listNetworkConnectionsCmd fetches all nmcli connection profiles and
// returns them as a netConnectionsMsg for the picker.
func listNetworkConnectionsCmd(role netPickerRole) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("nmcli", "-t", "-f", "NAME,DEVICE", "connection", "show").Output()
		if err != nil {
			return netConnectionsMsg{role: role}
		}
		var conns []netConnection
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// nmcli -t separates fields with ':'; names may contain spaces
			// but not unescaped colons in practice. SplitN(2) handles this.
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			name := strings.TrimSpace(parts[0])
			device := strings.TrimSpace(parts[1])
			if name == "" {
				continue
			}
			conns = append(conns, netConnection{
				name:   name,
				device: device,
				active: device != "--" && device != "",
			})
		}
		return netConnectionsMsg{role: role, connections: conns}
	}
}

// renderNetworkList renders the Network tab's left-pane content: a status
// summary, two switch action rows, and two configure rows.
func (m Model) renderNetworkList(width int) string {
	var b strings.Builder
	textWidth := formRowTextWidth(width)
	inNetwork := m.focus == focusLeft && m.leftMode == modeNetwork

	s := m.netStatus

	connLabel := "detecting…"
	connStyle := profileStyle
	switch {
	case s.isRPC:
		connLabel = "RPC (" + m.netRPCConn + ")"
		connStyle = runningStyle
	case s.isInternet:
		connLabel = "Internet (" + m.netInternetConn + ")"
		connStyle = infoStyle
	case s.checkErr != "":
		connLabel = "unavailable"
		connStyle = downStyle
	}

	linkLabel := s.linkState
	linkStyle := profileStyle
	switch s.linkState {
	case "UP":
		linkStyle = runningStyle
		if s.speed != "unknown" && s.speed != "" {
			linkLabel = "UP · " + s.speed
		}
	case "DOWN":
		linkStyle = downStyle
	}

	b.WriteString(sectionTitleStyle.Render("Status"))
	b.WriteString("\n")
	rowFmt := "  %-8s %s\n"
	b.WriteString(fmt.Sprintf(rowFmt,
		profileStyle.Render("active"),
		connStyle.Render(truncateText(connLabel, max(1, textWidth-12))),
	))
	b.WriteString(fmt.Sprintf(rowFmt,
		profileStyle.Render("link"),
		linkStyle.Render(truncateText(linkLabel, max(1, textWidth-12))),
	))
	b.WriteString(fmt.Sprintf(rowFmt,
		profileStyle.Render("iface"),
		profileStyle.Render(truncateText(m.netIface, max(1, textWidth-12))),
	))
	b.WriteString("\n")

	type actionRow struct {
		label string
		idx   int
		style lipgloss.Style
	}
	rows := []actionRow{
		{"→ RPC  (" + m.netRPCConn + ")", netRowSwitchRPC, profileStyle},
		{"→ Internet  (" + m.netInternetConn + ")", netRowSwitchInternet, profileStyle},
	}
	configRows := []actionRow{
		{"Set internet conn…", netRowSetInternet, addStyle},
		{"Set RPC conn…", netRowSetRPC, addStyle},
	}

	b.WriteString(sectionTitleStyle.Render("Switch"))
	b.WriteString("\n")
	for _, r := range rows {
		cursor := "  "
		style := r.style
		if r.idx == m.netCursor && inNetwork {
			cursor = cursorStyle.Render("> ")
			style = activeModelStyle
		}
		b.WriteString(cursor)
		b.WriteString(style.Render(truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render("Configure"))
	b.WriteString("\n")
	for _, r := range configRows {
		cursor := "  "
		style := r.style
		if r.idx == m.netCursor && inNetwork {
			cursor = cursorStyle.Render("> ")
			style = selectedAddStyle
		}
		b.WriteString(cursor)
		b.WriteString(style.Render(truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)))))
		b.WriteString("\n")
	}

	return b.String()
}

// renderNetworkDetails renders the right-pane Details content for the
// Network tab.
func (m Model) renderNetworkDetails() string {
	var b strings.Builder
	s := m.netStatus

	b.WriteString(modelStyle.Render("Network"))
	b.WriteString("\n\n")

	if s.checkErr != "" {
		b.WriteString(downStyle.Render("status check failed"))
		b.WriteString("\n")
		b.WriteString(detailMutedStyle.Render(s.checkErr))
		b.WriteString("\n")
		b.WriteString("\n")
	} else {
		connLabel := "none detected"
		connStyle := profileStyle
		switch {
		case s.isRPC:
			connLabel = m.netRPCConn
			connStyle = runningStyle
		case s.isInternet:
			connLabel = m.netInternetConn
			connStyle = infoStyle
		}
		linkStyle := profileStyle
		switch s.linkState {
		case "UP":
			linkStyle = runningStyle
		case "DOWN":
			linkStyle = downStyle
		}

		fmt.Fprintf(&b, "%s  %s\n", detailMutedStyle.Render("active conn"), connStyle.Render(connLabel))
		fmt.Fprintf(&b, "%s  %s\n", detailMutedStyle.Render("link state "), linkStyle.Render(s.linkState))
		fmt.Fprintf(&b, "%s  %s\n", detailMutedStyle.Render("speed      "), profileStyle.Render(s.speed))
		fmt.Fprintf(&b, "%s  %s\n", detailMutedStyle.Render("carrier    "), profileStyle.Render(fmt.Sprintf("%v", s.carrier)))
		fmt.Fprintf(&b, "%s  %s\n", detailMutedStyle.Render("iface      "), profileStyle.Render(m.netIface))
		b.WriteString("\n")
	}

	switch m.netCursor {
	case netRowSwitchRPC:
		b.WriteString(sectionTitleStyle.Render("→ Switch to RPC"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("Brings down the internet connection and brings\nup the RPC link (" + m.netRPCConn + ") for local host networking."))
		b.WriteString("\n")
	case netRowSwitchInternet:
		b.WriteString(sectionTitleStyle.Render("→ Switch to Internet"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("Brings down the RPC link and brings up the\ninternet connection (" + m.netInternetConn + ")."))
		b.WriteString("\n")
	case netRowSetInternet:
		b.WriteString(sectionTitleStyle.Render("Set internet conn…"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("Pick which nmcli connection profile to use\nas the internet connection.\n\nCurrently: " + m.netInternetConn))
		b.WriteString("\n")
	case netRowSetRPC:
		b.WriteString(sectionTitleStyle.Render("Set RPC conn…"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("Pick which nmcli connection profile to use\nas the RPC link to the Windows machine.\n\nCurrently: " + m.netRPCConn))
		b.WriteString("\n")
	}

	if isNetworkAuthError(m.err) {
		b.WriteString("\n")
		b.WriteString(downStyle.Render("Not authorized to control networking"))
		b.WriteString("\n\n")
		b.WriteString(sectionTitleStyle.Render("Fix"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("Add your user to the netdev group,\nthen log out and back in:"))
		b.WriteString("\n\n")
		b.WriteString(selectedProfileStyle.Render(networkFixCommand()))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("c to copy command"))
		b.WriteString("\n")
	}

	return b.String()
}

// viewNetworkSwitchModal renders the confirm modal for a pending switch.
func (m Model) viewNetworkSwitchModal() string {
	target := m.netInternetConn
	if m.netSwitch.toRPC {
		target = m.netRPCConn
	}

	switchOpt := "  Switch  "
	cancelOpt := "  Cancel  "
	if m.netSwitch.cursor == 0 {
		switchOpt = selectedProfileStyle.Render("[ Switch ]")
		cancelOpt = profileStyle.Render(cancelOpt)
	} else {
		switchOpt = profileStyle.Render(switchOpt)
		cancelOpt = selectedProfileStyle.Render("[ Cancel ]")
	}
	options := fmt.Sprintf("%s    %s", switchOpt, cancelOpt)

	title := modalTitleStyle.Render("Switch Network")
	label := profileStyle.Render("→ " + target)
	help := helpStyle.Render("←/→ choose  enter confirm  esc cancel")
	body := lipgloss.JoinVertical(lipgloss.Center, title, "", label, "", options, "", help)
	return modalStyle.Render(body)
}

// viewNetworkPickerModal renders the connection-picker modal.
func (m Model) viewNetworkPickerModal() string {
	var title string
	switch m.netPicker.role {
	case netPickerRoleInternet:
		title = "Set Internet Connection"
	case netPickerRoleRPC:
		title = "Set RPC Connection"
	}

	lines := []string{modalTitleStyle.Render(title), ""}

	if m.netPicker.loading {
		lines = append(lines, loadingStyle.Render("loading connections…"))
	} else if len(m.netPicker.connections) == 0 {
		lines = append(lines, profileStyle.Render("no connections found"))
	} else {
		for i, conn := range m.netPicker.connections {
			cursor := "  "
			nameStyle := profileStyle
			if i == m.netPicker.cursor {
				cursor = cursorStyle.Render("> ")
				nameStyle = activeModelStyle
			}
			row := cursor + nameStyle.Render(conn.name)
			if conn.active {
				row += "  " + runningStyle.Render("●")
			} else if conn.device != "--" && conn.device != "" {
				row += "  " + detailMutedStyle.Render(conn.device)
			}
			lines = append(lines, row)
		}
	}

	lines = append(lines, "", helpStyle.Render("↑↓ move  enter select  esc cancel"))
	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return modalStyle.Render(body)
}

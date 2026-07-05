package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// netStatusMsg carries the result of a periodic network status poll.
type netStatusMsg struct {
	activeConn string // name of the detected active connection, "" if unknown
	isRPC      bool   // activeConn matches the configured RPC conn name
	isInternet bool   // activeConn matches the configured Internet conn name
	linkState  string // "UP", "DOWN", or "UNKNOWN"
	speed      string // e.g. "10000Mb/s" or "unknown"
	carrier    bool   // true when ethtool reports "Link detected: yes"
	checkErr   string // non-empty when nmcli/ip failed
}

// netSwitchResultMsg carries the result of a network profile switch.
type netSwitchResultMsg struct {
	toRPC bool
	err   error
}

// netSwitchState backs the pending switch confirmation modal.
type netSwitchState struct {
	toRPC  bool
	cursor int // 0 = Switch (confirm), 1 = Cancel
}

// netRowSwitchRPC / netRowSwitchInternet are the netCursor index values for
// the two action rows in the Network tab's left pane.
const (
	netRowSwitchRPC      = 0
	netRowSwitchInternet = 1
)

// checkNetworkStatusCmd polls nmcli, ip-link, and ethtool asynchronously and
// returns the result as a netStatusMsg.
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

		if iface != "" {
			ipOut, err := exec.Command("ip", "link", "show", iface).Output()
			if err == nil {
				flat := string(ipOut)
				switch {
				case strings.Contains(flat, "state UP"):
					msg.linkState = "UP"
				case strings.Contains(flat, "state DOWN"):
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
		var downConn, upConn string
		if toRPC {
			downConn = internetConn
			upConn = rpcConn
		} else {
			downConn = rpcConn
			upConn = internetConn
		}

		if strings.TrimSpace(downConn) != "" {
			// Ignore down errors — the profile may already be inactive.
			exec.Command("nmcli", "connection", "down", downConn).Run() //nolint:errcheck
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

// renderNetworkList renders the Network tab's left-pane content: a status
// summary followed by two switchable action rows.
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

	b.WriteString(sectionTitleStyle.Render("Status") + "\n")
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
	b.WriteString(sectionTitleStyle.Render("Switch") + "\n")

	actions := []string{
		"→ RPC  (" + m.netRPCConn + ")",
		"→ Internet  (" + m.netInternetConn + ")",
	}
	for i, label := range actions {
		cursor := "  "
		style := profileStyle
		if i == m.netCursor && inNetwork {
			cursor = cursorStyle.Render("> ")
			style = activeModelStyle
		}
		b.WriteString(cursor + style.Render(truncateText(label, max(1, textWidth-lipgloss.Width(cursor)))) + "\n")
	}
	return b.String()
}

// renderNetworkDetails renders the right-pane Details content for the Network
// tab: extended status at the top, then a description of the selected action.
func (m Model) renderNetworkDetails(width int) string {
	var b strings.Builder
	s := m.netStatus

	b.WriteString(modelStyle.Render("Network") + "\n\n")

	if s.checkErr != "" {
		b.WriteString(downStyle.Render("status check failed") + "\n")
		b.WriteString(detailMutedStyle.Render(s.checkErr) + "\n")
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
		linkLabel := s.linkState
		linkStyle := profileStyle
		switch s.linkState {
		case "UP":
			linkStyle = runningStyle
		case "DOWN":
			linkStyle = downStyle
		}

		fmt.Fprintf(&b, "%s  %s\n",
			detailMutedStyle.Render("active conn"),
			connStyle.Render(connLabel),
		)
		fmt.Fprintf(&b, "%s  %s\n",
			detailMutedStyle.Render("link state "),
			linkStyle.Render(linkLabel),
		)
		fmt.Fprintf(&b, "%s  %s\n",
			detailMutedStyle.Render("speed      "),
			profileStyle.Render(s.speed),
		)
		fmt.Fprintf(&b, "%s  %s\n",
			detailMutedStyle.Render("carrier    "),
			profileStyle.Render(fmt.Sprintf("%v", s.carrier)),
		)
		fmt.Fprintf(&b, "%s  %s\n",
			detailMutedStyle.Render("iface      "),
			profileStyle.Render(m.netIface),
		)
		b.WriteString("\n")
	}

	// Action description for the selected row.
	switch m.netCursor {
	case netRowSwitchRPC:
		b.WriteString(sectionTitleStyle.Render("→ Switch to RPC") + "\n")
		b.WriteString(profileStyle.Render("Brings down the internet connection and brings\nup the RPC link ("+m.netRPCConn+") for local host networking.") + "\n")
	case netRowSwitchInternet:
		b.WriteString(sectionTitleStyle.Render("→ Switch to Internet") + "\n")
		b.WriteString(profileStyle.Render("Brings down the RPC link and brings up the\ninternet connection ("+m.netInternetConn+").") + "\n")
	}

	return b.String()
}

// viewNetworkSwitchModal renders the confirmation modal for a pending network
// profile switch.
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

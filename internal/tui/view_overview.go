package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/statusserver"
)

type gpuLoadSlice struct {
	label string
	info  statusserver.GPUDeviceInfo
}

// viewOverviewPage renders the complete Overview screen as a single outer box
// (╭ at col 0, ╮ at col totalW-1) with a vertical │ separator between two
// columns: ACTIVE SERVICES on the left, SYSTEM TELEMETRY on the right.
// Using one box eliminates the "disconnected inner boxes" problem entirely.
func (m Model) viewOverviewPage() string {
	totalW := m.width
	if totalW <= 0 {
		totalW = fallbackWidth
	}
	totalH := m.height
	if totalH <= 0 {
		totalH = fallbackHeight
	}

	bs := lipgloss.NewStyle().Foreground(lipgloss.Color("38"))

	// top(1) + blank(1) + content(contentH) + bottom(1) = totalH
	contentH := totalH - 3
	if contentH < 2 {
		contentH = 2
	}

	// Row format: │ leftContent │ rightContent │
	// 1 + 1 + leftCW + 1 + 1 + rightCW + 1 + 1 = totalW → leftCW+rightCW = totalW-6
	leftCW, rightCW := m.overviewColumnWidths(totalW)

	// sepCol is the visual column of the │ separator.
	// Row: │(0) space(1) leftContent(leftCW, cols 2..leftCW+1) space(leftCW+2) │(leftCW+3) …
	sepCol := leftCW + 3

	leftLines := strings.Split(strings.TrimRight(m.renderActiveServices(leftCW, contentH), "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(m.renderSystemTelemetry(rightCW, contentH), "\n"), "\n")

	navText, versionText := m.overviewNavVersion(leftCW, rightCW)

	var sb strings.Builder
	sb.WriteString(m.buildOverviewTopBorder(totalW, sepCol))
	sb.WriteString("\n")
	// Blank row between tab bar and content — │ at separator column.
	sb.WriteString(bs.Render("│") + strings.Repeat(" ", sepCol-1) + bs.Render("│") + strings.Repeat(" ", totalW-2-sepCol) + bs.Render("│") + "\n")

	sep := bs.Render("│")
	for i := 0; i < contentH; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		lpad := leftCW - lipgloss.Width(l)
		if lpad < 0 {
			lpad = 0
		}
		rpad := rightCW - lipgloss.Width(r)
		if rpad < 0 {
			rpad = 0
		}
		sb.WriteString(bs.Render("│") + " " + l + strings.Repeat(" ", lpad) + " " + sep + r + strings.Repeat(" ", rpad) + " " + bs.Render("│") + "\n")
	}
	sb.WriteString(m.buildOverviewBottomBorder(totalW, navText, versionText, sepCol))
	return sb.String()
}

// overviewColumnWidths returns (leftCW, rightCW) respecting the user-dragged
// separator position stored in overviewSepX.
func (m Model) overviewColumnWidths(totalW int) (leftCW, rightCW int) {
	const minLeft, minRight = 18, 14
	avail := totalW - 6
	if avail < minLeft+minRight {
		avail = minLeft + minRight
	}
	if m.overviewSepX > 0 {
		leftCW = m.overviewSepX - 3
	} else {
		leftCW = avail * 3 / 5
	}
	if leftCW < minLeft {
		leftCW = minLeft
	}
	if leftCW > avail-minRight {
		leftCW = avail - minRight
	}
	rightCW = avail - leftCW
	return
}

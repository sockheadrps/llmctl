package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/build"
)

// buildOverviewTopBorder builds ╭─ tabs ─┬─╮ with a ┬ at sepCol where the
// column separator │ meets the top border.
func (m Model) buildOverviewTopBorder(totalW, sepCol int) string {
	tabs := m.renderTabBarLabels()
	tabsW := lipgloss.Width(tabs)

	focused := m.focus == focusTabs
	dashColor := lipgloss.Color("38")
	if focused {
		dashColor = lipgloss.Color("39")
	}
	dashStyle := lipgloss.NewStyle().Foreground(dashColor)
	bs := lipgloss.NewStyle().Foreground(lipgloss.Color("38"))

	// Fixed prefix: ╭─ tabs  (visual width = tabsW+4: ╭─ space tabs space)
	preW := tabsW + 4
	// Dashes between prefix and ┬, then dashes from ┬ to ╮.
	leftDashes := sepCol - preW
	if leftDashes < 0 {
		leftDashes = 0
	}
	rightDashes := totalW - 1 - sepCol - 1 // cols sepCol+1..totalW-2
	if rightDashes < 0 {
		rightDashes = 0
	}

	return bs.Render("╭") +
		dashStyle.Render("─") + " " + tabs + " " +
		dashStyle.Render(strings.Repeat("─", leftDashes)) +
		bs.Render("┬") +
		dashStyle.Render(strings.Repeat("─", rightDashes)) +
		bs.Render("╮")
}

// buildOverviewBottomBorder builds ╰─ navText ─┴─ versionText ─╯ with a ┴ at
// sepCol where the column separator │ meets the bottom border.
// Left zone (cols 0..sepCol-1): ╰─ nav ─×n
// Right zone (cols sepCol+1..totalW-1): ─×m version ─╯
func (m Model) buildOverviewBottomBorder(totalW int, navText, versionText string, sepCol int) string {
	bs := lipgloss.NewStyle().Foreground(lipgloss.Color("38"))
	navW := lipgloss.Width(navText)
	verW := lipgloss.Width(versionText)

	// Left zone: ╰─ nav ─×leftDashes  (visual width = sepCol)
	// ╰(1) ─(1) sp(1) nav(navW) sp(1) ─×n = navW+4+n = sepCol → n = sepCol-navW-4
	var leftPart string
	if navText != "" {
		leftDashes := sepCol - navW - 4
		if leftDashes < 0 {
			leftDashes = 0
		}
		leftPart = bs.Render("╰") + bs.Render("─") + " " + navText + " " + bs.Render(strings.Repeat("─", leftDashes))
	} else {
		leftPart = bs.Render("╰") + bs.Render(strings.Repeat("─", sepCol-1))
	}

	// Right zone: ─×rightDashes version ─╯  (visual width = totalW-1-sepCol)
	// rightLen = totalW-1-sepCol chars for cols sepCol+1..totalW-1
	rightLen := totalW - 1 - sepCol
	var rightPart string
	if versionText != "" {
		// ─×r sp ver sp ─ ╯ → r + verW + 4 = rightLen → r = rightLen-verW-4
		rightDashes := rightLen - verW - 4
		if rightDashes < 0 {
			rightDashes = 0
		}
		rightPart = bs.Render(strings.Repeat("─", rightDashes)) + " " + versionText + " " + bs.Render("─") + bs.Render("╯")
	} else {
		rightPart = bs.Render(strings.Repeat("─", rightLen-1)) + bs.Render("╯")
	}

	return leftPart + bs.Render("┴") + rightPart
}

// overviewNavVersion returns the nav and version strings for the bottom border,
// sized to fit the column widths.
func (m Model) overviewNavVersion(leftCW, rightCW int) (navText, versionText string) {
	var statusStr string
	switch {
	case m.starting:
		statusStr = "  " + loadingStyle.Render("starting "+m.startingLabel+"…")
	case m.stopping:
		statusStr = "  " + loadingStyle.Render("stopping "+m.stoppingLabel+"…")
	case m.netSwitching:
		statusStr = "  " + loadingStyle.Render("switching network…")
	case m.err != nil:
		msg := firstLine(m.err.Error())
		if m.errLogPath != "" {
			msg += "  (e for logs)"
		}
		statusStr = "  " + errorStyle.Render(msg)
	}

	navFull := helpStyle.Render("click to copy addr  ·  ←→/ad tabs  ·  q quit") + statusStr
	navMid := helpStyle.Render("←→/ad tabs  ·  q quit")
	navMin := helpStyle.Render("q quit")
	switch {
	case leftCW >= lipgloss.Width(navFull)+2:
		navText = navFull
	case leftCW >= lipgloss.Width(navMid)+2:
		navText = navMid
	case leftCW >= lipgloss.Width(navMin)+2:
		navText = navMin
	}

	verFull := detailMutedStyle.Render("llmctl " + build.Version)
	verShort := detailMutedStyle.Render(build.Version)
	switch {
	case rightCW >= lipgloss.Width(verFull)+2:
		versionText = verFull
	case rightCW >= lipgloss.Width(verShort)+2:
		versionText = verShort
	}
	return
}

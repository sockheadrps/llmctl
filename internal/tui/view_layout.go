package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	tui_form "github.com/sockheadrps/llmctl/internal/tui/form"
)

// paneDimensions works out how big the main screen's panes should be so
// they fill the actual terminal instead of shrink-wrapping their content
// with empty space around them. leftWidth/rightWidth are content widths
// (Width() values — each box's border adds 2 more on render); targetLeftH
// is the content-height budget the left pane (and thus the whole layout)
// should fill, matching splitPaneHeight's units.
func (m Model) paneDimensions() (leftWidth, rightWidth, targetLeftH int) {
	termWidth, termHeight := m.width, m.height
	if termWidth <= 0 {
		termWidth = fallbackWidth
	}
	if termHeight <= 0 {
		termHeight = fallbackHeight
	}

	// The two boxes sit side by side with no gap; each has 2 columns of
	// border overhead not covered by its Width() value.
	avail := termWidth - 4
	if m.leftWidthOverride > 0 {
		leftWidth = m.leftWidthOverride
		if leftWidth < minLeftWidth {
			leftWidth = minLeftWidth
		}
		if leftWidth > avail-minRightWidth {
			leftWidth = avail - minRightWidth
		}
	} else {
		leftWidth = avail * 2 / 5
		if leftWidth < minLeftWidth {
			leftWidth = minLeftWidth
		}
	}
	rightWidth = avail - leftWidth
	if rightWidth < minRightWidth {
		rightWidth = minRightWidth
	}

	targetLeftH = termHeight - chromeHeight - 2 // -2 for the left box's own border
	if targetLeftH < minBodyHeight {
		targetLeftH = minBodyHeight
	}
	return leftWidth, rightWidth, targetLeftH
}

// computeSplitHeights returns the running/details content heights for the
// right column, honouring a user-dragged rightSplitOverride when set.
func (m Model) computeSplitHeights(leftH int) (runningH, detailsH int) {
	if m.rightSplitOverride > 0 {
		budget := leftH - 2
		if budget < 8 {
			budget = 8
		}
		runningH = m.rightSplitOverride
		if runningH < 3 {
			runningH = 3
		}
		if runningH > budget-3 {
			runningH = budget - 3
		}
		detailsH = budget - runningH
		if detailsH < 3 {
			detailsH = 3
		}
		return
	}
	return splitPaneHeight(leftH, len(m.running))
}

// splitPaneHeight divides the right column so its total rendered height
// (including each box's own border) matches leftHeight, the left pane's
// content height. Running entries are one line each, so that box is sized
// to just fit runningCount (with a small minimum/cap), while the Details
// pane stays compact so a selected model preview does not stretch the full
// interface and push the top of the TUI out of view.
func splitPaneHeight(leftHeight, runningCount int) (running, details int) {
	// The right column stacks two bordered boxes instead of the left's
	// one, so it has two extra border lines to account for.
	budget := leftHeight - 2
	if budget < 8 {
		budget = 8
	}

	running = runningCount + 3 // header + blank + at least one row
	if running < 5 {
		running = 5
	}
	if max := budget * 2 / 5; running > max {
		running = max
	}

	details = budget - running
	if details < 5 {
		details = 5
	}
	if details > 12 {
		details = 12
	}
	return running, details
}

func (m Model) mainDetailsLineCount() int {
	rightW, _, detailsH := m.mainDetailsGeometry()
	if detailsH <= 0 {
		return 0
	}
	return len(wrappedContentLines(m.renderDetails(rightW), rightW))
}

func (m Model) mainDetailsVisibleLines() int {
	_, _, detailsH := m.mainDetailsGeometry()
	return detailsH
}

func (m Model) mainDetailsGeometry() (rightW, runningH, detailsH int) {
	leftW, rightW, targetLeftH := m.paneDimensions()
	leftMeasureStyle := lipgloss.NewStyle().Width(leftW).Padding(0, 1)
	rightMeasureStyle := lipgloss.NewStyle().Width(rightW).Padding(0, 1)

	leftH := max(lipgloss.Height(leftMeasureStyle.Render(m.renderLeftPaneContent(leftW))), targetLeftH)
	runningH, detailsH = m.computeSplitHeights(leftH)
	if n := lipgloss.Height(rightMeasureStyle.Render(m.renderRunning())); n > runningH {
		runningH = n
		budget := leftH - 2
		if budget < 8 {
			budget = 8
		}
		detailsH = budget - runningH
		if detailsH < 3 {
			detailsH = 3
		}
	}

	rightAsLeftH := runningH + detailsH + 2
	if leftH > rightAsLeftH {
		detailsH += leftH - rightAsLeftH
	}
	return rightW, runningH, detailsH
}

func (m *Model) resetDetailsScroll() {
	m.detailsScroll = 0
	m.detailsDir = 1
	m.detailsPause = scrollPauseTicks
	m.detailsManualScroll = false
}

func (m *Model) advanceDetailsScroll(lines, visible int) {
	if m.detailsHovered || m.detailsManualScroll {
		return
	}
	m.detailsScroll, m.detailsDir, m.detailsPause = advanceAutoScroll(m.detailsScroll, m.detailsDir, m.detailsPause, lines, visible)
}

func advanceAutoScroll(offset, dir, pause, lines, visible int) (int, int, int) {
	if dir == 0 {
		dir = 1
	}
	maxScroll := max(0, lines-visible)
	if maxScroll == 0 {
		return 0, 1, 0
	}

	if pause > 0 {
		return offset, dir, pause - 1
	}

	offset += dir
	if offset >= maxScroll {
		return maxScroll, -1, scrollPauseTicks
	}
	if offset <= 0 {
		return 0, 1, scrollPauseTicks
	}
	return offset, dir, 0
}

func wrappedContentLines(content string, width int) []string {
	innerWidth := tui_form.FormDescriptionTextWidth(width)
	if innerWidth <= 0 {
		innerWidth = tui_form.FormDescriptionTextWidth(minRightWidth)
	}
	rendered := lipgloss.NewStyle().Width(innerWidth).Render(strings.TrimRight(content, "\n"))
	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func contentWindow(lines []string, visible, offset int) []string {
	if visible <= 0 {
		return nil
	}
	if len(lines) == 0 {
		return []string{""}
	}

	maxOffset := max(0, len(lines)-visible)
	offset = max(0, min(offset, maxOffset))

	window := make([]string, 0, min(visible, len(lines)))
	for i := offset; i < min(offset+visible, len(lines)); i++ {
		window = append(window, lines[i])
	}
	return window
}

func (m Model) renderDetailsWindow(content string, width, height int) string {
	lines := wrappedContentLines(content, width)
	return strings.Join(contentWindow(lines, height, m.detailsScroll), "\n")
}

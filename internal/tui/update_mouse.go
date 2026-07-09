package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	leftW, _, _ := m.paneDimensions()
	dividerLeft := leftW + 1  // right border of left pane
	dividerRight := leftW + 2 // left border of right pane

	// Use the actual rendered runningH (accounts for content overflow) to
	// locate the horizontal divider. Y=0 header, Y=1 body top border, so
	// divider row = 1 (top border) + runningH + 1 (bottom border of running) = runningH+2.
	_, actualRunningH, actualDetailsH := m.mainDetailsGeometry()
	hDividerY := actualRunningH + 2
	inRightColumn := msg.X > dividerRight

	inDetailsPane := inRightColumn && msg.Y > hDividerY && m.leftMode != modeRunning

	switch msg.Action {
	case tea.MouseActionPress:
		// Scroll wheel over the details pane: pause auto-scroll and let the user
		// navigate manually. detailsManualScroll stays set until the row changes.
		if inDetailsPane {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.detailsManualScroll = true
				if m.detailsScroll > 0 {
					m.detailsScroll--
				}
				return m, nil
			case tea.MouseButtonWheelDown:
				m.detailsManualScroll = true
				lines := m.mainDetailsLineCount()
				visible := m.mainDetailsVisibleLines()
				if m.detailsScroll < max(0, lines-visible) {
					m.detailsScroll++
				}
				return m, nil
			}
		}
		if msg.Button != tea.MouseButtonLeft {
			break
		}
		// Overview tab: drag separator or click service entries.
		if m.leftMode == modeOverview {
			leftCW, _ := m.overviewColumnWidths(m.width)
			sepCol := leftCW + 3
			if msg.X == sepCol {
				m.overviewSepDragging = true
				break
			}
			if run, ok := m.overviewClickedEntry(msg.X, msg.Y); ok {
				return m.copyOverviewEntry(run)
			}
			break
		}
		if msg.X == dividerLeft || msg.X == dividerRight {
			m.dividerDragging = true
		} else if inRightColumn && m.leftMode != modeRunning {
			if msg.Y >= hDividerY-1 && msg.Y <= hDividerY+2 {
				m.rightDividerDragging = true
			}
		}

	case tea.MouseActionMotion:
		// Track hover so auto-scroll pauses while the cursor is over the pane.
		m.detailsHovered = inDetailsPane
		if m.overviewSepDragging {
			const minLeft, minRight = 18, 14
			avail := m.width - 6
			newLeftCW := msg.X - 3
			if newLeftCW < minLeft {
				newLeftCW = minLeft
			}
			if newLeftCW > avail-minRight {
				newLeftCW = avail - minRight
			}
			m.overviewSepX = newLeftCW + 3
		}
		if m.dividerDragging {
			newLeft := msg.X - 1
			avail := m.width - 4
			if newLeft < minLeftWidth {
				newLeft = minLeftWidth
			}
			if newLeft > avail-minRightWidth {
				newLeft = avail - minRightWidth
			}
			m.leftWidthOverride = newLeft
		}
		if m.rightDividerDragging {
			// newRunningH = drag Y minus header row minus body top border.
			// The minimum is the raw content line count of the running list
			// (so the box is never set smaller than its content — that would
			// trigger the overflow correction which can push leftH past the
			// terminal height). actualRunningH is the rendered box height
			// (padded/filled), not the content height, so we measure content
			// directly.
			_, rightW, _ := m.paneDimensions()
			rightMeasure := lipgloss.NewStyle().Width(rightW).Padding(0, 1)
			contentH := lipgloss.Height(rightMeasure.Render(m.renderRunning()))
			minRunning := max(3, contentH)

			newRunningH := msg.Y - 2
			totalBudget := actualRunningH + actualDetailsH
			if newRunningH < minRunning {
				newRunningH = minRunning
			}
			if newRunningH > totalBudget-3 {
				newRunningH = totalBudget - 3
			}
			m.rightSplitOverride = newRunningH
		}

	case tea.MouseActionRelease:
		m.dividerDragging = false
		m.rightDividerDragging = false
		m.overviewSepDragging = false
	}
	return m, nil
}

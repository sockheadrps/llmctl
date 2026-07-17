package form

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FormSliderTotal returns the current GPU layers value from the form field,
// used as the upper bound of the tensor split slider.
func FormSliderTotal(gpuLayersValue string) int {
	val := strings.TrimSpace(gpuLayersValue)
	if val == "" {
		return 0
	}
	n := 0
	for _, r := range val {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	if n <= 0 {
		return 0
	}
	return n
}

// AdjustTensorSplit applies a delta to the client-side tensor split value and
// clamps the result to the available GPU layer total.
func AdjustTensorSplit(current, total, delta int) int {
	current += delta
	if current < 0 {
		return 0
	}
	if total > 0 && current > total {
		return total
	}
	return current
}

// FormRPCActive reports whether RPC is effectively enabled for the profile
// being edited.
func FormRPCActive(rpcEnabledValue string, cfgRPCEnabled bool) bool {
	val := strings.TrimSpace(rpcEnabledValue)
	switch val {
	case "false":
		return false
	case "true":
		return true
	default:
		return cfgRPCEnabled
	}
}

// FormDescriptionTextWidth converts a pane width to usable text width.
func FormDescriptionTextWidth(paneWidth int) int {
	return max(8, paneWidth-2)
}

// FormRowTextWidth converts a pane width to usable text width.
func FormRowTextWidth(paneWidth int) int {
	return max(8, paneWidth-2)
}

// FormVisibleRows returns the number of visible rows in the left form pane.
func FormVisibleRows(paneHeight int) int {
	return max(1, paneHeight-1)
}

// FormPaneHeight computes the outer form pane height from the window height.
func FormPaneHeight(windowHeight int) int {
	if windowHeight <= 0 {
		return 20
	}
	// title + blank line + bordered pane + newline + hotkey line
	return max(8, windowHeight-6)
}

// FormPaneWidths computes the left/details pane widths for the form screen.
func FormPaneWidths(termWidth int) (leftWidth, detailsWidth int) {
	if termWidth <= 0 {
		termWidth = 100
	}

	available := termWidth - 4
	if available < 36+24 {
		available = 36 + 24
	}

	detailsWidth = 36
	leftWidth = 70
	if leftWidth+detailsWidth > available {
		detailsWidth = max(24, min(36, available/3))
		leftWidth = available - detailsWidth
		if leftWidth < 36 {
			leftWidth = 36
			detailsWidth = max(24, available-leftWidth)
		}
	}
	return leftWidth, detailsWidth
}

// TruncateText applies a simple width-limited truncation.
func TruncateText(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 1 {
		return s[:width]
	}
	return s[:width-1] + "."
}

// FitStyledLine truncates a styled string if needed.
func FitStyledLine(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return TruncateText(s, width)
}

// DescriptionWindow returns a visible slice of wrapped lines.
func DescriptionWindow(lines []string, visible, offset int) []string {
	if visible <= 0 {
		return nil
	}
	if len(lines) == 0 {
		lines = []string{""}
	}

	maxOffset := max(0, len(lines)-visible)
	offset = max(0, min(offset, maxOffset))

	window := make([]string, 0, visible)
	for i := offset; i < min(offset+visible, len(lines)); i++ {
		window = append(window, lines[i])
	}
	for len(window) < visible {
		window = append(window, "")
	}
	return window
}

// WrapWords is a simple word wrapper used by the form details panel.
func WrapWords(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	lines := []string{words[0]}
	for _, word := range words[1:] {
		last := len(lines) - 1
		if len(lines[last])+1+len(word) <= width {
			lines[last] += " " + word
			continue
		}
		lines = append(lines, word)
	}
	return lines
}

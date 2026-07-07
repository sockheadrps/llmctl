package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/process"
)

type exportArgsState struct {
	label    string
	argsStr  string
	copied   bool
}

func (m Model) openExportArgs(r row) (tea.Model, tea.Cmd) {
	mdl, ok := m.cfg.Models[r.modelKey]
	if !ok {
		return m, nil
	}
	p, ok := mdl.Profiles[r.profileKey]
	if !ok {
		return m, nil
	}

	args := process.BuildProfileArgs(p)
	argsStr := strings.Join(args, " ")

	copied := false
	if err := writeClipboard(argsStr); err == nil {
		copied = true
	}

	m.exportArgs = exportArgsState{
		label:   r.label,
		argsStr: argsStr,
		copied:  copied,
	}
	m.screen = screenExportArgs
	return m, nil
}

func (m Model) updateExportArgs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.screen = screenMain
		case "c", " ", "enter":
			if err := writeClipboard(m.exportArgs.argsStr); err == nil {
				m.exportArgs.copied = true
			}
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonRight {
			if err := writeClipboard(m.exportArgs.argsStr); err == nil {
				m.exportArgs.copied = true
			}
		}
	}
	return m, nil
}

func (m Model) viewExportArgsModal() string {
	title := modalTitleStyle.Render("Export Args — " + m.exportArgs.label)

	// Wrap the args string to ~60 chars for display
	wrapWidth := 60
	words := strings.Fields(m.exportArgs.argsStr)
	var lines []string
	cur := ""
	for _, w := range words {
		if cur == "" {
			cur = w
		} else if len(cur)+1+len(w) <= wrapWidth {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) == 0 {
		lines = []string{"(no parameters)"}
	}

	argsDisplay := detailMutedStyle.Render(strings.Join(lines, "\n"))

	copyStatus := helpStyle.Render("c/enter/right-click copy · esc close")
	if m.exportArgs.copied {
		copyStatus = fmt.Sprintf("%s  %s",
			runningStyle.Render("✓ copied to clipboard"),
			helpStyle.Render("· esc close"),
		)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, title, "", argsDisplay, "", copyStatus)
	return modalStyle.Render(body)
}

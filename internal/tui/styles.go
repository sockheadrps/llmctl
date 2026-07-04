package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	focusedPaneStyle = paneStyle.
				BorderForeground(lipgloss.Color("205"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	modelStyle = lipgloss.NewStyle().
			Bold(true)

	profileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	selectedProfileStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	downStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	addStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Italic(true)

	selectedAddStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true).
				Italic(true)

	formLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Width(30)

	formFocusedLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true).
				Width(30)

	pendingDeleteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true).
				Reverse(true)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 3)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))
)

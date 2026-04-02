package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	headerActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))

	headerFilteredStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("214"))

	cellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	cursorCellStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("229")).
			Bold(true)

	cursorRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	// Filter overlay styles
	filterBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(1, 2)

	filterTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("99"))

	filterInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	filterItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	filterItemSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212"))

	filterItemCursorStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("252"))

	filterItemCursorSelectedStyle = lipgloss.NewStyle().
					Background(lipgloss.Color("236")).
					Foreground(lipgloss.Color("212"))

	filterHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
)

package tui

import lipgloss "charm.land/lipgloss/v2"

var (
	labelStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	selectedLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	separatorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpDisabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	savedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

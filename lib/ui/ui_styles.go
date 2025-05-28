package ui

import (
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	Style          = lipgloss.NewStyle()
	TextStyle      = Style.MarginLeft(3).Width(55)
	SelectYesStyle = TextStyle.Foreground(ansi.BrightGreen)
	SelectNoStyle  = TextStyle.Foreground(ansi.BrightMagenta)
	SpinnerStyle   = Style.Foreground(ansi.BrightGreen)
	LoadingStyle   = Style.Foreground(ansi.BrightBlue)
	HelpStyle      = Style.MarginLeft(3).MarginTop(1)
	HelpDescStyle  = Style.Foreground(lipgloss.Color("#56566B"))
	HelpKeyStyle   = Style.Foreground(lipgloss.Color("#787897"))
	AbortStyle     = Style.MarginTop(1).MarginLeft(2)
	ConsentStyle   = TextStyle.Foreground(ansi.BrightBlue)

	inputPromptStyle = Style.Foreground(ansi.BrightGreen)
	inputTextStyle   = Style.Foreground(ansi.BrightWhite)
)

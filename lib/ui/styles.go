package ui

import (
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	style            = lipgloss.NewStyle()
	textStyle        = style.MarginLeft(3).Width(55)
	selectYesStyle   = textStyle.Foreground(ansi.BrightGreen)
	selectNoStyle    = textStyle.MarginTop(1).Foreground(ansi.BrightMagenta)
	spinnerStyle     = style.Foreground(ansi.BrightGreen)
	loadingStyle     = style.Foreground(ansi.BrightBlue)
	helpStyle        = style.MarginLeft(3).MarginTop(1)
	helpDescStyle    = style.Foreground(lipgloss.Color("#56566B"))
	helpKeyStyle     = style.Foreground(lipgloss.Color("#787897"))
	inputPromptStyle = style.Foreground(ansi.BrightGreen)
	inputTextStyle   = style.Foreground(ansi.BrightWhite)
	abortStyle       = style.MarginTop(1).MarginLeft(2)
)

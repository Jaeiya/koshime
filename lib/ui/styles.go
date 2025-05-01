package ui

import (
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	style            = lipgloss.NewStyle()
	textStyle        = style.PaddingLeft(3)
	selectYesStyle   = textStyle.Foreground(ansi.BrightGreen)
	selectNoStyle    = textStyle.MarginTop(1).Foreground(ansi.BrightMagenta)
	spinnerStyle     = style.Foreground(ansi.BrightGreen)
	loadingStyle     = style.Foreground(ansi.BrightBlue)
	helpDescStyle    = style.Foreground(lipgloss.Color("#56566B"))
	helpKeyStyle     = style.Foreground(lipgloss.Color("#787897"))
	inputPromptStyle = style.Foreground(ansi.BrightGreen)
	inputTextStyle   = style.Foreground(ansi.BrightWhite)
)

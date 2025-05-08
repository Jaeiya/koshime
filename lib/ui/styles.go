package ui

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/textinput"
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

	inputPromptStyle = Style.Foreground(ansi.BrightGreen)
	inputTextStyle   = Style.Foreground(ansi.BrightWhite)
)

func NewTextInput() textinput.Model {
	input := textinput.New()
	input.SetWidth(20)
	input.Focus()
	input.Prompt = "   > "
	input.EchoCharacter = '•'
	input.Styles.Focused.Prompt = inputPromptStyle
	input.Styles.Focused.Text = inputTextStyle
	return input
}

type ListOptions struct {
	Items         []list.Item
	ShortHelpKeys []key.Binding
	Width         int
	MaxHeight     int
	ItemsPerPage  int
}

func NewList(o ListOptions) list.Model {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(ansi.BrightGreen).
		BorderForeground(ansi.BrightGreen)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Foreground(ansi.Blue)
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(ansi.White)
	d.Styles.NormalDesc = d.Styles.NormalDesc.Foreground(ansi.BrightBlack)
	d.Styles.FilterMatch = Style

	itemHeight := o.ItemsPerPage * 3 // items plus padding
	offset := 6                      // size of non-items
	height := min(o.MaxHeight, itemHeight+offset)

	l := list.New(o.Items, d, o.Width, height)
	l.SetShowTitle(false)
	l.DisableQuitKeybindings()
	l.FilterInput.VirtualCursor = false

	l.Help.Styles.ShortDesc = HelpDescStyle
	l.Help.Styles.FullDesc = HelpDescStyle
	l.Help.Styles.ShortKey = HelpKeyStyle
	l.Help.Styles.FullKey = HelpKeyStyle
	l.Styles.StatusBar = l.Styles.StatusBar.MarginLeft(3).
		Foreground(HelpKeyStyle.GetForeground())
	l.Styles.StatusBarFilterCount = HelpDescStyle
	l.FilterInput.Styles.Focused.Prompt = l.Styles.Filter.Focused.Prompt.Foreground(ansi.Yellow)

	if o.ShortHelpKeys != nil {
		l.AdditionalShortHelpKeys = func() []key.Binding {
			return o.ShortHelpKeys
		}
		l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	}

	return l
}

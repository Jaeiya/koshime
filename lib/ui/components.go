package ui

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/x/ansi"
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

type ListItem struct {
	index int
	title string
	desc  string
}

// Index should get an index of an external slice, which
// should correlate to the list item. The index can be
// set to 0 if not needed.
func (l ListItem) Index() int          { return l.index }
func (l ListItem) Title() string       { return l.title }
func (l ListItem) Description() string { return l.desc }
func (l ListItem) FilterValue() string { return l.title + " " + l.desc }

func NewListItem(title, desc string, index int) ListItem {
	return ListItem{index, title, desc}
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

	itemsPerPage := min(len(o.Items), o.ItemsPerPage)
	itemHeight := itemsPerPage * 3 // items plus padding
	offset := 6                    // size of non-items
	if len(o.Items) < itemsPerPage {
		offset = 0
	}
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

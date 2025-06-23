package ui

import "github.com/charmbracelet/bubbles/v2/key"

type MainKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Select   key.Binding
	Submit   key.Binding
	EscBack  key.Binding
	Abort    key.Binding
	MainMenu key.Binding
	HelpMore key.Binding
	HelpLess key.Binding
	Back     key.Binding
	Exit     key.Binding
}

var KeyMap = MainKeyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:     key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	Right:    key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
	Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Submit:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	HelpMore: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more help")),
	HelpLess: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "less help")),
	Abort:    key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "abort")),
	EscBack:  key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc/←", "back")),
	Back:     key.NewBinding(key.WithKeys("backspace", "left"), key.WithHelp("←", "back")),
	MainMenu: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "menu")),
	Exit:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "exit")),
}

type KeyHelpInfo[T any] struct {
	ShortHelp func(T) []key.Binding
	FullHelp  func(T) [][]key.Binding
}

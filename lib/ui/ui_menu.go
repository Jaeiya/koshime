package ui

import (
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type (
	MenuIndexMsg int
	OptionFunc   func(m MenuModel) MenuModel
)

type MenuModel struct {
	config struct {
		rotateMenu       bool
		menuDescriptions []string
	}
	menuIndex int
	menuItems []string
}

func WithMenuRotation() OptionFunc {
	return func(m MenuModel) MenuModel {
		m.config.rotateMenu = true
		return m
	}
}

func WithMenuDescriptions(descriptions []string) OptionFunc {
	return func(m MenuModel) MenuModel {
		m.config.menuDescriptions = descriptions
		return m
	}
}

func NewMenuModel(items []string, options ...OptionFunc) MenuModel {
	m := MenuModel{}
	m.menuItems = items

	for _, o := range options {
		m = o(m)
	}

	descLen := len(m.config.menuDescriptions)
	if descLen > 0 && descLen != len(m.menuItems) {
		panic("number of menu descriptions does not match number of menu items")
	}

	return m
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, KeyMap.Up):
			if m.config.rotateMenu {
				m.menuIndex = (m.menuIndex - 1 + len(m.menuItems)) % len(m.menuItems)
			} else {
				if m.menuIndex == 0 {
					break
				}
				m.menuIndex--
			}

		case key.Matches(msg, KeyMap.Down):
			if m.config.rotateMenu {
				m.menuIndex = (m.menuIndex + 1) % len(m.menuItems)
			} else {
				if m.menuIndex+1 == len(m.menuItems) {
					break
				}
				m.menuIndex++
			}

		case key.Matches(msg, KeyMap.Select):
			return m, func() tea.Msg { return MenuIndexMsg(m.menuIndex) }
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	var descStr string

	if len(m.config.menuDescriptions) > 0 {
		descStr = lipgloss.JoinHorizontal(
			lipgloss.Left,
			DisplayMenuItems(m.menuItems, m.menuIndex),
			Style.MarginLeft(3).
				Width(35).
				Foreground(ansi.Magenta).
				Render(m.config.menuDescriptions[m.menuIndex]),
		)
		return descStr
	}

	return DisplayMenuItems(m.menuItems, m.menuIndex)
}

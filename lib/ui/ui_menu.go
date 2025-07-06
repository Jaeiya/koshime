package ui

import (
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type (
	MenuIndexMsg int
	OptionFunc   func(m MenuModel) MenuModel
)

type MenuModel struct {
	config struct {
		rotateMenu       bool
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

func NewMenuModel(items []string, options ...OptionFunc) MenuModel {
	m := MenuModel{}
	m.menuItems = items

	for _, o := range options {
		m = o(m)
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
	return DisplayMenuItems(m.menuItems, m.menuIndex)
}

package ui

import (
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type MenuIndexMsg int

type MenuModel struct {
	menuIndex int
	menuItems []string
}

func NewMenuModel(items []string) MenuModel {
	m := MenuModel{}
	m.menuItems = items
	return m
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, KeyMap.Up):
			m.menuIndex = (m.menuIndex - 1 + len(m.menuItems)) % len(m.menuItems)

		case key.Matches(msg, KeyMap.Down):
			m.menuIndex = (m.menuIndex + 1) % len(m.menuItems)

		case key.Matches(msg, KeyMap.Select):
			return m, func() tea.Msg { return MenuIndexMsg(m.menuIndex) }
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	return DisplayMenuItems(m.menuItems, m.menuIndex)
}

package ui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type (
	MenuItemSelMsg int
	OptionFunc     func(m MenuModel) MenuModel
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
			return m, func() tea.Msg { return MenuItemSelMsg(m.menuIndex) }
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	var descStr string

	if len(m.config.menuDescriptions) > 0 {
		descStr = lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.displayMenuItems(m.menuItems, m.menuIndex),
			Style.MarginLeft(1).
				Width(35).
				Foreground(ansi.Magenta).
				Render(m.config.menuDescriptions[m.menuIndex]),
		)
		return descStr
	}

	return m.displayMenuItems(m.menuItems, m.menuIndex)
}

func (m *MenuModel) Select(idx int) error {
	if idx < 0 || idx >= len(m.menuItems) {
		return fmt.Errorf("cannot select menu item index: out of bounds")
	}
	m.menuIndex = idx
	return nil
}

func (m MenuModel) displayMenuItems(items []string, selectedIndex int) string {
	items = slices.Clone(items)

	if selectedIndex >= len(items) {
		panic("menu index is beyond the item count")
	}

	if selectedIndex < 0 {
		panic("menu index is negative")
	}

	maxItemWidth := 0
	for _, item := range items {
		itemWidth := lipgloss.Width(item)
		if itemWidth > maxItemWidth {
			maxItemWidth = itemWidth
		}
	}

	padding := 3
	caret := " > "
	itemWidth := maxItemWidth + padding + lipgloss.Width(caret)

	menuStyle := TextStyle.MarginLeft(3).
		Width(itemWidth).
		PaddingRight(padding)

	for i, item := range items {
		if i == selectedIndex {
			items[i] = menuStyle.Foreground(ansi.BrightGreen).
				Background(ansi.Black).
				Render(caret + item)
		} else {
			items[i] = menuStyle.Render(strings.Repeat(" ", lipgloss.Width(caret)) + item)
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		items...,
	)
}

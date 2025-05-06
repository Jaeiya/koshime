package ui

import (
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type Consent int

const (
	No = Consent(iota)
	Yes
)

type consentModel struct {
	pos Consent
}

func (consentModel) Init() tea.Cmd {
	return nil
}

func (m consentModel) Update(msg tea.Msg) consentModel {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.Down):
			m.pos = Yes
		case key.Matches(msg, keyMap.Up):
			m.pos = No
		}
	}
	return m
}

func (m consentModel) View(msg ...string) string {
	var yes, no string
	if m.pos == No {
		no = selectNoStyle.Render("> No")
		yes = textStyle.Render("  Yes")
	} else {
		yes = selectYesStyle.Render("> Yes")
		no = textStyle.MarginTop(1).Render("  No")
	}

	msg = append(msg, no, yes)
	return lipgloss.JoinVertical(lipgloss.Left, msg...)
}

// Get returns the current consent value
// and resets the consent position for re-use.
func (m *consentModel) Get() Consent {
	lastPos := m.pos
	// Reset for re-use
	m.pos = No
	return lastPos
}

func (m *consentModel) SetConsentPos(pos Consent) {
	m.pos = pos
}

func (m *consentModel) Reset() {
	m.pos = No
}

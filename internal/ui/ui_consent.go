package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Consent int

const (
	No = Consent(iota)
	Yes
)

type ConsentModel struct {
	pos Consent
}

func (m ConsentModel) Update(msg tea.Msg) ConsentModel {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			m.pos = Yes
		case "k", "up":
			m.pos = No
		}
	}
	return m
}

func (m ConsentModel) View(msg ...string) string {
	var yes, no string
	if m.pos == No {
		no = SelectNoStyle.Render("> No")
		yes = TextStyle.Render("  Yes")
	} else {
		yes = SelectYesStyle.Render("> Yes")
		no = TextStyle.Render("  No")
	}

	msg = append(msg, "", no, yes)
	return lipgloss.JoinVertical(lipgloss.Left, msg...)
}

// Select returns the currently selected consent value
// and resets the consent position to a default of No.
func (m *ConsentModel) Select() Consent {
	lastPos := m.pos
	// Reset for re-use
	m.pos = No
	return lastPos
}

func (m *ConsentModel) SetConsentPos(pos Consent) {
	m.pos = pos
}

func (m *ConsentModel) Reset() {
	m.pos = No
}

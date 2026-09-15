package ui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/logger"
)

type Consent int

const (
	No = Consent(iota)
	Yes
)

type ConsentModel struct {
	pos      Consent
	isActive bool
}

func (m ConsentModel) Update(msg tea.Msg) ConsentModel {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			logger.Log(logger.Hot, "Update(): highlight Yes")
			m.pos = Yes
		case "k", "up":
			logger.Log(logger.Hot, "Update(): highlight No")
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

func (m ConsentModel) ShortHelp() []key.Binding {
	return []key.Binding{KeyMap.Up, KeyMap.Down, KeyMap.Select}
}

func (m ConsentModel) FullHelp() [][]key.Binding {
	return nil
}

// Select returns the currently selected consent value
// and resets the consent position to a default of No.
func (m *ConsentModel) Select() Consent {
	logger.LogFunc(logger.Debug, func() string {
		selStr := "Yes"
		if m.pos == No {
			selStr = "No"
		}
		return fmt.Sprintf("Select(): %s", selStr)
	})
	lastPos := m.pos
	// Allow us to re-use model
	m.Reset()
	return lastPos
}

// Activate sets the 'isActive' flag to true, allowing you to
// then use IsActive() to determine if the model should be
// updated/viewed manually.
func (m *ConsentModel) Activate() {
	m.isActive = true
}

func (m ConsentModel) IsActive() bool {
	return m.isActive
}

func (m *ConsentModel) SetConsentPos(pos Consent) {
	logger.Log(logger.Debug, "SetConsentPos(): %d", pos)
	m.pos = pos
}

func (m *ConsentModel) Reset() {
	m.pos = No
	m.isActive = false
}

package views

import (
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type AnimeDisplayModel struct {
	keys struct {
		displayMode key.Binding
	}
	displayMode ui.DisplayMode
}

func NewAnimeDisplayModel() *AnimeDisplayModel {
	m := &AnimeDisplayModel{}
	m.keys.displayMode = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "display mode"))
	return m
}

func (m *AnimeDisplayModel) Update(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.displayMode):
			switch m.displayMode {
			case ui.Simple:
				m.displayMode = ui.Extended
			case ui.Extended:
				m.displayMode = ui.All
			case ui.All:
				m.displayMode = ui.Simple
			}
		}
	}
}

func (m AnimeDisplayModel) View(ai ui.AnimeInfo) string {
	return ui.DisplayAnimeInfo(ai, m.displayMode)
}

func (m AnimeDisplayModel) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.displayMode}
}

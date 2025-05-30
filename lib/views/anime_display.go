package views

import (
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type AnimeDisplayModel struct {
	keys struct {
		openSynopsis  key.Binding
		closeSynopsis key.Binding
	}
	showSynopsis bool
}

func NewAnimeDisplayModel() *AnimeDisplayModel {
	m := &AnimeDisplayModel{}
	m.keys.openSynopsis = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "open synopsis"))
	m.keys.closeSynopsis = key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "close synopsis"),
	)
	return m
}

func (m *AnimeDisplayModel) Update(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.openSynopsis):
			m.showSynopsis = !m.showSynopsis
		}
	}
}

func (m AnimeDisplayModel) View(ai ui.AnimeInfo) string {
	return ui.DisplayAnimeInfo(ai, m.showSynopsis)
}

func (m AnimeDisplayModel) ShortHelp() []key.Binding {
	if m.showSynopsis {
		return []key.Binding{m.keys.closeSynopsis}
	}
	return []key.Binding{m.keys.openSynopsis}
}

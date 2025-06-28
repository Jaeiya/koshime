
import (
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type Some_View int

const (
	Some_Default = Some_View(iota)
)

type Some_Model struct {
	windowSize tea.WindowSizeMsg
	ui         struct {
		loader ui.LoaderModel
	}
	state Some_State
}

type Some_State struct {
	err error
}

func newSomeModel() Some_Model {
	m := Some_Model{}
	return m
}

func (m Some_Model) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			return m, exitToMenu
		}

	case error:
		m.state.err = msg
	}
	return m, nil
}

func (m Some_Model) View() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.err != nil {
		return ui.DisplayError(m.state.err), nil
	}

	return "", nil
}

func (m Some_Model) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (m Some_Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

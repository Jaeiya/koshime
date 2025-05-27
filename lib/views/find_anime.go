package views

import (
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type Find_AnimeView int

const (
	Find_QueryEntry = Find_AnimeView(iota)
	Find_Results
	Find_SelectedAnime
)

type Find_AnimeHelp map[Find_AnimeView]ui.KeyHelpInfo[Find_AnimeModel]

type Find_AnimeModel struct {
	windowSize struct {
		width  int
		height int
	}
	ui struct {
		animeSearch  *ui.AnimeSearchModel
		animeDisplay *ui.AnimeDisplayModel
	}
	db    *database.Database
	state Find_AnimeState
}

type Find_AnimeState struct {
	view          Find_AnimeView
	selectedAnime ui.AnimeInfo
}

func newFindAnimeModel(db *database.Database) Find_AnimeModel {
	m := Find_AnimeModel{db: db}

	m.ui.animeSearch = ui.NewAnimeSearchModel(db,
		ui.WithHeader("Find Anime"),
		ui.WithExit(),
		ui.WithMinInputLen(4),
		ui.WithItemsPerPage(5),
		ui.WithMaxResults(15),
		ui.WithInputWidth(25),
		ui.WithAnimeSelection(""),
	)

	return m
}

func (m Find_AnimeModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize.width = msg.Width
		m.windowSize.height = msg.Height

	case ui.SelectedAnimeMsg:
		m.state.view = Find_SelectedAnime

	case ui.AnimeSearchExitMsg:
		m.reset()
		return m, abort
	}

	switch m.state.view {
	case Find_QueryEntry:
		cmd = m.ui.animeSearch.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Find_AnimeModel) View() (string, *tea.Cursor) {
	switch m.state.view {
	case Find_QueryEntry:
		return m.ui.animeSearch.View()
	}

	return "missing view", nil
}

func (m Find_AnimeModel) ShortHelp() []key.Binding {
	return m.ui.animeSearch.ShortHelp()
}

func (m Find_AnimeModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m Find_AnimeModel) UpdateSelected(msg tea.Msg) (Find_AnimeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
		}
	}
	return m, nil
}

func (m Find_AnimeModel) ViewAnime() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Find Anime", "Entry Info"),
		"",
		m.ui.animeDisplay.View(m.state.selectedAnime),
	)
}

func (m *Find_AnimeModel) reset() {
	m.state = Find_AnimeState{}
}

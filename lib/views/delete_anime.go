package views

import (
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type DelAnime_View int

const (
	DelAnime_Query = DelAnime_View(iota)
	DelAnime_Deleted
)

type (
	DelAnimeErrorMsg   = error
	DelAnimeSuccessMsg struct{}
	DelAnime_Help      map[DelAnime_View]ui.KeyHelpInfo[DelAnime_Model]
)

type DelAnime_Model struct {
	windowSize struct {
		width  int
		height int
	}
	ui struct {
		animeSearch *AnimeSearchModel
		loader      ui.LoaderModel
	}
	db      *database.Database
	helpMap DelAnime_Help
	state   DelAnime_State
}

type DelAnime_State struct {
	err           error
	view          DelAnime_View
	selectedAnime ui.AnimeInfo
}

func newDelAnimeModel(db *database.Database) DelAnime_Model {
	m := DelAnime_Model{db: db}
	m.ui.animeSearch = NewAnimeSearchModel(
		db,
		WithHeader("Delete Anime"),
		WithExit(),
		WithMaxResults(5),
		WithItemsPerPage(5),
		WithMinInputLen(3),
		WithInputWidth(30),
		WithAnimeSelection("Are you sure you want to delete the above Anime?"),
		WithLocalSource(),
	)

	m.ui.loader = ui.NewLoader()

	m.helpMap = DelAnime_Help{
		DelAnime_Deleted: {
			ShortHelp: func(da DelAnime_Model) []key.Binding {
				return []key.Binding{ui.KeyMap.Select}
			},
		},
	}
	return m
}

func (m DelAnime_Model) Init() tea.Cmd {
	return nil
}

func (m DelAnime_Model) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize.width = msg.Width
		m.windowSize.height = msg.Height

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.state.err != nil {
				m.reset()
				return m, nil
			}
		}

	case AnimeSearchExitMsg:
		m.reset()
		return m, exitToMenu

	case SelectedAnimeMsg:
		m.ui.loader, cmd = m.ui.loader.Start("Deleting Anime")
		m.state.selectedAnime = msg
		m.state.view = DelAnime_Deleted
		return m, tea.Batch(cmd, m.deleteAnime)

	}

	switch m.state.view {
	case DelAnime_Query:
		cmd = m.ui.animeSearch.Update(msg)
		cmds = append(cmds, cmd)
	case DelAnime_Deleted:
		m, cmd = m.UpdateDeleted(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m DelAnime_Model) View() (string, *tea.Cursor) {
	switch m.state.view {
	case DelAnime_Query:
		return m.ui.animeSearch.View()
	case DelAnime_Deleted:
		return m.ViewDeleted()
	}
	return "DelAnime::missing view", nil
}

func (m DelAnime_Model) ShortHelp() []key.Binding {
	if m.state.err != nil {
		return []key.Binding{ui.KeyMap.EscBack}
	}

	if m.state.view == DelAnime_Query {
		return m.ui.animeSearch.ShortHelp()
	}

	return m.helpMap[m.state.view].ShortHelp(m)
}

func (m DelAnime_Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m DelAnime_Model) UpdateDeleted(msg tea.Msg) (DelAnime_Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.state.err == nil {
				m.reset()
				return m, exitToMenu
			}

		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.state.err != nil {
				m.reset()
			}
		}

	case DelAnimeSuccessMsg:
		m.ui.loader.Stop()

	case DelAnimeErrorMsg:
		m.ui.loader.Stop()
		m.state.err = msg
	}
	return m, nil
}

func (m DelAnime_Model) ViewDeleted() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.err != nil {
		return ui.DisplayError(m.state.err), nil
	}

	str := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Delete Anime", "Success"),
		ui.TextStyle.MarginTop(1).Render(utils.ColorText("Anime Successfully Deleted")),
		ui.TextStyle.MarginTop(1).Foreground(ansi.BrightGreen).Render("> Continue"),
	)
	return str, nil
}

func (m *DelAnime_Model) reset() {
	m.state = DelAnime_State{}
	m.state.view = DelAnime_Query
	m.ui.animeSearch.Reset()
}

func (m DelAnime_Model) deleteAnime() tea.Msg {
	p := m.db.Profile()
	_, err := kitsu.DeleteLibAnime(m.state.selectedAnime.LibID, p.AccessToken)
	if err != nil {
		return DelAnimeErrorMsg(err)
	}
	err = m.db.DeleteAnimeById(m.state.selectedAnime.LibID)
	if err != nil {
		return DelAnimeErrorMsg(err)
	}
	return DelAnimeSuccessMsg{}
}

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

type (
	DelAnime_View int
)

const (
	DelAnime_Query = DelAnime_View(iota)
	DelAnime_Consent
	DelAnime_Success
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
		animeSearch *ui.AnimeSearchModel
		loader      ui.LoaderModel
		consent     ui.ConsentModel
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
	m.ui.loader = ui.NewLoader()
	m.ui.animeSearch = ui.NewAnimeSearchModel(
		db,
		ui.WithHeader("Delete Anime"),
		ui.WithMaxResults(5),
		ui.WithItemsPerPage(5),
		ui.WithMinInputLen(3),
		ui.WithInputWidth(30),
		ui.WithLocalSource(),
	)

	m.helpMap = DelAnime_Help{
		DelAnime_Consent: {
			ShortHelp: func(da DelAnime_Model) []key.Binding {
				return []key.Binding{
					ui.KeyMap.Up,
					ui.KeyMap.Down,
					ui.KeyMap.Select,
					ui.KeyMap.EscBack,
				}
			},
		},
		DelAnime_Success: {
			ShortHelp: func(da DelAnime_Model) []key.Binding {
				return []key.Binding{ui.KeyMap.Select}
			},
		},
	}
	return m
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

	case ui.AnimeSearchExitMsg:
		m.reset()
		return m, abort

	case ui.SelectedAnimeMsg:
		m.state.view = DelAnime_Consent
		m.state.selectedAnime = msg

	case DelAnimeErrorMsg:
		m.ui.loader.Stop()
		m.state.err = msg

	case DelAnimeSuccessMsg:
		m.ui.loader.Stop()
		m.state.view = DelAnime_Success

	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case DelAnime_Query:
		cmd = m.ui.animeSearch.Update(msg)
		cmds = append(cmds, cmd)
	case DelAnime_Consent:
		m, cmd = m.UpdateConsent(msg)
		cmds = append(cmds, cmd)
	case DelAnime_Success:
		m, cmd = m.UpdateSuccess(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m DelAnime_Model) View() (string, *tea.Cursor) {
	if m.state.err != nil {
		return ui.DisplayError(m.state.err), nil
	}

	switch m.state.view {
	case DelAnime_Query:
		return m.ui.animeSearch.View()
	case DelAnime_Consent:
		return m.ViewConsent()
	case DelAnime_Success:
		return m.ViewSuccess()
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

func (m DelAnime_Model) UpdateConsent(msg tea.Msg) (DelAnime_Model, tea.Cmd) {
	var cmd tea.Cmd
	m.ui.consent = m.ui.consent.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Submit):
			if m.ui.consent.Select() == ui.No {
				m.state.view = DelAnime_Query
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Deleting Anime")
			return m, tea.Batch(cmd, m.deleteAnime)

		case key.Matches(msg, ui.KeyMap.EscBack):
			m.state.view = DelAnime_Query
		}
	}
	return m, nil
}

func (m DelAnime_Model) ViewConsent() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	str := lipgloss.JoinVertical(
		lipgloss.Left,
		newViewHeader("Delete Anime")("Consent"),
		ui.Style.MarginTop(1).Render(ui.DisplayAnimeInfo(m.state.selectedAnime, false)),
		m.ui.consent.View(
			"",
			ui.TextStyle.Render(
				utils.ColorText(";b;Are you sure you want to delete the above Anime?"),
			),
			"",
		),
	)

	return str, nil
}

func (m DelAnime_Model) UpdateSuccess(msg tea.Msg) (DelAnime_Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			return m, abort
		}
	}
	return m, nil
}

func (m DelAnime_Model) ViewSuccess() (string, *tea.Cursor) {
	str := lipgloss.JoinVertical(
		lipgloss.Left,
		newViewHeader("Delete Anime")("Success"),
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
	p := m.db.GetProfile()
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

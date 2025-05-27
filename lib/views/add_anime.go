package views

import (
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type (
	Add_AnimeView int
)

const (
	AddAnime_Query = Add_AnimeView(iota)
	AddAnime_Review
	AddAnime_Rss
	AddAnime_RssResults
	AddAnime_RssReview
)

type (
	AnimeAddedMsg struct{}
	AddAnime_Help map[Add_AnimeView]ui.KeyHelpInfo[AddAnime_Model]
)

type AddAnime_Model struct {
	windowSize struct {
		width  int
		height int
	}
	config struct {
		maxInputWidth   int
		minInputLen     int // Min required chars to submit input
		itemsPerPage    int // How many list items per page
		maxAnimeResults int // Max number of results to search kitsu for
	}
	ui struct {
		loader      ui.LoaderModel
		input       textinput.Model
		list        list.Model
		consent     ui.ConsentModel
		animeSearch *ui.AnimeSearchModel
	}
	helpMap AddAnime_Help
	db      *database.Database
	state   Add_AnimeModelState
}

type Add_AnimeModelState struct {
	view          Add_AnimeView
	fetchErr      error
	selectedAnime ui.AnimeInfo
}

func newAddAnimeModel(db *database.Database) AddAnime_Model {
	m := AddAnime_Model{db: db}
	m.config.minInputLen = 4
	m.config.maxInputWidth = 30
	m.config.itemsPerPage = 5
	m.config.maxAnimeResults = 10

	m.ui.animeSearch = ui.NewAnimeSearchModel(
		db,
		ui.WithHeader("Add Anime"),
		ui.WithExit(),
		ui.WithMinInputLen(4),
		ui.WithItemsPerPage(5),
		ui.WithMaxResults(10),
		ui.WithInputWidth(30),
		ui.WithAnimeSelection("Do you want to add the above anime to your library?"),
		ui.WithKitsuSource([]kitsu.AnimeStatus{kitsu.AnimeNew}),
	)

	m.ui.list = ui.NewList(ui.ListOptions{})
	m.ui.input = ui.NewTextInput()
	m.ui.input.SetWidth(m.config.maxInputWidth)
	m.ui.input.Placeholder = "Enter query"
	m.ui.loader = ui.NewLoader()

	m.helpMap = AddAnime_Help{
		AddAnime_Query: {
			ShortHelp: func(AddAnime_Model) []key.Binding {
				return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.Abort}
			},
		},
	}
	return m
}

func (m AddAnime_Model) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize.width = msg.Width
		m.windowSize.height = msg.Height

	case ui.AnimeSearchExitMsg:
		m.reset()
		return m, abort

	case ui.SelectedAnimeMsg:
		m.ui.loader, cmd = m.ui.loader.Start("Adding Anime")
		m.state.view = AddAnime_Review
		m.state.selectedAnime = msg
		return m, tea.Batch(cmd, m.addAnime(msg.ID))
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case AddAnime_Query:
		cmd = m.ui.animeSearch.Update(msg)
		cmds = append(cmds, cmd)
	case AddAnime_Review:
		m, cmd = m.UpdateReview(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AddAnime_Model) View() (string, *tea.Cursor) {
	switch m.state.view {
	case AddAnime_Query:
		return m.ui.animeSearch.View()
	case AddAnime_Review:
		return m.ViewReview()
	}
	return "AddAnime::missing view", nil
}

func (m AddAnime_Model) ShortHelp() []key.Binding {
	if m.state.fetchErr != nil {
		return []key.Binding{ui.KeyMap.EscBack}
	}

	if m.state.view == AddAnime_Query {
		return m.ui.animeSearch.ShortHelp()
	}

	if bindings, exists := m.helpMap[m.state.view]; exists {
		return bindings.ShortHelp(m)
	}

	return []key.Binding{}
}

func (m AddAnime_Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m AddAnime_Model) UpdateReview(msg tea.Msg) (AddAnime_Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.state.fetchErr == nil {
				m.reset()
				return m, abort
			}

		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.state.fetchErr != nil {
				m.reset()
			}
		}

	case AnimeAddedMsg:
		m.ui.loader.Stop()

	case FetchErrorMsg:
		m.ui.loader.Stop()
		m.state.fetchErr = msg
	}
	return m, nil
}

func (m AddAnime_Model) ViewReview() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.fetchErr != nil {
		return ui.DisplayError(m.state.fetchErr), nil
	}

	str := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Add Anime", "Success"),
		ui.TextStyle.MarginTop(1).Render(utils.ColorText("Anime Successfully Added")),
		ui.TextStyle.MarginTop(1).Foreground(ansi.BrightGreen).Render("> Continue"),
	)
	return str, nil
}

func (m *AddAnime_Model) reset() {
	m.state = Add_AnimeModelState{
		view: AddAnime_Query,
	}
	m.ui.animeSearch.Reset()
	m.ui.input.Reset()
}

func (m AddAnime_Model) addAnime(animeID string) tea.Cmd {
	return func() tea.Msg {
		p := m.db.GetProfile()
		libID, err := kitsu.AddAnime(animeID, p.ID, p.AccessToken, kitsu.LibAnimeWatching)
		if err != nil {
			return FetchErrorMsg(err)
		}

		anime := m.state.selectedAnime
		err = m.db.AddAnime(kitsu.LibraryEntry{
			ID:        animeID,
			LibID:     libID,
			JPN_Title: anime.JpnTitle,
			ENG_Title: anime.EngTitle,
			AltTitles: anime.AltTitles,
			Episodes:  anime.Episodes,
			Type:      anime.ShowType,
			Status:    anime.Status,
			Progress:  anime.Progress,
			Synopsis:  anime.Synopsis,
			Slug:      anime.Slug,
		})
		if err != nil {
			return FetchErrorMsg(err)
		}
		return AnimeAddedMsg{}
	}
}

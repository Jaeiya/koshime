package views

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/x/ansi"
)

type AddAnime_View int

const (
	AddAnime_Selection = AddAnime_View(iota)
	AddAnime_Query
	AddAnime_Review
)

type (
	AnimeAddedMsg struct{}
	AddAnime_Help map[AddAnime_View]ui.KeyHelpInfo[AddAnime_Model]
)

type AddAnime_Model struct {
	windowSize tea.WindowSizeMsg
	config     struct {
		maxInputWidth   int
		minInputLen     int // Min required chars to submit input
		itemsPerPage    int // How many list items per page
		maxAnimeResults int // Max number of results to search kitsu for
	}
	ui struct {
		loader      ui.LoaderModel
		input       textinput.Model
		consent     ui.ConsentModel
		animeSearch *AnimeSearchModel
	}
	helpMap AddAnime_Help
	db      *database.Database
	state   AddAnime_State
}

type AddAnime_State struct {
	view          AddAnime_View
	fetchErr      error
	selectedAnime ui.AnimeInfo
	menuIndex     int
}

func newAddAnimeModel(db *database.Database) AddAnime_Model {
	m := AddAnime_Model{db: db}
	m.config.minInputLen = 4
	m.config.maxInputWidth = 30
	m.config.itemsPerPage = 5
	m.config.maxAnimeResults = 10

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

func (m AddAnime_Model) Init() tea.Cmd {
	return nil
}

func (m AddAnime_Model) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case AnimeSearchExitMsg:
		m.reset()
		return m, exitToMenu

	case SelectedAnimeMsg:
		m.ui.loader, cmd = m.ui.loader.Start("Adding Anime")
		m.state.view = AddAnime_Review
		m.state.selectedAnime = msg.Value
		return m, tea.Batch(cmd, m.addAnime(msg.Value.ID))
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case AddAnime_Selection:
		m, cmd = m.UpdateSelection(msg)
		cmds = append(cmds, cmd)
	case AddAnime_Query:
		cmd = m.ui.animeSearch.Update(msg)
		cmds = append(cmds, cmd)
	case AddAnime_Review:
		m, cmd = m.UpdateReview(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AddAnime_Model) View() tea.View {
	switch m.state.view {
	case AddAnime_Selection:
		return m.ViewSelection()
	case AddAnime_Query:
		return m.ui.animeSearch.View()
	case AddAnime_Review:
		return m.ViewReview()
	}
	return tea.NewView("missing AddAnime view")
}

func (m AddAnime_Model) ShortHelp() []key.Binding {
	if m.state.fetchErr != nil {
		return []key.Binding{ui.KeyMap.EscBack}
	}

	if m.state.view == AddAnime_Selection {
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select, ui.KeyMap.EscBack}
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

func (m AddAnime_Model) UpdateSelection(msg tea.Msg) (AddAnime_Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			return m, exitToMenu

		case key.Matches(msg, ui.KeyMap.Up):
			if m.state.menuIndex == 0 {
				return m, nil
			}
			m.state.menuIndex -= 1
			return m, nil

		case key.Matches(msg, ui.KeyMap.Down):
			if m.state.menuIndex == 1 {
				return m, nil
			}
			m.state.menuIndex += 1
			return m, nil

		case key.Matches(msg, ui.KeyMap.Select):
			switch m.state.menuIndex {
			// Airing or Upcoming anime
			case 0:
				m.InitAnimeSearch([]kitsu.AnimeStatus{kitsu.AnimeNew})
			// Completed anime
			case 1:
				m.InitAnimeSearch([]kitsu.AnimeStatus{kitsu.AnimeFinished})
			}
			m.ui.animeSearch.Update(m.windowSize)
			m.state.view = AddAnime_Query
		}
	}
	return m, nil
}

func (m AddAnime_Model) ViewSelection() tea.View {
	var menu string
	menuStyle := ui.Style.MarginLeft(3).
		Width(15).
		Background(ansi.Black).
		Foreground(ansi.BrightGreen)

	if m.state.menuIndex == 0 {
		menu = lipgloss.JoinVertical(
			lipgloss.Left,
			menuStyle.Render(" > Airing"),
			ui.TextStyle.Render("   Completed"),
		)
	} else {
		menu = lipgloss.JoinVertical(
			lipgloss.Left,
			ui.TextStyle.Render("   Airing"),
			menuStyle.Render(" > Completed"),
		)
	}

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Add Anime"),
		"",
		ui.DisplayText([]string{
			`If the anime you want to add is ;m;not;x; currently airing, but
;g;will;x; air in the future, then select ;dc;airing;x;.`,
			`If you know the anime has already been out for several months or more,
then it's probably ;dc;completed;x;.`,
		}, 1),
		menu,
	)

	return tea.NewView(view)
}

func (m AddAnime_Model) UpdateReview(msg tea.Msg) (AddAnime_Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.state.fetchErr == nil {
				m.reset()
				return m, exitToMenu
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

func (m AddAnime_Model) ViewReview() tea.View {
	if m.ui.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.ui.loader.View()))
	}

	if m.state.fetchErr != nil {
		return tea.NewView(ui.DisplayError(m.state.fetchErr))
	}

	str := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Add Anime", "Success"),
		ui.TextStyle.MarginTop(1).Render(utils.ColorText("Anime Successfully Added")),
		ui.TextStyle.MarginTop(1).Foreground(ansi.BrightGreen).Render("> Continue"),
	)
	return tea.NewView(str)
}

func (m *AddAnime_Model) InitAnimeSearch(animeStatus []kitsu.AnimeStatus) {
	m.ui.animeSearch = NewAnimeSearchModel(
		m.db,
		WithHeader("Add Anime"),
		WithExit(),
		WithMinInputLen(4),
		WithItemsPerPage(5),
		WithMaxResults(10),
		WithInputWidth(30),
		WithAnimeSelection("Do you want to add the above anime to your library?"),
		WithKitsuSource(animeStatus),
	)
}

func (m *AddAnime_Model) reset() {
	m.state = AddAnime_State{
		view: AddAnime_Selection,
	}
	m.ui.animeSearch.Reset()
	m.ui.input.Reset()
}

func (m AddAnime_Model) addAnime(animeID string) tea.Cmd {
	return func() tea.Msg {
		p := m.db.Profile()
		libID, err := kitsu.AddAnime(animeID, p.ID, p.AccessToken, kitsu.LibAnimeWatching)
		if err != nil {
			return FetchErrorMsg{Msg: err.Error()}
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
			Progress:  0,
			AvgRating: anime.AvgRating,
			Synopsis:  anime.Synopsis,
			Slug:      anime.Slug,
			QbtFeed: struct {
				Name    string
				RuleURI string
			}{},
		})
		if err != nil {
			return FetchErrorMsg{
				Msg: fmt.Errorf("failed to add anime to database: %w", err).Error(),
			}
		}
		return AnimeAddedMsg{}
	}
}

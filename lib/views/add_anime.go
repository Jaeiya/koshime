package views

import (
	"fmt"

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
	AddAnime_Results
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
		loader  ui.LoaderModel
		input   textinput.Model
		list    list.Model
		consent ui.ConsentModel
	}
	keys struct {
		openSynopsis  key.Binding
		closeSynopsis key.Binding
	}
	helpMap AddAnime_Help
	db      *database.Database
	state   Add_AnimeModelState
}

type Add_AnimeModelState struct {
	view          Add_AnimeView
	fetchErr      error
	results       []ui.AnimeInfo
	selectedAnime ui.AnimeInfo
	showSynopsis  bool
	animeAdded    bool
}

func newAddAnimeModel(db *database.Database) AddAnime_Model {
	m := AddAnime_Model{db: db}
	m.config.minInputLen = 4
	m.config.maxInputWidth = 30
	m.config.itemsPerPage = 5
	m.config.maxAnimeResults = 10

	m.ui.list = ui.NewList(ui.ListOptions{})
	m.ui.input = ui.NewTextInput()
	m.ui.input.SetWidth(m.config.maxInputWidth)
	m.ui.input.Placeholder = "Enter query"
	m.ui.loader = ui.NewLoader()

	m.keys.openSynopsis = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "open synopsis"))
	m.keys.closeSynopsis = key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "close synopsis"),
	)

	m.helpMap = AddAnime_Help{
		AddAnime_Query: {
			ShortHelp: func(AddAnime_Model) []key.Binding {
				return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.Abort}
			},
		},
		AddAnime_Results: {
			ShortHelp: func(m AddAnime_Model) []key.Binding {
				if !m.ui.loader.IsLoading() && len(m.state.results) == 0 {
					return []key.Binding{ui.KeyMap.EscBack}
				}
				return []key.Binding{}
			},
		},
		AddAnime_Review: {
			ShortHelp: func(m AddAnime_Model) []key.Binding {
				if m.state.animeAdded {
					return []key.Binding{}
				}
				synKey := m.keys.openSynopsis
				if m.state.showSynopsis {
					synKey = m.keys.closeSynopsis
				}
				return []key.Binding{synKey, ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
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
	}

	loader := m.ui.loader
	if loader.IsLoading() {
		m.ui.loader, cmd = loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case AddAnime_Query:
		m, cmd = m.UpdateAnimeQuery(msg)
		cmds = append(cmds, cmd)
	case AddAnime_Results:
		m, cmd = m.UpdateAnimeResults(msg)
		cmds = append(cmds, cmd)
	case AddAnime_Review:
		m, cmd = m.UpdateAnimeReview(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AddAnime_Model) View() (string, *tea.Cursor) {
	switch m.state.view {
	case AddAnime_Query:
		return m.ViewQueryAnime()
	case AddAnime_Results:
		return m.ViewAnimeResults()
	case AddAnime_Review:
		return m.ViewAnimeReview()
	}
	return "AddAnime::missing view", nil
}

func (m AddAnime_Model) ShortHelp() []key.Binding {
	if m.state.fetchErr != nil {
		return []key.Binding{ui.KeyMap.EscBack}
	}
	if bindings, exists := m.helpMap[m.state.view]; exists {
		return bindings.ShortHelp(m)
	}
	return []key.Binding{}
}

func (m AddAnime_Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m AddAnime_Model) UpdateAnimeQuery(msg tea.Msg) (AddAnime_Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			return m, abort

		case key.Matches(msg, ui.KeyMap.Submit):
			// Do not submit query below min input length
			if utils.RuneCount(m.ui.input.Value()) < m.config.minInputLen {
				break
			}

			m.ui.loader, cmd = m.ui.loader.Start("Finding Anime")
			cmds = append(cmds, cmd)

			cmd = ui.FindKitsuAnime(
				m.ui.input.Value(),
				m.config.maxAnimeResults,
				[]kitsu.AnimeStatus{kitsu.AnimeNew},
			)
			cmds = append(cmds, cmd)

			m.state.view = AddAnime_Results
			return m, tea.Batch(cmds...)
		}
	}

	m.ui.input, cmd = m.ui.input.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m AddAnime_Model) ViewQueryAnime() (string, *tea.Cursor) {
	c := m.ui.input.Cursor()
	c.Shape = tea.CursorBar

	header := ui.Style.MarginBottom(1).Render(addAnimeMsgs.header)
	body := lipgloss.JoinVertical(lipgloss.Left, header, addAnimeMsgs.queryDesc)

	c.Y += lipgloss.Height(body)
	inputView := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.ui.input.View(),
		ui.DisplayCharLimit(m.config.minInputLen, m.ui.input.Value()),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		body,
		inputView,
	), c
}

func (m AddAnime_Model) UpdateAnimeResults(msg tea.Msg) (AddAnime_Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			// Delegate esc key to list, during filter operations
			if m.ui.list.FilterState() > list.Unfiltered {
				break
			}
			m.reset()

		case key.Matches(msg, ui.KeyMap.Back):
			m.reset()

		case key.Matches(msg, ui.KeyMap.Select):
			// Do not attempt to select an item while filtering
			if m.ui.list.FilterState() == list.Filtering {
				break
			}
			item := m.ui.list.SelectedItem().(ui.ListItem)
			m.state.view = AddAnime_Review
			return m, func() tea.Msg {
				return m.state.results[item.Index()]
			}
		}

	case FetchErrorMsg:
		m.state.fetchErr = msg

	case ui.AnimeFinderResult:
		m.ui.loader.Stop()
		m.state.results = msg.InfoItems
		m.ui.list = ui.NewList(
			ui.ListOptions{
				Items:         msg.ListItems,
				ShortHelpKeys: []key.Binding{ui.KeyMap.Back},
				Width:         m.windowSize.width,
				MaxHeight:     int(float64(m.windowSize.height) * 0.66),
				ItemsPerPage:  m.config.itemsPerPage,
			},
		)
	}

	m.ui.list, cmd = m.ui.list.Update(msg)
	return m, cmd
}

func (m AddAnime_Model) ViewAnimeResults() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if len(m.state.results) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			findAnimeMsgs.viewHeader("Results"),
			ui.TextStyle.MarginTop(1).Render(
				utils.ColorText(
					fmt.Sprintf("No results found for: ;y;%s", m.ui.input.Value()),
				),
			),
		), nil
	}

	if m.state.fetchErr != nil {
		return ui.DisplayError(m.state.fetchErr), nil
	}

	h := addAnimeMsgs.viewHeader("Results")
	var c *tea.Cursor
	// The filter has no margin, so we enforce
	if m.ui.list.FilterState() == list.Filtering {
		h = ui.Style.MarginBottom(1).Render(h)
		c = m.ui.list.FilterInput.Cursor()
		c.Shape = tea.CursorBlock
		c.Color = ansi.Yellow
		c.Y += lipgloss.Height(h)
		c.X += 2 // Adjust for custom margin
	}
	return lipgloss.JoinVertical(lipgloss.Left, h, m.ui.list.View()), c
}

func (m AddAnime_Model) UpdateAnimeReview(msg tea.Msg) (AddAnime_Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.state.fetchErr != nil {
				m.reset()
			} else {
				m.state.view = AddAnime_Results
			}

		case key.Matches(msg, m.keys.openSynopsis):
			m.state.showSynopsis = !m.state.showSynopsis

		case key.Matches(msg, ui.KeyMap.Submit):
			if m.state.fetchErr != nil || m.ui.loader.IsLoading() || m.state.animeAdded {
				break
			}
			if m.ui.consent.Select() == ui.No {
				m.state.view = AddAnime_Results
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Adding Anime")
			return m, tea.Batch(cmd, m.addAnime(m.state.selectedAnime.ID))
		}

	case AnimeAddedMsg:
		m.ui.loader.Stop()
		m.state.animeAdded = true

	case FetchErrorMsg:
		m.ui.loader.Stop()
		m.state.fetchErr = msg

	case ui.AnimeInfo:
		m.state.selectedAnime = msg
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m AddAnime_Model) ViewAnimeReview() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.fetchErr != nil {
		return ui.DisplayError(m.state.fetchErr), nil
	}

	if m.state.animeAdded {
		return ui.TextStyle.MarginTop(1).Render("Anime Added Successfully"), nil
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		findAnimeMsgs.viewHeader("Entry Info"),
		"",
		ui.DisplayAnimeInfo(m.state.selectedAnime, m.state.showSynopsis),
		ui.TextStyle.MarginTop(1).Render(
			m.ui.consent.View(utils.ColorText(";b;Would you like to add the above anime to your library?"), ""),
		),
	), nil
}

func (m *AddAnime_Model) reset() {
	m.state = Add_AnimeModelState{
		view: AddAnime_Query,
	}
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

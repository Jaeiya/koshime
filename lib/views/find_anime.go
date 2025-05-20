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

type FindAnimeView int

const (
	Find_QueryEntry = FindAnimeView(iota)
	Find_Results
	Find_SelectedAnime
)

var findAnimeHelpMap = map[FindAnimeView]HelpInfo[findAnimeModel]{
	Find_QueryEntry: {
		ShortHelp: func(findAnimeModel) []key.Binding {
			return []key.Binding{keyMap.Submit, keyMap.Abort}
		},
	},
	Find_SelectedAnime: {
		ShortHelp: func(findAnimeModel) []key.Binding {
			return []key.Binding{keyMap.EscBack}
		},
	},
}

type findAnimeModel struct {
	windowSize struct {
		width  int
		height int
	}
	config struct {
		minInputLen  int
		itemsPerPage int
		maxResults   int
	}
	ui struct {
		list   list.Model
		input  textinput.Model
		loader ui.LoaderModel
	}
	keys struct {
		tab key.Binding
	}
	db             *database.Database
	animeFinderMap map[AnimeSource]AnimeFinder
	sourceStrMap   map[AnimeSource]string
	state          findAnimeState
}

type findAnimeState struct {
	fetchErr      FetchErrorMsg
	view          FindAnimeView
	animeResults  []ui.AnimeInfo
	selectedIndex int
	find          struct {
		source   AnimeSource
		failed   bool
		notFound bool
	}
}

func NewFindAnimeModel(db *database.Database) findAnimeModel {
	m := findAnimeModel{db: db}
	m.ui.list = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	// Prevent esc/q key from sending tea.Quit from inside list
	m.ui.list.DisableQuitKeybindings()

	m.ui.input = ui.NewTextInput()
	m.ui.input.Focus()
	m.ui.input.Placeholder = "Enter your query"
	m.ui.input.SetWidth(20)

	m.ui.loader = ui.NewLoader()

	m.config.minInputLen = 4  // Minimum characters to submit search
	m.config.itemsPerPage = 5 // Max list items to display per page
	m.config.maxResults = 10  // Max results to find per search

	m.keys.tab = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "source"))

	m.sourceStrMap = map[AnimeSource]string{
		Kitsu: findAnimeMsgs.kitsu,
		Local: findAnimeMsgs.local,
	}

	m.animeFinderMap = map[AnimeSource]AnimeFinder{
		Kitsu: NewKitsuAnimeFinder(
			10,
			[]kitsu.AnimeStatus{kitsu.AnimeNew, kitsu.AnimeFinished},
		),
		Local: NewLocalAnimeFinder(10, db),
	}
	return m
}

func (m findAnimeModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize.width = msg.Width
		m.windowSize.height = msg.Height

	case FetchErrorMsg:
		m.ui.loader.Stop()
		m.state.view = Find_Results
		m.state.find.failed = true
		m.state.fetchErr = msg
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case Find_QueryEntry:
		m, cmd = m.UpdateQueryEntry(msg)
		cmds = append(cmds, cmd)
	case Find_Results:
		m, cmd = m.UpdateResults(msg)
		cmds = append(cmds, cmd)
	case Find_SelectedAnime:
		m, cmd = m.UpdateAnime(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m findAnimeModel) View() (string, *tea.Cursor) {
	switch m.state.view {
	case Find_QueryEntry:
		return m.ViewQueryEntry()

	case Find_Results:
		return m.ViewResults()

	case Find_SelectedAnime:
		return m.ViewAnime(), nil
	}

	return "missing view", nil
}

func (m findAnimeModel) UpdateQueryEntry(msg tea.Msg) (findAnimeModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			m.Reset()
			return m, abort

		case key.Matches(msg, keyMap.Submit):
			hasShortInput := utils.RuneCount(m.ui.input.Value()) < m.config.minInputLen

			if m.ui.loader.IsLoading() || hasShortInput {
				break
			}

			m.state.view = Find_Results
			m.ui.loader, cmd = m.ui.loader.Start("Find Anime")
			return m, tea.Batch(cmd, m.findAnime(m.ui.input.Value()))

		case key.Matches(msg, m.keys.tab):
			m.state.find.source = (m.state.find.source + 1) % 2
		}
	}

	m.ui.input, cmd = m.ui.input.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m findAnimeModel) ViewQueryEntry() (string, *tea.Cursor) {
	c := m.ui.input.Cursor()
	c.Shape = tea.CursorBar

	search := ui.TextStyle.Foreground(ansi.BrightBlack).
		Render("Source: " + m.sourceStrMap[m.state.find.source])

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		findAnimeMsgs.header,
		findAnimeMsgs.title,
		search,
		ui.Style.MarginTop(1).Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.ui.input.View(),
			ui.DisplayCharLimit(m.config.minInputLen, m.ui.input.Value()),
		)),
	)
	c.Y += lipgloss.Height(view) - 1
	return lipgloss.JoinVertical(lipgloss.Left, view), c
}

func (m findAnimeModel) UpdateResults(msg tea.Msg) (findAnimeModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		// Go back to query-entry-view from results-view
		case key.Matches(msg, keyMap.MainMenu):
			// List needs 'Esc' control to cancel filter
			if m.ui.list.FilterState() > list.Unfiltered {
				break
			}
			m.Reset()

		// Go back to query-entry-view from results-view
		case key.Matches(msg, keyMap.Back):
			if m.ui.list.FilterState() != list.Filtering {
				m.Reset()
			}

		// Select Anime
		case key.Matches(msg, keyMap.Submit):
			// List needs 'Enter' control for applying filter
			if m.ui.list.FilterState() == list.Filtering {
				break
			}
			if !m.state.find.notFound {
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.selectedIndex = item.Index()
				m.state.view = Find_SelectedAnime
			}

		}

	case FetchedNoResultsMsg:
		m.ui.loader.Stop()
		m.state.find.notFound = true

	case AnimeFinderResult:
		m.ui.loader.Stop()
		m.state.view = Find_Results
		m.state.animeResults = msg.infoItems

		m.ui.list = ui.NewList(
			ui.ListOptions{
				Items:         msg.listItems,
				ShortHelpKeys: []key.Binding{keyMap.Back},
				Width:         m.windowSize.width,
				MaxHeight:     int(float64(m.windowSize.height) * 0.66),
				ItemsPerPage:  m.config.itemsPerPage,
			},
		)
	}

	m.ui.list, cmd = m.ui.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m findAnimeModel) ViewResults() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.find.failed {
		return m.state.fetchErr.Error(), nil
	}

	if m.state.find.notFound {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			findAnimeMsgs.viewHeader("Results"),
			findAnimeMsgs.notFound(
				m.ui.input.Value(),
				m.state.find.source.String(),
			),
		), nil
	}

	h := findAnimeMsgs.viewHeader("Results")
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

func (m findAnimeModel) UpdateAnime(msg tea.Msg) (findAnimeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.EscBack, keyMap.Back):
			m.state.view = Find_Results
		}
	}
	return m, nil
}

func (m findAnimeModel) ViewAnime() string {
	if m.state.animeResults != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			findAnimeMsgs.viewHeader("Entry Info"),
			"",
			ui.DisplayAnimeInfo(m.state.animeResults[m.state.selectedIndex]),
		)
	}

	return "missing local or kitsu results to display"
}

func (m findAnimeModel) ShortHelp() []key.Binding {
	if m.ui.loader.IsLoading() {
		return []key.Binding{}
	}

	if v, exists := findAnimeHelpMap[m.state.view]; exists {
		return v.ShortHelp(m)
	}

	return []key.Binding{}
}

func (m findAnimeModel) FullHelp() [][]key.Binding {
	if v, exists := findAnimeHelpMap[m.state.view]; exists {
		return v.FullHelp(m)
	}
	return [][]key.Binding{}
}

func (m *findAnimeModel) Reset() {
	// Do not reset source
	source := m.state.find.source
	m.state = findAnimeState{}
	m.state.find.source = source
	m.ui.input.Reset()
}

func (m *findAnimeModel) findAnime(query string) tea.Cmd {
	return func() tea.Msg {
		var result AnimeFinderResult
		var err error

		switch m.state.find.source {
		case Kitsu:
			result, err = m.animeFinderMap[Kitsu].Search(query)
			if err != nil {
				return FetchErrorMsg(err)
			}

		case Local:
			result, err = m.animeFinderMap[Local].Search(query)
			if err != nil {
				return FetchErrorMsg(err)
			}
		}

		if len(result.infoItems) == 0 {
			return FetchedNoResultsMsg{}
		}
		m.state.animeResults = result.infoItems
		return result
	}
}

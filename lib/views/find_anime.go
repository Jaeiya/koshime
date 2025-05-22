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

type Find_AnimeView int

const (
	Find_QueryEntry = Find_AnimeView(iota)
	Find_Results
	Find_SelectedAnime
)

type Find_AnimeHelp map[Find_AnimeView]HelpInfo[Find_AnimeModel]

type Find_AnimeModel struct {
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
		tab           key.Binding
		openSynopsis  key.Binding
		closeSynopsis key.Binding
	}
	db             *database.Database
	helpMap        Find_AnimeHelp
	animeFinderMap map[AnimeSource]AnimeFinder
	sourceStrMap   map[AnimeSource]string
	state          Find_AnimeState
}

type Find_AnimeState struct {
	view          Find_AnimeView
	fetchErr      FetchErrorMsg
	results       []ui.AnimeInfo
	selectedAnime ui.AnimeInfo
	source        AnimeSource
	showSynopsis  bool
}

func newFindAnimeModel(db *database.Database) Find_AnimeModel {
	m := Find_AnimeModel{db: db}

	m.ui.loader = ui.NewLoader()
	m.ui.list = ui.NewList(ui.ListOptions{})

	m.ui.input = ui.NewTextInput()
	m.ui.input.Focus()
	m.ui.input.Placeholder = "Enter your query"
	m.ui.input.SetWidth(20)

	m.config.minInputLen = 4  // Minimum characters to submit search
	m.config.itemsPerPage = 5 // Max list items to display per page
	m.config.maxResults = 10  // Max results to find per search

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

	m.keys.tab = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "source"))
	m.keys.openSynopsis = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "open synopsis"))
	m.keys.closeSynopsis = key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "close synopsis"),
	)

	m.helpMap = Find_AnimeHelp{
		Find_QueryEntry: {
			ShortHelp: func(Find_AnimeModel) []key.Binding {
				return []key.Binding{keyMap.Submit, keyMap.Abort}
			},
		},
		Find_Results: {
			ShortHelp: func(m Find_AnimeModel) []key.Binding {
				if !m.ui.loader.IsLoading() && len(m.state.results) == 0 {
					return []key.Binding{keyMap.EscBack}
				}
				return []key.Binding{}
			},
		},
		Find_SelectedAnime: {
			ShortHelp: func(m Find_AnimeModel) []key.Binding {
				synKey := m.keys.openSynopsis
				if m.state.showSynopsis {
					synKey = m.keys.closeSynopsis
				}
				return []key.Binding{synKey, keyMap.EscBack}
			},
		},
	}
	return m
}

func (m Find_AnimeModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize.width = msg.Width
		m.windowSize.height = msg.Height

	case FetchErrorMsg:
		m.ui.loader.Stop()
		m.state.view = Find_Results
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

func (m Find_AnimeModel) View() (string, *tea.Cursor) {
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

func (m Find_AnimeModel) ShortHelp() []key.Binding {
	if m.ui.loader.IsLoading() {
		return []key.Binding{}
	}

	if v, exists := m.helpMap[m.state.view]; exists {
		return v.ShortHelp(m)
	}

	return []key.Binding{}
}

func (m Find_AnimeModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m Find_AnimeModel) UpdateQueryEntry(msg tea.Msg) (Find_AnimeModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			m.reset()
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
			m.state.source = (m.state.source + 1) % 2
		}
	}

	m.ui.input, cmd = m.ui.input.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Find_AnimeModel) ViewQueryEntry() (string, *tea.Cursor) {
	c := m.ui.input.Cursor()
	c.Shape = tea.CursorBar

	search := ui.TextStyle.Foreground(ansi.BrightBlack).
		Render("Source: " + m.sourceStrMap[m.state.source])

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

func (m Find_AnimeModel) UpdateResults(msg tea.Msg) (Find_AnimeModel, tea.Cmd) {
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
			m.reset()

		// Go back to query-entry-view from results-view
		case key.Matches(msg, keyMap.Back):
			if m.ui.list.FilterState() != list.Filtering {
				m.reset()
			}

		// Select Anime
		case key.Matches(msg, keyMap.Submit):
			// List needs 'Enter' control for applying filter
			if m.ui.list.FilterState() == list.Filtering {
				break
			}
			if len(m.state.results) > 0 {
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.selectedAnime = m.state.results[item.Index()]
				m.state.view = Find_SelectedAnime
			}

		}

	case AnimeFinderResult:
		m.ui.loader.Stop()
		m.state.view = Find_Results
		m.state.results = msg.infoItems

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

func (m Find_AnimeModel) ViewResults() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.fetchErr != nil {
		return m.state.fetchErr.Error(), nil
	}

	if len(m.ui.list.Items()) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			findAnimeMsgs.viewHeader("Results"),
			findAnimeMsgs.notFound(
				m.ui.input.Value(),
				m.state.source.String(),
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

func (m Find_AnimeModel) UpdateAnime(msg tea.Msg) (Find_AnimeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.EscBack, keyMap.Back):
			m.state.view = Find_Results

		case key.Matches(msg, m.keys.openSynopsis):
			m.state.showSynopsis = !m.state.showSynopsis
		}
	}
	return m, nil
}

func (m Find_AnimeModel) ViewAnime() string {
	if m.state.results != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			findAnimeMsgs.viewHeader("Entry Info"),
			"",
			ui.DisplayAnimeInfo(m.state.selectedAnime, m.state.showSynopsis),
		)
	}

	return "missing local or kitsu results to display"
}

func (m *Find_AnimeModel) reset() {
	// Do not reset source
	source := m.state.source
	m.state = Find_AnimeState{}
	m.state.source = source
	m.ui.input.Reset()
}

func (m *Find_AnimeModel) findAnime(query string) tea.Cmd {
	return func() tea.Msg {
		var result AnimeFinderResult
		var err error

		switch m.state.source {
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

		m.state.results = result.infoItems
		return result
	}
}

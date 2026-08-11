package views

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/charmbracelet/x/ansi"
)

type AnimeSearchView int

const (
	AnimeSearch_Query = AnimeSearchView(iota)
	AnimeSearch_Results
	AnimeSearch_Selected
)

type (
	SelectedAnimeMsg struct {
		Value kitsu.Anime
	}
	AnimeSearchExitMsg struct{}
	AnimeSearchOption  func(*AnimeSearchConfig)
	AnimeSearchHelp    map[AnimeSearchView]ui.KeyHelpInfo[AnimeSearchModel]
)

type AnimeSearchConfig struct {
	header            string
	consentHeader     string
	inputWidth        int
	minInputLen       int
	itemsPerPage      int
	maxResults        int
	source            app.AnimeFinderSource
	kitsuStatus       []kitsu.AnimeStatus
	useAnimeSelection bool
	escSendsExit      bool
}

func WithHeader(h string) AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		fac.header = h
	}
}

func WithExit() AnimeSearchOption {
	return func(asc *AnimeSearchConfig) {
		asc.escSendsExit = true
	}
}

// WithAnimeSelection enables a 3rd view that allows
// a user to review the anime selection, toggle the
// synopsis, and consent to submitting the anime
// for selection.
func WithAnimeSelection(consent string) AnimeSearchOption {
	return func(asc *AnimeSearchConfig) {
		asc.useAnimeSelection = true
		asc.consentHeader = consent
	}
}

func WithInputWidth(w int) AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		fac.inputWidth = w
	}
}

func WithMinInputLen(len int) AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		fac.minInputLen = len
	}
}

func WithItemsPerPage(itemsPerPage int) AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		fac.itemsPerPage = itemsPerPage
	}
}

func WithMaxResults(max int) AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		fac.maxResults = max
	}
}

func WithKitsuSource(s []kitsu.AnimeStatus) AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		if fac.source != app.NoSource {
			panic("cannot set to [kitsu] source; already using another source")
		}
		fac.kitsuStatus = s
		fac.source = app.Kitsu
	}
}

func WithLocalSource() AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		if fac.source != app.NoSource {
			panic("cannot set to [local] source; already using another source")
		}
		fac.source = app.Local
	}
}

type AnimeSearchModel struct {
	windowSize struct {
		width  int
		height int
	}
	config struct {
		source            app.AnimeFinderSource
		header            string
		consentHeader     string
		inputWidth        int
		minInputLen       int
		itemsPerPage      int
		maxResults        int
		useAnimeSelection bool
		escSendsExit      bool
	}
	ui struct {
		list         list.Model
		input        textinput.Model
		loader       ui.LoaderModel
		animeDisplay *AnimeDisplayModel
		consent      ui.ConsentModel
	}
	keys struct {
		tab key.Binding
	}
	helpMap        AnimeSearchHelp
	db             *database.Database
	state          AnimeSearchState
	animeFinderMap map[app.AnimeFinderSource]app.AnimeFinder
}

type AnimeSearchState struct {
	fetchErr      error
	view          AnimeSearchView
	source        app.AnimeFinderSource
	results       []kitsu.Anime
	selectedAnime kitsu.Anime
}

func NewAnimeSearchModel(db *database.Database, opts ...AnimeSearchOption) *AnimeSearchModel {
	cfg := &AnimeSearchConfig{}
	for _, o := range opts {
		o(cfg)
	}

	m := &AnimeSearchModel{db: db}

	m.ui.loader = ui.NewLoader()
	m.ui.list = ui.NewList(ui.ListOptions{})
	m.ui.animeDisplay = NewAnimeDisplayModel()

	m.ui.input = ui.NewTextInput()
	m.ui.input.Focus()
	m.ui.input.Placeholder = "Enter your query"
	m.ui.input.SetWidth(20)

	m.config.useAnimeSelection = cfg.useAnimeSelection
	m.config.escSendsExit = cfg.escSendsExit

	if cfg.inputWidth > 0 {
		m.ui.input.SetWidth(cfg.inputWidth)
	}

	m.config.consentHeader = cfg.consentHeader

	m.config.header = "Find Anime"
	if cfg.header != "" {
		m.config.header = cfg.header
	}

	m.config.minInputLen = 4 // Minimum characters to submit search
	if cfg.minInputLen > 0 {
		m.config.minInputLen = cfg.minInputLen
	}

	m.config.itemsPerPage = 5 // Max list items to display per page
	if cfg.itemsPerPage > 0 {
		m.config.itemsPerPage = cfg.itemsPerPage
	}

	m.config.maxResults = 10 // Max results to find per search
	if cfg.maxResults > 0 {
		m.config.maxResults = cfg.maxResults
	}

	m.config.source = cfg.source
	if cfg.source != app.NoSource {
		m.state.source = cfg.source
	} else {
		m.state.source = app.Kitsu
	}

	if cfg.kitsuStatus == nil {
		cfg.kitsuStatus = []kitsu.AnimeStatus{kitsu.AnimeNew, kitsu.AnimeFinished}
	}

	m.animeFinderMap = map[app.AnimeFinderSource]app.AnimeFinder{
		app.Kitsu: app.NewKitsuAnimeFinder(m.config.maxResults, cfg.kitsuStatus),
		app.Local: app.NewLocalAnimeFinder(m.config.maxResults, db),
	}

	m.keys.tab = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "source"))

	m.helpMap = AnimeSearchHelp{
		AnimeSearch_Query: {
			ShortHelp: func(fam AnimeSearchModel) []key.Binding {
				keys := make([]key.Binding, 0, 4)
				if m.config.source == app.NoSource {
					keys = append(keys, fam.keys.tab, ui.KeyMap.Submit)
				}
				if m.config.source != app.NoSource {
					keys = append(keys, ui.KeyMap.Submit)
				}
				if m.config.escSendsExit {
					keys = append(keys, ui.KeyMap.MainMenu)
				}
				return keys
			},
		},
		AnimeSearch_Results: {
			ShortHelp: func(fam AnimeSearchModel) []key.Binding {
				if !m.ui.loader.IsLoading() && len(m.state.results) == 0 {
					return []key.Binding{ui.KeyMap.EscBack}
				}
				return []key.Binding{}
			},
		},
		AnimeSearch_Selected: {
			ShortHelp: func(fam AnimeSearchModel) []key.Binding {
				if !fam.config.useAnimeSelection {
					return []key.Binding{}
				}
				if fam.config.consentHeader == "" {
					return []key.Binding{
						m.ui.animeDisplay.ShortHelp()[0],
						ui.KeyMap.EscBack,
					}
				}
				return []key.Binding{
					m.ui.animeDisplay.ShortHelp()[0],
					ui.KeyMap.Submit,
					ui.KeyMap.EscBack,
				}
			},
		},
	}

	return m
}

func (m *AnimeSearchModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize.width = msg.Width
		m.windowSize.height = msg.Height
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Abort):
			if m.state.view == AnimeSearch_Query && m.config.escSendsExit {
				return func() tea.Msg { return AnimeSearchExitMsg{} }
			}
		}
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case AnimeSearch_Query:
		cmd = m.UpdateQuery(msg)
		cmds = append(cmds, cmd)
	case AnimeSearch_Results:
		cmd = m.UpdateResults(msg)
		cmds = append(cmds, cmd)
	case AnimeSearch_Selected:
		cmd = m.UpdateSelection(msg)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (m AnimeSearchModel) View() tea.View {
	switch m.state.view {
	case AnimeSearch_Query:
		return m.ViewQuery()
	case AnimeSearch_Results:
		return m.ViewResults()
	case AnimeSearch_Selected:
		return m.ViewSelection()
	}
	return tea.NewView("missing AnimeSearch view")
}

func (m AnimeSearchModel) ShortHelp() []key.Binding {
	return m.helpMap[m.state.view].ShortHelp(m)
}

func (m AnimeSearchModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m *AnimeSearchModel) UpdateQuery(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Submit):
			hasShortInput := utils.RuneCount(m.ui.input.Value()) < m.config.minInputLen

			if m.ui.loader.IsLoading() || hasShortInput {
				break
			}

			m.ui.loader, cmd = m.ui.loader.Start(m.config.header)
			m.state.view = AnimeSearch_Results
			return tea.Batch(cmd, m.findAnime(m.ui.input.Value()))

		case key.Matches(msg, m.keys.tab):
			// Only show tab when defaulting to multi-source search
			if m.config.source != app.NoSource {
				break
			}
			m.state.source = (m.state.source + 1) % 3
			// Ignore 'NoSource' state
			if m.state.source == 0 {
				m.state.source += 1
			}
		}
	}

	m.ui.input, cmd = m.ui.input.Update(msg)
	cmds = append(cmds, cmd)
	return tea.Batch(cmds...)
}

func (m AnimeSearchModel) ViewQuery() tea.View {
	c := m.ui.input.Cursor()
	c.Shape = tea.CursorBar

	desc := ui.Style.MarginTop(1).Render(ui.DisplayText([]string{
		`Lookup an anime by any ;b;word ;x;or ;b;phrase;x;. Try to use
words that might be in the ;dc;title ;x;or ;dc;description;x;, for
better results.`,
	}, 0))

	if m.config.source == app.NoSource {
		desc = ui.DisplayText(
			[]string{
				`;x;You can search for a ;b;full title;x;, ;b;phrase;x;, or just a ;b;single
word;x;. You can even search for ;b;part ;x;of a word. Your query will be applied to all
available titles, as well as the synopsis.`,
				`The ;dgu;Kitsu;x; source searches ;b;all ;x;of Kitsu (not just your Kitsu
app.ary) for any matches.`,
				`The ;dgu;Local;x; source searches your ;b;Koshime ;x;database for any matches.
It only contains anime that you're currently watching.`,
			},
			1, 1,
		)
	}

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle(m.config.header),
		desc,
	)

	footer := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.Style.MarginTop(1).Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.ui.input.View(),
			ui.DisplayCharLimit(m.config.minInputLen, m.ui.input.Value()),
		)),
	)

	view := tea.NewView(header)
	if m.config.source == app.NoSource {
		sourceName, sourceEmoji := m.state.source.Name()
		search := ui.TextStyle.Foreground(ansi.BrightBlack).
			Render(utils.ColorText(fmt.Sprintf(";bk;Source: ;dgu;%s;x;%s", sourceName, sourceEmoji)))

		view.Content = lipgloss.JoinVertical(
			lipgloss.Left,
			view.Content,
			search,
		)
	}

	view.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		view.Content,
		footer,
	)

	view.Cursor = c
	view.Cursor.Y += lipgloss.Height(view.Content) - 1
	return view
}

func (m *AnimeSearchModel) UpdateResults(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		// Go back to query-entry-view from results-view
		case key.Matches(msg, ui.KeyMap.MainMenu):
			// List needs 'Esc' control to cancel filter
			if m.ui.list.FilterState() > list.Unfiltered {
				break
			}
			m.Reset()

		// Go back to query-entry-view from results-view
		case key.Matches(msg, ui.KeyMap.Back):
			if m.ui.list.FilterState() != list.Filtering {
				m.Reset()
			}

		// Select Anime
		case key.Matches(msg, ui.KeyMap.Submit):
			// List needs 'Enter' control for applying filter
			if m.ui.list.FilterState() == list.Filtering {
				break
			}
			if len(m.state.results) > 0 {
				//nolint:errcheck // it will ALWAYS be a list item
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.selectedAnime = m.state.results[item.Index()]
				if m.config.useAnimeSelection {
					m.state.view = AnimeSearch_Selected
				} else {
					return func() tea.Msg { return SelectedAnimeMsg{m.state.selectedAnime} }
				}
			}

		}

	case app.AnimeFinderResult:
		m.ui.loader.Stop()
		m.state.results = msg.InfoItems

		m.ui.list = ui.NewList(
			ui.ListOptions{
				Items:         msg.ListItems,
				ShortHelpKeys: []key.Binding{ui.KeyMap.Select, ui.KeyMap.Back},
				Width:         m.windowSize.width,
				MaxHeight:     int(float64(m.windowSize.height) * 0.66),
				ItemsPerPage:  m.config.itemsPerPage,
				EnableFilter:  true,
			},
		)
	}

	m.ui.list, cmd = m.ui.list.Update(msg)
	cmds = append(cmds, cmd)
	return tea.Batch(cmds...)
}

func (m AnimeSearchModel) ViewResults() tea.View {
	if m.ui.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.ui.loader.View()))
	}

	if m.state.fetchErr != nil {
		return tea.NewView(ui.DisplayError(m.state.fetchErr))
	}

	if len(m.ui.list.Items()) == 0 {
		sourceName, _ := m.state.source.Name()
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle(m.config.header, "Results"),
			ui.DisplayText([]string{
				fmt.Sprintf(
					"No ;dgu;%s;x; results found for: ;y;%s",
					sourceName,
					m.ui.input.Value(),
				),
			}, 0, 1),
		))
	}

	h := ui.DisplaySubTitle(m.config.header, "Results")
	view := tea.NewView("")

	if m.ui.list.FilterState() == list.Filtering {
		// The filter has no margin, so we enforce
		h = ui.Style.MarginBottom(1).Render(h)
		view.Cursor = m.ui.list.FilterInput.Cursor()
		view.Cursor.Shape = tea.CursorBlock
		view.Cursor.Color = ansi.Yellow
		view.Cursor.Y += lipgloss.Height(h)
		view.Cursor.X += 2 // Adjust for custom margin
	}

	view.Content = lipgloss.JoinVertical(lipgloss.Left, h, m.ui.list.View())
	return view
}

func (m *AnimeSearchModel) UpdateSelection(msg tea.Msg) tea.Cmd {
	if m.config.consentHeader != "" {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Submit):
			if m.config.consentHeader == "" {
				break
			}
			if m.ui.consent.Select() == ui.No {
				m.state.view = AnimeSearch_Results
				return nil
			}
			return func() tea.Msg { return SelectedAnimeMsg{m.state.selectedAnime} }

		case key.Matches(msg, ui.KeyMap.EscBack, ui.KeyMap.Back):
			m.state.view = AnimeSearch_Results
			m.ui.consent.Reset()
		}
	}

	m.ui.animeDisplay.Update(msg)
	return nil
}

func (m AnimeSearchModel) ViewSelection() tea.View {
	if m.state.results != nil {
		consentStyle := ui.TextStyle.Foreground(ansi.BrightBlue)

		view := tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle(m.config.header, "Entry Info"),
			"",
			m.ui.animeDisplay.View(m.state.selectedAnime),
		))

		if m.config.consentHeader != "" {
			view.Content = lipgloss.JoinVertical(
				lipgloss.Left,
				view.Content,
				"",
				m.ui.consent.View(consentStyle.Render(m.config.consentHeader)),
			)
			return view
		}

		return view
	}

	sourceName, _ := m.state.source.Name()
	return tea.NewView(fmt.Sprintf("missing [%s] results to display", sourceName))
}

func (m *AnimeSearchModel) Reset() {
	source := m.state.source
	m.state = AnimeSearchState{}
	m.state.source = source
	m.ui.input.Reset()
	m.ui.consent.Reset()
}

func (m *AnimeSearchModel) findAnime(query string) tea.Cmd {
	return func() tea.Msg {
		var result app.AnimeFinderResult
		var err error

		switch m.state.source {
		case app.Kitsu:
			result, err = m.animeFinderMap[app.Kitsu].Search(query)
			if err != nil {
				return err
			}

		case app.Local:
			result, err = m.animeFinderMap[app.Local].Search(query)
			if err != nil {
				return err
			}
		default:
		}

		m.state.results = result.InfoItems
		return result
	}
}

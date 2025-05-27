package ui

import (
	"fmt"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type AnimeSearchView int

const (
	AnimeSearch_Query = AnimeSearchView(iota)
	AnimeSearch_Results
	AnimeSearch_Selected
)

type (
	SelectedAnimeMsg   = AnimeInfo
	AnimeSearchExitMsg struct{}
	AnimeSearchOption  func(*AnimeSearchConfig)
	AnimeSearchHelp    map[AnimeSearchView]KeyHelpInfo[AnimeSearchModel]
)

type AnimeSearchConfig struct {
	header            string
	consentHeader     string
	inputWidth        int
	minInputLen       int
	itemsPerPage      int
	maxResults        int
	source            AnimeFinderSource
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

func WithItemsPerPage(ipp int) AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		fac.itemsPerPage = ipp
	}
}

func WithMaxResults(max int) AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		fac.maxResults = max
	}
}

func WithKitsuSource(s []kitsu.AnimeStatus) AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		if fac.source != NoSource {
			panic("cannot set to [kitsu] source; already using another source")
		}
		fac.kitsuStatus = s
		fac.source = Kitsu
	}
}

func WithLocalSource() AnimeSearchOption {
	return func(fac *AnimeSearchConfig) {
		if fac.source != NoSource {
			panic("cannot set to [local] source; already using another source")
		}
		fac.source = Local
	}
}

type AnimeSearchModel struct {
	windowSize struct {
		width  int
		height int
	}
	config struct {
		source            AnimeFinderSource
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
		loader       LoaderModel
		animeDisplay *AnimeDisplayModel
		consent      ConsentModel
	}
	keys struct {
		tab key.Binding
	}
	helpMap        AnimeSearchHelp
	db             *database.Database
	state          AnimeSearchState
	animeFinderMap map[AnimeFinderSource]AnimeFinder
}

type AnimeSearchState struct {
	fetchErr      error
	view          AnimeSearchView
	source        AnimeFinderSource
	results       []AnimeInfo
	selectedAnime AnimeInfo
}

func NewAnimeSearchModel(db *database.Database, opts ...AnimeSearchOption) *AnimeSearchModel {
	cfg := &AnimeSearchConfig{}
	for _, o := range opts {
		o(cfg)
	}

	m := &AnimeSearchModel{db: db}

	m.ui.loader = NewLoader()
	m.ui.list = NewList(ListOptions{})
	m.ui.animeDisplay = NewAnimeDisplayModel()

	m.ui.input = NewTextInput()
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
	if cfg.source != NoSource {
		m.state.source = cfg.source
	} else {
		m.state.source = Kitsu
	}

	if cfg.kitsuStatus == nil {
		cfg.kitsuStatus = []kitsu.AnimeStatus{kitsu.AnimeNew, kitsu.AnimeFinished}
	}

	m.animeFinderMap = map[AnimeFinderSource]AnimeFinder{
		Kitsu: NewKitsuAnimeFinder(m.config.maxResults, cfg.kitsuStatus),
		Local: NewLocalAnimeFinder(m.config.maxResults, db),
	}

	m.keys.tab = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "source"))

	m.helpMap = AnimeSearchHelp{
		AnimeSearch_Query: {
			ShortHelp: func(fam AnimeSearchModel) []key.Binding {
				keys := make([]key.Binding, 0, 4)
				if m.config.source == NoSource {
					keys = append(keys, fam.keys.tab, KeyMap.Submit)
				}
				if m.config.escSendsExit {
					keys = append(keys, KeyMap.MainMenu)
				}
				if m.config.source != NoSource {
					keys = append(keys, KeyMap.Submit)
				}
				return keys
			},
		},
		AnimeSearch_Results: {
			ShortHelp: func(fam AnimeSearchModel) []key.Binding {
				if !m.ui.loader.IsLoading() && len(m.state.results) == 0 {
					return []key.Binding{KeyMap.EscBack}
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
						NewAnimeDisplayModel().ShortHelp()[0],
						KeyMap.EscBack,
					}
				}
				return []key.Binding{
					m.ui.animeDisplay.ShortHelp()[0],
					KeyMap.Submit,
					KeyMap.EscBack,
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
		case key.Matches(msg, KeyMap.Abort):
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

func (m AnimeSearchModel) View() (string, *tea.Cursor) {
	switch m.state.view {
	case AnimeSearch_Query:
		return m.ViewQuery()
	case AnimeSearch_Results:
		return m.ViewResults()
	case AnimeSearch_Selected:
		return m.ViewSelection()
	}
	return "", nil
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
		case key.Matches(msg, KeyMap.Submit):
			hasShortInput := utils.RuneCount(m.ui.input.Value()) < m.config.minInputLen

			if m.ui.loader.IsLoading() || hasShortInput {
				break
			}

			m.ui.loader, cmd = m.ui.loader.Start(m.config.header)
			m.state.view = AnimeSearch_Results
			return tea.Batch(cmd, m.findAnime(m.ui.input.Value()))

		case key.Matches(msg, m.keys.tab):
			// Only show tab when defaulting to multi-source search
			if m.config.source != NoSource {
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

func (m AnimeSearchModel) ViewQuery() (string, *tea.Cursor) {
	c := m.ui.input.Cursor()
	c.Shape = tea.CursorBar

	search := ""
	if m.config.source == NoSource {
		search = TextStyle.Foreground(ansi.BrightBlack).
			Render(utils.ColorText(fmt.Sprintf(";bk;Source: ;dgu;%s", m.state.source)))
	}

	desc := Style.MarginTop(1).Render(DisplayText([]string{
		`Lookup an anime by any ;b;word ;x;or ;b;phrase;x;. Try to use
words that might be in the ;dc;title ;x;or ;dc;description;x;, for
better results.`,
	}, 0))

	if m.config.source == NoSource {
		desc = DisplayText(
			[]string{
				`;x;You can search for a ;b;full title;x;, ;b;phrase;x;, or just a ;b;single
word;x;. You can even search for ;b;part ;x;of a word. Your query will be applied to all
available titles, as well as the synopsis.`,
				`The ;dgu;Kitsu;x; source searches ;b;all ;x;of Kitsu (not just your Kitsu
library) for any matches.`,
				`The ;dgu;Local;x; source searches your ;b;Koshime ;x;database for any matches.
It only contains anime that you're currently watching.`,
			},
			0, 1, 1,
		)
	}

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		DisplayTitle(m.config.header),
		desc,
	)

	footer := lipgloss.JoinVertical(
		lipgloss.Left,
		Style.MarginTop(1).Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.ui.input.View(),
			DisplayCharLimit(m.config.minInputLen, m.ui.input.Value()),
		)),
	)

	view := header
	if m.config.source == NoSource {
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			search,
		)
	}

	view = lipgloss.JoinVertical(
		lipgloss.Left,
		view,
		footer,
	)
	c.Y += lipgloss.Height(view) - 1
	return view, c
}

func (m *AnimeSearchModel) UpdateResults(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		// Go back to query-entry-view from results-view
		case key.Matches(msg, KeyMap.MainMenu):
			// List needs 'Esc' control to cancel filter
			if m.ui.list.FilterState() > list.Unfiltered {
				break
			}
			m.Reset()

		// Go back to query-entry-view from results-view
		case key.Matches(msg, KeyMap.Back):
			if m.ui.list.FilterState() != list.Filtering {
				m.Reset()
			}

		// Select Anime
		case key.Matches(msg, KeyMap.Submit):
			// List needs 'Enter' control for applying filter
			if m.ui.list.FilterState() == list.Filtering {
				break
			}
			if len(m.state.results) > 0 {
				item := m.ui.list.SelectedItem().(ListItem)
				m.state.selectedAnime = m.state.results[item.Index()]
				if m.config.useAnimeSelection {
					m.state.view = AnimeSearch_Selected
				} else {
					return func() tea.Msg { return SelectedAnimeMsg(m.state.selectedAnime) }
				}
			}

		}

	case AnimeFinderResult:
		m.ui.loader.Stop()
		m.state.results = msg.InfoItems

		m.ui.list = NewList(
			ListOptions{
				Items:         msg.ListItems,
				ShortHelpKeys: []key.Binding{KeyMap.Back},
				Width:         m.windowSize.width,
				MaxHeight:     int(float64(m.windowSize.height) * 0.66),
				ItemsPerPage:  m.config.itemsPerPage,
			},
		)
	}

	m.ui.list, cmd = m.ui.list.Update(msg)
	cmds = append(cmds, cmd)
	return tea.Batch(cmds...)
}

func (m AnimeSearchModel) ViewResults() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.fetchErr != nil {
		return DisplayError(m.state.fetchErr), nil
	}

	if len(m.ui.list.Items()) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			DisplaySubTitle(m.config.header, "Results"),
			DisplayText([]string{
				fmt.Sprintf(
					";x;No ;dgu;%s;x; results found for: ;y;%s",
					m.state.source,
					m.ui.input.Value(),
				),
			}, 0, 1),
		), nil
	}

	h := DisplaySubTitle(m.config.header, "Results")
	var c *tea.Cursor
	// The filter has no margin, so we enforce
	if m.ui.list.FilterState() == list.Filtering {
		h = Style.MarginBottom(1).Render(h)
		c = m.ui.list.FilterInput.Cursor()
		c.Shape = tea.CursorBlock
		c.Color = ansi.Yellow
		c.Y += lipgloss.Height(h)
		c.X += 2 // Adjust for custom margin
	}
	return lipgloss.JoinVertical(lipgloss.Left, h, m.ui.list.View()), c
}

func (m *AnimeSearchModel) UpdateSelection(msg tea.Msg) tea.Cmd {
	if m.config.consentHeader != "" {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, KeyMap.Submit):
			if m.config.consentHeader == "" {
				break
			}
			if m.ui.consent.Select() == No {
				m.state.view = AnimeSearch_Results
				return nil
			}
			return func() tea.Msg { return SelectedAnimeMsg(m.state.selectedAnime) }

		case key.Matches(msg, KeyMap.EscBack, KeyMap.Back):
			m.state.view = AnimeSearch_Results
		}
	}

	m.ui.animeDisplay.Update(msg)
	return nil
}

func (m AnimeSearchModel) ViewSelection() (string, *tea.Cursor) {
	if m.state.results != nil {
		consentStyle := TextStyle.Foreground(ansi.BrightBlue)

		body := lipgloss.JoinVertical(
			lipgloss.Left,
			DisplaySubTitle(m.config.header, "Entry Info"),
			"",
			m.ui.animeDisplay.View(m.state.selectedAnime),
		)

		if m.config.consentHeader != "" {
			return lipgloss.JoinVertical(
				lipgloss.Left,
				body,
				"",
				m.ui.consent.View(consentStyle.Render(m.config.consentHeader)),
			), nil
		}

		return body, nil
	}

	return fmt.Sprintf("missing [%s] results to display", m.state.source), nil
}

func (m *AnimeSearchModel) Reset() {
	source := m.state.source
	m.state = AnimeSearchState{}
	m.state.source = source
	m.ui.input.Reset()
}

func (m *AnimeSearchModel) findAnime(query string) tea.Cmd {
	return func() tea.Msg {
		var result AnimeFinderResult
		var err error

		switch m.state.source {
		case Kitsu:
			result, err = m.animeFinderMap[Kitsu].Search(query)
			if err != nil {
				return err
			}

		case Local:
			result, err = m.animeFinderMap[Local].Search(query)
			if err != nil {
				return err
			}
		}

		m.state.results = result.InfoItems
		return result
	}
}

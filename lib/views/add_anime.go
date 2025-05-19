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
	AddAnimeView int
)

const (
	Add_AnimeQuery = AddAnimeView(iota)
	Add_AnimeResults
	Add_AnimeReview
	Add_Rss
	Add_RssResults
	Add_RssReview
)

type HelpInfo struct {
	ShortHelp []key.Binding
	FullHelp  [][]key.Binding
}

var newAnimeHelpMap = map[AddAnimeView]HelpInfo{
	Add_AnimeQuery: {
		ShortHelp: []key.Binding{keyMap.Submit, keyMap.Abort},
	},
	Add_AnimeReview: {
		ShortHelp: []key.Binding{keyMap.EscBack},
	},
}

type addAnimeModel struct {
	windowSize struct {
		width  int
		height int
	}
	config struct {
		maxInputWidth int
		minInputLen   int // Min required chars to submit input
		itemsPerPage  int // How many list items per page
	}
	ui struct {
		loader ui.LoaderModel
		input  textinput.Model
		list   list.Model
	}
	db    *database.Database
	state addAnimeModelState
}

type addAnimeModelState struct {
	view          AddAnimeView
	fetchErr      error
	results       []ui.AnimeInfo
	selectedAnime ui.AnimeInfo
}

func newAddAnimeModel(db *database.Database) addAnimeModel {
	m := addAnimeModel{db: db}
	m.config.minInputLen = 4
	m.config.maxInputWidth = 30
	m.config.itemsPerPage = 5

	m.ui.input = ui.NewTextInput()
	m.ui.input.SetWidth(m.config.maxInputWidth)
	m.ui.input.Placeholder = "Enter query"
	m.ui.loader = ui.NewLoader()

	return m
}

func (m addAnimeModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
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
	case Add_AnimeQuery:
		m, cmd = m.UpdateAnimeQuery(msg)
		cmds = append(cmds, cmd)
	case Add_AnimeResults:
		m, cmd = m.UpdateAnimeResults(msg)
		cmds = append(cmds, cmd)
	case Add_AnimeReview:
		m, cmd = m.UpdateAnimeReview(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m addAnimeModel) View() (string, *tea.Cursor) {
	switch m.state.view {
	case Add_AnimeQuery:
		return m.ViewQueryAnime()
	case Add_AnimeResults:
		return m.ViewAnimeResults()
	case Add_AnimeReview:
		return m.ViewAnimeReview()
	}
	return "AddAnime::missing view", nil
}

func (m addAnimeModel) ShortHelp() []key.Binding {
	if m.state.fetchErr != nil {
		return []key.Binding{keyMap.EscBack}
	}
	if bindings, exists := newAnimeHelpMap[m.state.view]; exists {
		return bindings.ShortHelp
	}
	return []key.Binding{}
}

func (m addAnimeModel) FullHelp() [][]key.Binding {
	if bindings, exists := newAnimeHelpMap[m.state.view]; exists {
		return bindings.FullHelp
	}
	return [][]key.Binding{}
}

func (m addAnimeModel) UpdateAnimeQuery(msg tea.Msg) (addAnimeModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			return m, abort

		case key.Matches(msg, keyMap.Submit):
			// Do not submit query below min input length
			if utils.RuneCount(m.ui.input.Value()) < m.config.minInputLen {
				break
			}
			m.ui.loader, cmd = m.ui.loader.Start("Finding Anime")
			m.state.view = Add_AnimeResults
			return m, tea.Batch(cmd, m.findAnime(m.ui.input.Value()))
		}
	}

	m.ui.input, cmd = m.ui.input.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m addAnimeModel) ViewQueryAnime() (string, *tea.Cursor) {
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

func (m addAnimeModel) UpdateAnimeResults(msg tea.Msg) (addAnimeModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			m.reset()

		case key.Matches(msg, keyMap.Select):
			item := m.ui.list.SelectedItem().(ui.ListItem)
			m.state.view = Add_AnimeReview
			return m, func() tea.Msg {
				return m.state.results[item.Index()]
			}
		}

	case FetchErrorMsg:
		m.state.fetchErr = msg

	case AnimeFinderResult:
		m.ui.loader.Stop()
		m.state.results = msg.infoItems
		m.ui.list = ui.NewList(
			ui.ListOptions{
				Items: msg.listItems,
				// ShortHelpKeys: []key.Binding{m.keys.backspace},
				Width:        m.windowSize.width,
				MaxHeight:    int(float64(m.windowSize.height) * 0.66),
				ItemsPerPage: m.config.itemsPerPage,
			},
		)
	}

	if len(m.ui.list.Items()) > 0 {
		m.ui.list, cmd = m.ui.list.Update(msg)
	}
	return m, cmd
}

func (m addAnimeModel) ViewAnimeResults() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.fetchErr != nil {
		return ui.Style.MarginTop(1).Render(m.state.fetchErr.Error()), nil
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

func (m addAnimeModel) UpdateAnimeReview(msg tea.Msg) (addAnimeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.EscBack):
			m.state.view = Add_AnimeResults
		}
	case ui.AnimeInfo:
		m.state.selectedAnime = msg
	}
	return m, nil
}

func (m addAnimeModel) ViewAnimeReview() (string, *tea.Cursor) {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		findAnimeMsgs.viewHeader("Entry Info"),
		"",
		ui.DisplayAnimeInfo(m.state.selectedAnime),
	), nil
}

func (m *addAnimeModel) reset() {
	m.state = addAnimeModelState{
		view: Add_AnimeQuery,
	}
	m.ui.input.Reset()
}

func (m addAnimeModel) findAnime(query string) tea.Cmd {
	return func() tea.Msg {
		af := NewKitsuAnimeFinder(10, []kitsu.AnimeStatus{kitsu.AnimeNew})
		anime, err := af.Search(query)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return anime
	}
}

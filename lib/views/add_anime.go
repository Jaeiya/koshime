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
	Add_AnimeQuery = Add_AnimeView(iota)
	Add_AnimeResults
	Add_AnimeReview
	Add_Rss
	Add_RssResults
	Add_RssReview
)

type Add_AnimeHelp map[Add_AnimeView]HelpInfo[Add_AnimeModel]

type Add_AnimeModel struct {
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
	helpMap Add_AnimeHelp
	db      *database.Database
	state   Add_AnimeModelState
}

type Add_AnimeModelState struct {
	view          Add_AnimeView
	fetchErr      error
	results       []ui.AnimeInfo
	selectedAnime ui.AnimeInfo
	showSynopsis  bool
}

func newAddAnimeModel(db *database.Database) Add_AnimeModel {
	m := Add_AnimeModel{db: db}
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

	m.helpMap = Add_AnimeHelp{
		Add_AnimeQuery: {
			ShortHelp: func(Add_AnimeModel) []key.Binding {
				return []key.Binding{keyMap.Submit, keyMap.Abort}
			},
		},
		Add_AnimeResults: {
			ShortHelp: func(m Add_AnimeModel) []key.Binding {
				if !m.ui.loader.IsLoading() && len(m.state.results) == 0 {
					return []key.Binding{keyMap.EscBack}
				}
				return []key.Binding{}
			},
		},
		Add_AnimeReview: {
			ShortHelp: func(m Add_AnimeModel) []key.Binding {
				synKey := m.keys.openSynopsis
				if m.state.showSynopsis {
					synKey = m.keys.closeSynopsis
				}
				return []key.Binding{synKey, keyMap.Up, keyMap.Down, keyMap.Select}
			},
		},
	}
	return m
}

func (m Add_AnimeModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
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

func (m Add_AnimeModel) View() (string, *tea.Cursor) {
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

func (m Add_AnimeModel) ShortHelp() []key.Binding {
	if m.state.fetchErr != nil {
		return []key.Binding{keyMap.EscBack}
	}
	if bindings, exists := m.helpMap[m.state.view]; exists {
		return bindings.ShortHelp(m)
	}
	return []key.Binding{}
}

func (m Add_AnimeModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m Add_AnimeModel) UpdateAnimeQuery(msg tea.Msg) (Add_AnimeModel, tea.Cmd) {
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

func (m Add_AnimeModel) ViewQueryAnime() (string, *tea.Cursor) {
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

func (m Add_AnimeModel) UpdateAnimeResults(msg tea.Msg) (Add_AnimeModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			// Delegate esc key to list, during filter operations
			if m.ui.list.FilterState() > list.Unfiltered {
				break
			}
			m.reset()

		case key.Matches(msg, keyMap.Back):
			m.reset()

		case key.Matches(msg, keyMap.Select):
			// Do not attempt to select an item while filtering
			if m.ui.list.FilterState() == list.Filtering {
				break
			}
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
				Items:         msg.listItems,
				ShortHelpKeys: []key.Binding{keyMap.Back},
				Width:         m.windowSize.width,
				MaxHeight:     int(float64(m.windowSize.height) * 0.66),
				ItemsPerPage:  m.config.itemsPerPage,
			},
		)
	}

	m.ui.list, cmd = m.ui.list.Update(msg)
	return m, cmd
}

func (m Add_AnimeModel) ViewAnimeResults() (string, *tea.Cursor) {
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

func (m Add_AnimeModel) UpdateAnimeReview(msg tea.Msg) (Add_AnimeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.EscBack):
			m.state.view = Add_AnimeResults

		case key.Matches(msg, m.keys.openSynopsis):
			m.state.showSynopsis = !m.state.showSynopsis

		case key.Matches(msg, keyMap.Submit):
			if m.ui.consent.Select() == ui.No {
				m.state.view = Add_AnimeResults
				return m, nil
			}
		}
	case ui.AnimeInfo:
		m.state.selectedAnime = msg
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m Add_AnimeModel) ViewAnimeReview() (string, *tea.Cursor) {
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

func (m *Add_AnimeModel) reset() {
	m.state = Add_AnimeModelState{
		view: Add_AnimeQuery,
	}
	m.ui.input.Reset()
}

func (m Add_AnimeModel) findAnime(query string) tea.Cmd {
	return func() tea.Msg {
		af := NewKitsuAnimeFinder(m.config.maxAnimeResults, []kitsu.AnimeStatus{kitsu.AnimeNew})
		anime, err := af.Search(query)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return anime
	}
}

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

type AddAnimeView int

const (
	Add_QueryAnime = AddAnimeView(iota)
	Add_Results
	Add_ReviewAnime
	Add_RSS
	Add_RSSResults
	Add_ReviewRSS
)

type AddAnimeModel struct {
	config struct {
		minInputLen  int
		itemsPerPage int
	}
	windowSize struct {
		width  int
		height int
	}
	input  textinput.Model
	list   list.Model
	loader ui.LoaderModel
	state  struct {
		view    AddAnimeView
		results kitsu.AnimeData
		find    struct {
			passed bool
			failed bool
		}
	}
}

func NewAddAnimeModel(db *database.Database) AddAnimeModel {
	m := AddAnimeModel{}

	m.list = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	// Prevent esc/q key from sending tea.Quit from inside list
	m.list.DisableQuitKeybindings()

	m.input = ui.NewTextInput()
	m.input.SetWidth(30)
	m.input.Placeholder = "Enter query"

	m.loader = ui.NewLoader()

	m.config.minInputLen = 4
	m.config.itemsPerPage = 5

	return m
}

func (m AddAnimeModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize.width = msg.Width
		m.windowSize.height = msg.Height
	}

	if m.loader.IsLoading() {
		m.loader, cmd = m.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case Add_QueryAnime:
		m, cmd = m.updateAnimeQuery(msg)
		cmds = append(cmds, cmd)

	case Add_Results:
		m, cmd = m.updateAnimeResults(msg)
		cmds = append(cmds, cmd)

	case Add_ReviewAnime:
		m, cmd = m.updateAnimeReview(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AddAnimeModel) View() (string, *tea.Cursor) {
	switch m.state.view {
	case Add_QueryAnime:
		return m.viewAnimeQuery()

	case Add_Results:
		return m.viewAnimeResults()

	case Add_ReviewAnime:
		return m.viewAnimeReview()
	}
	return "AddAnimeModel::missing view", nil
}

func (m AddAnimeModel) ShortHelp() []key.Binding {
	switch m.state.view {
	case Add_QueryAnime:
		return []key.Binding{
			keyMap.Submit, keyMap.MainMenu,
		}
	}

	return []key.Binding{}
}

func (m AddAnimeModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m AddAnimeModel) updateAnimeQuery(msg tea.Msg) (AddAnimeModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			return m, abort

		case key.Matches(msg, keyMap.Submit):
			// Do not submit query below min input length
			if utils.RuneCount(m.input.Value()) < m.config.minInputLen {
				break
			}
			m.loader, cmd = m.loader.Start("Finding Anime")
			m.state.view = Add_Results
			return m, tea.Batch(cmd, m.findAnime(m.input.Value()))
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m AddAnimeModel) viewAnimeQuery() (string, *tea.Cursor) {
	c := m.input.Cursor()
	c.Shape = tea.CursorBar

	header := ui.Style.MarginBottom(1).Render(addAnimeMsgs.header)
	body := lipgloss.JoinVertical(lipgloss.Left, header, addAnimeMsgs.queryDesc)

	c.Y += lipgloss.Height(body)
	inputView := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.input.View(),
		ui.DisplayCharLimit(m.config.minInputLen, m.input.Value()),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		body,
		inputView,
	), c
}

func (m AddAnimeModel) updateAnimeResults(msg tea.Msg) (AddAnimeModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			m.reset()
		}

	case FetchedKitsuEntriesMsg:
		m.loader.Stop()
		m.list = ui.NewList(
			ui.ListOptions{
				Items: msg.items,
				// ShortHelpKeys: []key.Binding{m.keys.backspace},
				Width:        m.windowSize.width,
				MaxHeight:    int(float64(m.windowSize.height) * 0.66),
				ItemsPerPage: m.config.itemsPerPage,
			},
		)
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m AddAnimeModel) viewAnimeResults() (string, *tea.Cursor) {
	if m.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.loader.View()), nil
	}

	h := addAnimeMsgs.viewHeader("Results")
	var c *tea.Cursor
	// The filter has no margin, so we enforce
	if m.list.FilterState() == list.Filtering {
		h = ui.Style.MarginBottom(1).Render(h)
		c = m.list.FilterInput.Cursor()
		c.Shape = tea.CursorBlock
		c.Color = ansi.Yellow
		c.Y += lipgloss.Height(h)
		c.X += 2 // Adjust for custom margin
	}
	return lipgloss.JoinVertical(lipgloss.Left, h, m.list.View()), c
}

func (m AddAnimeModel) updateAnimeReview(msg tea.Msg) (AddAnimeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			// handle back
		}
	}
	return m, nil
}

func (m AddAnimeModel) viewAnimeReview() (string, *tea.Cursor) {
	return "", nil
}

func (m *AddAnimeModel) reset() {
	m.state.view = Add_QueryAnime
	m.input.Reset()
}

func (m AddAnimeModel) findAnime(query string) tea.Cmd {
	return func() tea.Msg {
		anime, err := kitsu.FindAnime(query, []kitsu.AnimeStatus{kitsu.AnimeNew}, 10)
		if err != nil {
			return FetchErrorMsg(err)
		}
		list := make([]list.Item, len(anime))
		for i, item := range anime {
			list[i] = ui.NewListItem(
				item.Attributes.CanonicalTitle,
				item.Attributes.Titles.English,
				i,
			)
		}
		return FetchedKitsuEntriesMsg{list, anime}
	}
}

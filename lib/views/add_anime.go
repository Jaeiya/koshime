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

type addAnimeViewState interface {
	Update(msg tea.Msg, m *AddAnimeModel) tea.Cmd
	View(m AddAnimeModel) (string, *tea.Cursor)
	ShortHelp() []key.Binding
	FullHelp() [][]key.Binding
}

type AddAnimeModel struct {
	windowSize struct {
		width  int
		height int
	}
	loader       ui.LoaderModel
	viewStateMap map[AddAnimeView]addAnimeViewState
	currentView  AddAnimeView
}

func NewAddAnimeModel(db *database.Database) AddAnimeModel {
	m := AddAnimeModel{}
	m.loader = ui.NewLoader()
	m.viewStateMap = make(map[AddAnimeView]addAnimeViewState, 2)
	m.viewStateMap[Add_QueryAnime] = newAddAnimeQueryView()
	m.viewStateMap[Add_Results] = newAddAnimeResultsView()
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

	if v, exists := m.viewStateMap[m.currentView]; exists {
		cmd = v.Update(msg, &m)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AddAnimeModel) View() (string, *tea.Cursor) {
	if v, exists := m.viewStateMap[m.currentView]; exists {
		return v.View(m)
	}
	return "AddAnimeModel::missing view", nil
}

func (m AddAnimeModel) ShortHelp() []key.Binding {
	if v, exists := m.viewStateMap[m.currentView]; exists {
		return v.ShortHelp()
	}
	return []key.Binding{}
}

func (m AddAnimeModel) FullHelp() [][]key.Binding {
	if v, exists := m.viewStateMap[m.currentView]; exists {
		return v.FullHelp()
	}
	return [][]key.Binding{}
}

func (m *AddAnimeModel) SetView(v AddAnimeView) {
	m.currentView = v
}

type addAnimeQueryView struct {
	minInputLen int
	input       textinput.Model
}

func newAddAnimeQueryView() *addAnimeQueryView {
	v := &addAnimeQueryView{}
	v.input = ui.NewTextInput()
	v.input.SetWidth(30)
	v.input.Placeholder = "Enter query"
	return v
}

func (v *addAnimeQueryView) Update(msg tea.Msg, m *AddAnimeModel) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			return abort

		case key.Matches(msg, keyMap.Submit):
			// Do not submit query below min input length
			if utils.RuneCount(v.input.Value()) < v.minInputLen {
				break
			}
			m.loader, cmd = m.loader.Start("Finding Anime")
			m.SetView(Add_Results)
			return tea.Batch(cmd, v.findAnime(v.input.Value()))
		}
	}

	v.input, cmd = v.input.Update(msg)
	return cmd
}

func (v addAnimeQueryView) View(m AddAnimeModel) (string, *tea.Cursor) {
	c := v.input.Cursor()
	c.Shape = tea.CursorBar

	header := ui.Style.MarginBottom(1).Render(addAnimeMsgs.header)
	body := lipgloss.JoinVertical(lipgloss.Left, header, addAnimeMsgs.queryDesc)

	c.Y += lipgloss.Height(body)
	inputView := lipgloss.JoinHorizontal(
		lipgloss.Left,
		v.input.View(),
		ui.DisplayCharLimit(v.minInputLen, v.input.Value()),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		body,
		inputView,
	), c
}

func (v addAnimeQueryView) ShortHelp() []key.Binding {
	return []key.Binding{
		keyMap.Submit, keyMap.MainMenu,
	}
}

func (v addAnimeQueryView) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (v addAnimeQueryView) findAnime(query string) tea.Cmd {
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

type addAnimeResultsView struct {
	list         list.Model
	itemsPerPage int
}

func newAddAnimeResultsView() *addAnimeResultsView {
	v := &addAnimeResultsView{}
	return v
}

func (v *addAnimeResultsView) Update(msg tea.Msg, m *AddAnimeModel) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			m.reset()
		}

	case FetchedKitsuEntriesMsg:
		m.loader.Stop()
		v.list = ui.NewList(
			ui.ListOptions{
				Items: msg.items,
				// ShortHelpKeys: []key.Binding{m.keys.backspace},
				Width:        m.windowSize.width,
				MaxHeight:    int(float64(m.windowSize.height) * 0.66),
				ItemsPerPage: v.itemsPerPage,
			},
		)
	}

	if len(v.list.Items()) > 0 {
		v.list, cmd = v.list.Update(msg)
	}
	return cmd
}

func (v addAnimeResultsView) View(m AddAnimeModel) (string, *tea.Cursor) {
	if m.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.loader.View()), nil
	}

	h := addAnimeMsgs.viewHeader("Results")
	var c *tea.Cursor
	// The filter has no margin, so we enforce
	if v.list.FilterState() == list.Filtering {
		h = ui.Style.MarginBottom(1).Render(h)
		c = v.list.FilterInput.Cursor()
		c.Shape = tea.CursorBlock
		c.Color = ansi.Yellow
		c.Y += lipgloss.Height(h)
		c.X += 2 // Adjust for custom margin
	}
	return lipgloss.JoinVertical(lipgloss.Left, h, v.list.View()), c
}

func (v addAnimeResultsView) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (v addAnimeResultsView) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m *AddAnimeModel) reset() {
	m.SetView(Add_QueryAnime)
}

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

type ReviewAnimeMsg = ui.AnimeInfo

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
	m.viewStateMap[Add_ReviewAnime] = newAddAnimeReviewView()
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

func (m *AddAnimeModel) reset() {
	m.SetView(Add_QueryAnime)
}

type addAnimeQueryView struct {
	minInputLen int
	input       textinput.Model
	animeFinder AnimeFinder
}

func newAddAnimeQueryView() *addAnimeQueryView {
	v := &addAnimeQueryView{}
	v.input = ui.NewTextInput()
	v.input.SetWidth(30)
	v.input.Placeholder = "Enter query"
	v.minInputLen = 4
	v.animeFinder = NewKitsuAnimeFinder(10, []kitsu.AnimeStatus{kitsu.AnimeNew})
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
		anime, err := v.animeFinder.Search(query)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return anime
	}
}

type addAnimeResultsView struct {
	list         list.Model
	itemsPerPage int
	results      []ui.AnimeInfo
}

func newAddAnimeResultsView() *addAnimeResultsView {
	v := &addAnimeResultsView{}
	v.itemsPerPage = 5
	return v
}

func (v *addAnimeResultsView) Update(msg tea.Msg, m *AddAnimeModel) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			m.reset()

		case key.Matches(msg, keyMap.Select):
			m.SetView(Add_ReviewAnime)
			item := v.list.SelectedItem().(ui.ListItem)
			return func() tea.Msg {
				return v.results[item.Index()]
			}
		}

	case AnimeFinderResult:
		m.loader.Stop()
		v.results = msg.infoItems
		v.list = ui.NewList(
			ui.ListOptions{
				Items: msg.listItems,
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

type addAnimeReviewView struct {
	animeInfo ui.AnimeInfo
}

func newAddAnimeReviewView() *addAnimeReviewView {
	v := &addAnimeReviewView{}
	return v
}

func (v *addAnimeReviewView) Update(msg tea.Msg, m *AddAnimeModel) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.EscBack):
			m.SetView(Add_Results)
		}
	case ReviewAnimeMsg:
		v.animeInfo = msg
	}
	return nil
}

func (v addAnimeReviewView) View(m AddAnimeModel) (string, *tea.Cursor) {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		findAnimeMsgs.viewHeader("Entry Info"),
		"",
		ui.DisplayAnimeInfo(v.animeInfo),
	), nil
}

func (v addAnimeReviewView) ShortHelp() []key.Binding {
	return []key.Binding{
		keyMap.EscBack,
	}
}

func (v addAnimeReviewView) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

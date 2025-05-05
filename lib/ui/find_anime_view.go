package ui

import (
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type MenuFindView int

const (
	Find_DefaultView = MenuFindView(iota)
	Find_ReviewView
)

type animeItem struct {
	title, desc string
}

func (i animeItem) Title() string       { return i.title }
func (i animeItem) Description() string { return i.desc }
func (i animeItem) FilterValue() string { return i.title }

type findMenuModel struct {
	list       list.Model
	input      textinput.Model
	loader     LoaderModel
	windowSize struct {
		width  int
		height int
	}
	maxResults int
	state      struct {
		view    MenuFindView
		results []kitsu.Anime
		find    struct {
			passed bool
			failed bool
		}
	}
}

func NewFindMenuModel(maxResults int) findMenuModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(true)
	l.DisableQuitKeybindings()
	input := textinput.New()
	input.SetWidth(30)
	input.Focus()
	input.CharLimit = 40
	input.Prompt = "   > "
	input.EchoCharacter = '•'
	input.Styles.Focused.Prompt = inputPromptStyle
	input.Styles.Focused.Text = inputTextStyle

	return findMenuModel{list: l, input: input, loader: NewLoader(), maxResults: maxResults}
}

func (m findMenuModel) Init() tea.Cmd {
	return nil
}

func (m findMenuModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize.width = msg.Width
		m.windowSize.height = msg.Height

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.MainMenu):
			// Exit to main menu only when filter is cancelled
			if m.state.view == Find_ReviewView {
				if m.list.FilterState() > list.Unfiltered {
					break
				}
			}
			m.Reset()
			return m, func() tea.Msg { return AbortMsg{} }

		case key.Matches(msg, keyMap.Submit):
			switch m.state.view {
			case Find_DefaultView:
				if !m.loader.IsLoading() {
					m.loader.SetLoadingState(true)
					m.loader.SetText("Finding Anime")
					return m, tea.Batch(m.loader.Start, m.findAnime(m.input.Value()))
				}
			}
		}

	case FetchErrorMsg:
		m.loader.Stop()
		m.input.Reset()
		panic(msg)

	case []kitsu.Anime:
		m.input.Reset()
		m.loader.Stop()
		m.list = m.newAnimeList(msg)
		m.state.find.passed = true
		m.state.view = Find_ReviewView
	}

	m.loader, cmd = m.loader.Update(msg)
	cmds = append(cmds, cmd)

	switch m.state.view {
	case Find_DefaultView:
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	case Find_ReviewView:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m findMenuModel) View() (string, *tea.Cursor) {
	if m.loader.IsLoading() {
		return style.MarginTop(1).Render(m.loader.View()), nil
	}

	if m.state.find.failed {
	}

	if m.state.find.passed {
		return m.list.View(), nil
	}

	c := m.input.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		textStyle.MarginTop(1).Render("Enter partial or full anime title."),
		style.MarginTop(1).Render(m.input.View()),
	)
	c.Y += lipgloss.Height(view) - 1
	return lipgloss.JoinVertical(lipgloss.Left, view), c
}

func (m findMenuModel) ShortHelp() []key.Binding {
	switch m.state.view {
	case Find_DefaultView:
		return []key.Binding{
			keyMap.Submit, keyMap.MainMenu,
		}
	}
	return []key.Binding{}
}

func (m findMenuModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m *findMenuModel) Reset() {
	m.state.view = Find_DefaultView
	m.input.Reset()
	m.state.results = nil
	m.state.find.passed = false
	m.state.find.failed = false
}

func (m findMenuModel) newAnimeList(animeList []kitsu.Anime) list.Model {
	items := make([]list.Item, len(animeList))
	for i, anime := range animeList {
		items[i] = animeItem{
			anime.Attributes.CanonicalTitle,
			anime.Attributes.Titles.English,
		}
	}
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(ansi.BrightGreen).
		BorderForeground(ansi.BrightGreen)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Foreground(ansi.Blue)
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(lipgloss.Color("#A7A7B5"))
	d.Styles.NormalDesc = d.Styles.NormalDesc.Foreground(lipgloss.Color("#696974"))
	l := list.New(items, d, m.windowSize.width, int(float64(m.windowSize.height)*0.66))
	l.Title = "Anime Results"
	l.SetShowTitle(true)
	l.DisableQuitKeybindings()
	l.Styles.Title = style.Foreground(ansi.BrightBlue).Background(ansi.Black)

	return l
}

func (m findMenuModel) findAnime(query string) tea.Cmd {
	return func() tea.Msg {
		anime, err := kitsu.FindAnime(
			query,
			[]kitsu.AnimeStatus{kitsu.AnimeNew, kitsu.AnimeFinished},
			m.maxResults,
		)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return anime
	}
}

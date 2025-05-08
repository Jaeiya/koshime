package views

import (
	"fmt"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
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
	Find_ResultsView
)

type animeListItem struct {
	title, desc string
}

func (i animeListItem) Title() string       { return i.title }
func (i animeListItem) Description() string { return i.desc }
func (i animeListItem) FilterValue() string { return i.title }

type fetchedItems []list.Item

type findMenuModel struct {
	list       list.Model
	input      textinput.Model
	loader     ui.LoaderModel
	db         *database.Database
	windowSize struct {
		width  int
		height int
	}
	maxResults int
	sourceMap  map[AnimeSource]string
	state      struct {
		fetchErr FetchErrorMsg
		source   AnimeSource
		view     MenuFindView
		results  []kitsu.Anime
		find     struct {
			passed bool
			failed bool
		}
	}
	keys struct {
		tab       key.Binding
		backspace key.Binding
	}
}

func NewFindMenuModel(db *database.Database, maxResults int) findMenuModel {
	input := ui.NewTextInput()
	input.SetWidth(30)
	input.Focus()
	m := findMenuModel{input: input, loader: ui.NewLoader(), maxResults: maxResults}
	m.db = db
	m.keys.tab = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "source"))
	m.keys.backspace = key.NewBinding(
		key.WithKeys("backspace"),
		key.WithHelp("backspace", "back"),
	)

	m.sourceMap = map[AnimeSource]string{
		Kitsu: findAnimeMsgs.kitsu,
		Cache: findAnimeMsgs.cache,
	}
	return m
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
			// Go back to find-view from results-view
			if m.state.view == Find_ResultsView && m.list.FilterState() != list.Filtering {
				m.Reset()
				break
			}
			m.Reset()
			// Back to main menu
			return m, abort

		case key.Matches(msg, m.keys.backspace):
			// Go back to find-view from results-view
			if m.state.view == Find_ResultsView && m.list.FilterState() != list.Filtering {
				m.Reset()
				return m, nil
			}

		case key.Matches(msg, keyMap.Submit):
			switch m.state.view {
			case Find_DefaultView:
				if m.loader.IsLoading() {
					break
				}
				m.loader.SetLoadingState(true)
				m.loader.SetText("Finding Anime")
				return m, tea.Batch(m.loader.Start, m.findAnime(m.input.Value()))
			}

		case key.Matches(msg, m.keys.tab):
			if m.state.view == Find_DefaultView {
				m.state.source = (m.state.source + 1) % 2
			}
		}

	case FetchErrorMsg:
		m.loader.Stop()
		m.input.Reset()
		m.state.find.failed = true
		m.state.fetchErr = msg

	case fetchedItems:
		m.input.Reset()
		m.loader.Stop()
		m.list = m.newAnimeList(msg)
		m.state.find.passed = true
		m.state.view = Find_ResultsView
	}

	m.loader, cmd = m.loader.Update(msg)
	cmds = append(cmds, cmd)

	switch m.state.view {
	case Find_DefaultView:
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	case Find_ResultsView:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m findMenuModel) View() (string, *tea.Cursor) {
	if m.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.loader.View()), nil
	}

	if m.state.find.failed {
		return m.state.fetchErr.Error(), nil
	}

	if m.state.find.passed {
		return m.list.View(), nil
	}

	c := m.input.Cursor()
	c.Shape = tea.CursorBar

	search := ui.TextStyle.Foreground(ansi.BrightBlack).
		Render("Source: " + m.sourceMap[m.state.source])

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		findAnimeMsgs.title,
		search,
		ui.Style.MarginTop(1).Render(m.input.View()),
	)
	c.Y += lipgloss.Height(view) - 1
	return lipgloss.JoinVertical(lipgloss.Left, view), c
}

func (m findMenuModel) ShortHelp() []key.Binding {
	switch m.state.view {
	case Find_DefaultView:
		return []key.Binding{
			keyMap.Submit, keyMap.MainMenu, m.keys.tab,
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

func (m findMenuModel) newAnimeList(animeList []list.Item) list.Model {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(ansi.BrightGreen).
		BorderForeground(ansi.BrightGreen)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Foreground(ansi.Blue)
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(lipgloss.Color("#A7A7B5"))
	d.Styles.NormalDesc = d.Styles.NormalDesc.Foreground(lipgloss.Color("#696974"))
	l := list.New(animeList, d, m.windowSize.width, int(float64(m.windowSize.height)*0.66))
	l.Title = "Anime Results"
	l.Help.Styles.ShortDesc = ui.HelpDescStyle
	l.Help.Styles.FullDesc = ui.HelpDescStyle
	l.Help.Styles.ShortKey = ui.HelpKeyStyle
	l.Help.Styles.FullKey = ui.HelpKeyStyle
	l.SetShowTitle(true)
	l.DisableQuitKeybindings()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{m.keys.backspace}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	l.Styles.Title = ui.Style.Foreground(ansi.BrightBlue).Background(ansi.Black)

	return l
}

func (m findMenuModel) findAnime(query string) tea.Cmd {
	return func() tea.Msg {
		var items []list.Item
		switch m.state.source {
		case Kitsu:
			anime, err := kitsu.FindAnime(
				query,
				[]kitsu.AnimeStatus{kitsu.AnimeNew, kitsu.AnimeFinished},
				m.maxResults,
			)
			if err != nil {
				return FetchErrorMsg(err)
			}
			if len(anime) == 0 {
				return FetchErrorMsg(fmt.Errorf("no anime found"))
			}
			items = make([]list.Item, len(anime))
			for i, item := range anime {
				items[i] = animeListItem{
					item.Attributes.CanonicalTitle,
					item.Attributes.Titles.English,
				}
			}

		case Cache:
			anime, err := m.db.FindAnime(query)
			if err != nil {
				return FetchErrorMsg(err)
			}
			if len(anime) == 0 {
				return FetchErrorMsg(fmt.Errorf("no anime found"))
			}
			items = make([]list.Item, len(anime))
			for i, item := range anime {
				items[i] = animeListItem{
					item.JPN_Title,
					item.ENG_Title,
				}
			}
		}

		if len(items) == 0 {
			return FetchErrorMsg(fmt.Errorf("unrecognized anime source: %d", m.state.source))
		}
		return fetchedItems(items)
	}
}

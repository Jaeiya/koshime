package views

import (
	"fmt"
	"net/url"
	"strconv"

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

type MenuFindView int

const (
	Find_DefaultView = MenuFindView(iota)
	Find_ResultsView
	Find_AnimeView
	Find_LoadingView
)

type animeListItem struct {
	title, desc string
	index       int
}

func (i animeListItem) Title() string       { return i.title }
func (i animeListItem) Description() string { return i.desc }
func (i animeListItem) FilterValue() string { return i.title + " " + i.desc }

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
		fetchErr      FetchErrorMsg
		source        AnimeSource
		view          MenuFindView
		kitsuResults  []kitsu.Anime
		localResults  []database.LibraryEntry
		selectedIndex int
		find          struct {
			failed   bool
			notFound bool
		}
	}
	keys struct {
		tab       key.Binding
		backspace key.Binding
		escBack   key.Binding
	}
}

func NewFindMenuModel(db *database.Database, maxResults int) findMenuModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	input := ui.NewTextInput()
	input.SetWidth(30)
	input.Focus()
	m := findMenuModel{input: input, loader: ui.NewLoader(), maxResults: maxResults, list: l}
	m.db = db
	m.keys.tab = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "source"))
	m.keys.backspace = key.NewBinding(
		key.WithKeys("left", "backspace"),
		key.WithHelp("←", "back"),
	)
	m.keys.escBack = key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc/←", "back"),
	)

	m.sourceMap = map[AnimeSource]string{
		Kitsu: findAnimeMsgs.kitsu,
		Local: findAnimeMsgs.local,
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
			if m.state.view == Find_ResultsView {
				if m.list.FilterState() > list.Unfiltered {
					break
				}
				m.Reset()
				return m, nil
			}

			if m.state.view == Find_AnimeView {
				m.state.view = Find_ResultsView
				return m, nil
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

			if m.state.view == Find_AnimeView {
				m.state.view = Find_ResultsView
				return m, nil
			}

		case key.Matches(msg, keyMap.Submit):
			switch m.state.view {
			case Find_DefaultView:
				if m.loader.IsLoading() {
					break
				}
				m.loader.SetLoadingState(true)
				m.loader.SetText("Find Anime")
				m.state.view = Find_LoadingView
				return m, tea.Batch(m.loader.Start, m.findAnime(m.input.Value()))

			case Find_ResultsView:
				item := m.list.SelectedItem().(animeListItem)
				m.state.selectedIndex = item.index
				m.state.view = Find_AnimeView
			}

		case key.Matches(msg, m.keys.tab):
			if m.state.view == Find_DefaultView {
				m.state.source = (m.state.source + 1) % 2
			}
		}

	case FetchErrorMsg, FetchedNoResultsMsg, FetchedListItemsMsg[kitsu.Anime], FetchedListItemsMsg[database.LibraryEntry]:
		m.loader.Stop()
		m.state.view = Find_ResultsView
		switch msg := msg.(type) {
		case FetchErrorMsg:
			m.state.find.failed = true
			m.state.fetchErr = msg

		case FetchedNoResultsMsg:
			m.state.find.notFound = true

		case FetchedListItemsMsg[database.LibraryEntry]:
			m.state.localResults = msg.results
			m.list = ui.NewList(
				ui.ListOptions{
					Items:         msg.items,
					ShortHelpKeys: []key.Binding{m.keys.backspace},
					Width:         m.windowSize.width,
					MaxHeight:     int(float64(m.windowSize.height) * 0.66),
					ItemsPerPage:  5,
				},
			)

		case FetchedListItemsMsg[kitsu.Anime]:
			m.state.kitsuResults = msg.results
			m.list = ui.NewList(
				ui.ListOptions{
					Items:         msg.items,
					ShortHelpKeys: []key.Binding{m.keys.backspace},
					Width:         m.windowSize.width,
					MaxHeight:     int(float64(m.windowSize.height) * 0.66),
					ItemsPerPage:  5,
				},
			)
		}
	}

	if m.loader.IsLoading() {
		m.loader, cmd = m.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

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
	switch m.state.view {
	case Find_LoadingView:
		if m.loader.IsLoading() {
			return ui.Style.MarginTop(1).Render(m.loader.View()), nil
		}
	case Find_ResultsView:
		return m.ViewResults()

	case Find_AnimeView:
		return m.ViewAnime(), nil

	}

	c := m.input.Cursor()
	c.Shape = tea.CursorBar

	search := ui.TextStyle.Foreground(ansi.BrightBlack).
		Render("Source: " + m.sourceMap[m.state.source])

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		findAnimeMsgs.header,
		findAnimeMsgs.title,
		search,
		ui.Style.MarginTop(1).Render(m.input.View()),
	)
	c.Y += lipgloss.Height(view) - 1
	return lipgloss.JoinVertical(lipgloss.Left, view), c
}

func (m findMenuModel) ViewResults() (string, *tea.Cursor) {
	if m.state.find.failed {
		return m.state.fetchErr.Error(), nil
	}

	if m.state.find.notFound {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			findAnimeMsgs.header,
			findAnimeMsgs.notFound(m.input.Value()),
		), nil
	}

	h := findAnimeMsgs.header
	var c *tea.Cursor
	// The filter has no margin, so we enforce
	if m.list.FilterState() == list.Filtering {
		h = ui.Style.MarginBottom(1).Render(h)
		c = m.list.FilterInput.Cursor()
		c.Shape = tea.CursorBlock
		c.Color = ansi.Yellow
		c.Y += lipgloss.Height(h)
		c.X += 2
	}
	return lipgloss.JoinVertical(lipgloss.Left, h, m.list.View()), c
}

func (m findMenuModel) ViewAnime() string {
	if m.state.localResults != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			findAnimeMsgs.header,
			"",
			m.displayAnimeInfo(m.state.localResults[m.state.selectedIndex]),
		)
	}

	if m.state.kitsuResults != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			findAnimeMsgs.header,
			"",
			m.displayAnimeInfo(m.state.kitsuResults[m.state.selectedIndex]),
		)
	}

	return "missing local or kitsu results to display"
}

func (m findMenuModel) ShortHelp() []key.Binding {
	switch m.state.view {
	case Find_DefaultView:
		return []key.Binding{
			keyMap.Submit, keyMap.MainMenu, m.keys.tab,
		}

	case Find_ResultsView, Find_AnimeView:
		if m.state.find.notFound || m.state.view == Find_AnimeView {
			return []key.Binding{
				m.keys.escBack,
			}
		}

	case Find_LoadingView:
		return []key.Binding{}

	}

	return []key.Binding{}
}

func (m findMenuModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m *findMenuModel) Reset() {
	m.state.view = Find_DefaultView
	m.input.Reset()
	m.state.kitsuResults = nil
	m.state.localResults = nil
	m.state.find.failed = false
	m.state.find.notFound = false
}

func (m *findMenuModel) findAnime(query string) tea.Cmd {
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
				return FetchedNoResultsMsg{}
			}
			m.state.kitsuResults = anime
			items = make([]list.Item, len(anime))
			for i, item := range anime {
				items[i] = animeListItem{
					item.Attributes.CanonicalTitle,
					item.Attributes.Titles.English,
					i,
				}
			}
			return FetchedListItemsMsg[kitsu.Anime]{
				items:   items,
				results: anime,
			}

		case Local:
			anime, err := m.db.FindAnime(query)
			if err != nil {
				return FetchErrorMsg(err)
			}
			if len(anime) == 0 {
				return FetchedNoResultsMsg{}
			}
			m.state.localResults = anime
			items = make([]list.Item, len(anime))
			for i, item := range anime {
				items[i] = animeListItem{
					item.JPN_Title,
					item.ENG_Title,
					i,
				}
			}
			return FetchedListItemsMsg[database.LibraryEntry]{
				items:   items,
				results: anime,
			}
		}

		return FetchErrorMsg(fmt.Errorf("unrecognized anime source: %d", m.state.source))
	}
}

func (m findMenuModel) displayAnimeInfo(animeData any) string {
	switch d := animeData.(type) {
	case database.LibraryEntry:
		return m.stringifyAnimeInfo(
			d.JPN_Title,
			d.ENG_Title,
			d.Slug,
			"",
			d.AltTitles,
			d.Progress,
			d.Episodes,
		)

	case kitsu.Anime:
		return m.stringifyAnimeInfo(
			d.Attributes.CanonicalTitle,
			d.Attributes.Titles.English,
			d.Attributes.Slug,
			d.Attributes.Synopsis,
			d.Attributes.AltTitles,
			-1,
			d.Attributes.EpCount,
		)
	}
	return "unsupported anime data type"
}

func (findMenuModel) stringifyAnimeInfo(
	title, engTitle, slug, synopsis string,
	altTitles []string,
	progress, episodes int,
) string {
	itemCount := 4
	if synopsis != "" {
		itemCount += 1
	}
	items := make([]string, itemCount+len(altTitles))
	headers := make([]string, len(items))

	headers[0], headers[1] = utils.ColorText(";b;Title"), utils.ColorText(";dc;English")
	items[0], items[1] = title, engTitle

	for i, altTitle := range altTitles {
		headers[i+2] = utils.ColorText(";db;AltTitle")
		items[i+2] = altTitle
	}
	itemPos := 2 + len(altTitles)

	totalEpsStr := strconv.Itoa(episodes)
	if episodes == 0 {
		totalEpsStr = "Unknown"
	}
	link, _ := url.JoinPath(kitsu.KitsuDomain, slug)
	link = utils.ColorText(";bk;" + link)

	if progress > -1 {
		headers[itemPos] = utils.ColorText(";y;Progress")
		items[itemPos] = utils.ColorText(
			fmt.Sprintf(";dg;%d ;y;/ ;m;%s", progress, totalEpsStr),
		)
	} else {
		headers[itemPos] = utils.ColorText(";dc;Episodes")
		items[itemPos] = utils.ColorText(fmt.Sprintf(";m;%s", totalEpsStr))
	}
	if synopsis != "" {
		itemPos++
		headers[itemPos] = utils.ColorText(";dc;Synopsis")
		items[itemPos] = synopsis
	}
	itemPos++
	headers[itemPos] = utils.ColorText(";x;link")
	items[itemPos] = link

	return newList(
		headers,
		items,
	)
}

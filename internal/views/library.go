package views

import (
	"fmt"
	"slices"

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
)

const (
	MenuDrop = ui.MenuIndexMsg(iota)
	MenuComplete
	MenuDelete
)

type Library_View int

const (
	LibraryMenu = Library_View(iota)
	LibrarySearch
	LibraryReload
	LibraryFeed
	LibraryDrop
	LibraryComplete
	LibraryDelete
)

type (
	LibraryReloadedMsg struct {
		Value     []kitsu.Anime
		LastIndex int
	}
	LibraryAnimeSearchMsg struct {
		Value kitsu.Anime
		Found bool
	}
	LibraryFeedMsg struct {
		Value []kitsu.Anime
	}
)

type Library_Model struct {
	windowSize tea.WindowSizeMsg
	ui         struct {
		loader       ui.LoaderModel
		consent      ui.ConsentModel
		animeDisplay *AnimeDisplayModel
		menu         ui.MenuModel
		list         list.Model
		input        textinput.Model
	}
	keys struct {
		reload key.Binding
		search key.Binding
		feed   key.Binding
	}
	db          *database.Database
	state       Library_State
	minInputLen int
}

type Library_State struct {
	err               error
	view              Library_View
	anime             []kitsu.Anime
	animeIndex        int
	searchAnimeResult struct {
		Value kitsu.Anime
		Found bool
	}
	filesNotFound bool
}

func newLibraryModel(db *database.Database) Library_Model {
	m := Library_Model{db: db, minInputLen: 4}
	m.ui.list = ui.NewList(ui.ListOptions{})
	m.ui.loader = ui.NewLoader()
	m.ui.animeDisplay = NewAnimeDisplayModel()
	m.ui.input = ui.NewTextInput()
	m.ui.menu = ui.NewMenuModel([]string{
		"Drop",
		"Complete",
		"Delete",
	}, ui.WithMenuRotation(), ui.WithMenuDescriptions([]string{
		`Drops the selected anime above and removes it from local database.`,
		`Sets status of selected anime above, to completed.`,
		`Deletes the selected anime above from Kitsu and local database.`,
	}))
	m.keys.reload = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload"))
	m.keys.search = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "search"))
	m.keys.feed = key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "update feeds"))
	anime := db.Anime()
	slices.SortFunc(anime, app.CompareAnime)
	m.state.anime = anime
	return m
}

func (m Library_Model) Init() tea.Cmd {
	return nil
}

func (m Library_Model) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.search):
			if m.state.view > LibraryMenu {
				break
			}
			m.state.view = LibrarySearch
			return m, nil

		case key.Matches(msg, m.keys.reload):
			// Never execute reload in another view
			if m.state.view > LibraryMenu {
				break
			}
			m.state.view = LibraryReload

		case key.Matches(msg, m.keys.feed):
			// Do not allow this action unless user has qBt setup
			if m.db.Profile().QbtPort == 0 {
				break
			}
			// Never execute feed update in another view
			if m.state.view > LibraryMenu {
				break
			}
			m.state.view = LibraryFeed

		case key.Matches(msg, ui.KeyMap.MainMenu):
			if m.state.view == LibraryMenu {
				return m, exitToMenu
			}
			if m.state.view == LibrarySearch && m.state.searchAnimeResult.Found {
				break
			}
			// If we were moved to another view via search view
			if m.state.searchAnimeResult.Found {
				m.state.view = LibrarySearch
				return m, nil
			}
			m.state.view = LibraryMenu
		}

	case error:
		m.state.err = msg
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case LibraryMenu:
		m, cmd = m.UpdateMenu(msg)
		cmds = append(cmds, cmd)

	case LibrarySearch:
		m, cmd = m.UpdateSearch(msg)
		cmds = append(cmds, cmd)

	case LibraryReload:
		m, cmd = m.UpdateReload(msg)
		cmds = append(cmds, cmd)

	case LibraryFeed:
		m, cmd = m.UpdateFeed(msg)
		cmds = append(cmds, cmd)

	case LibraryDelete:
		m, cmd = m.UpdateDelete(msg)
		cmds = append(cmds, cmd)

	case LibraryDrop:
		m, cmd = m.UpdateDrop(msg)
		cmds = append(cmds, cmd)

	case LibraryComplete:
		m, cmd = m.UpdateCompleted(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Library_Model) View() tea.View {
	if m.ui.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.ui.loader.View()))
	}

	if m.state.err != nil {
		return tea.NewView(ui.DisplayError(m.state.err))
	}

	switch m.state.view {
	case LibraryMenu:
		return m.ViewMenu()
	case LibrarySearch:
		return m.ViewSearch()
	case LibraryReload:
		return m.ViewReload()
	case LibraryFeed:
		return m.ViewFeed()
	case LibraryDrop:
		return m.ViewDrop()
	case LibraryDelete:
		return m.ViewDeleting()
	case LibraryComplete:
		return m.ViewCompleted()
	default:
		return tea.NewView("missing Library view")
	}
}

func (m Library_Model) ShortHelp() []key.Binding {
	keys := []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down}

	if m.state.filesNotFound {
		return []key.Binding{ui.KeyMap.EscBack}
	}

	if m.state.view == LibrarySearch {
		if m.state.searchAnimeResult.Found {
			keys = append(keys, m.ui.animeDisplay.ShortHelp()[0], ui.KeyMap.EscBack)
			return keys
		}
		return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.EscBack}
	}

	if m.state.view > LibrarySearch {
		return []key.Binding{
			ui.KeyMap.Up,
			ui.KeyMap.Down,
			ui.KeyMap.Select,
			ui.KeyMap.EscBack,
		}
	}

	if len(m.state.anime) > 0 {
		switch {
		case m.state.animeIndex == len(m.state.anime)-1:
			keys = append(keys, ui.KeyMap.Prev)

		case m.state.animeIndex > 0:
			keys = append(keys, ui.KeyMap.Prev, ui.KeyMap.Next)

		default:
			keys = append(keys, ui.KeyMap.Next)
		}
	}
	keys = append(keys, m.ui.animeDisplay.ShortHelp()[0], ui.KeyMap.HelpMore)
	return keys
}

func (m Library_Model) FullHelp() [][]key.Binding {
	// Prevent conflicts with list component
	if m.state.view == LibrarySearch {
		return nil
	}

	// We only show the user this key if they have qBt setup.
	feedHelp := m.keys.feed
	if m.db.Profile().QbtPort == 0 {
		feedHelp = key.Binding{}
	}

	return [][]key.Binding{
		{
			ui.KeyMap.Up,
			ui.KeyMap.Down,
			ui.KeyMap.Prev,
			ui.KeyMap.Next,
			ui.KeyMap.Select,
		},
		{
			m.ui.animeDisplay.ShortHelp()[0],
			m.keys.reload,
			m.keys.search,
			feedHelp,
			ui.KeyMap.MainMenu,
			ui.KeyMap.HelpLess,
		},
	}
}

func (m Library_Model) UpdateMenu(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Left):
			if m.state.animeIndex == 0 {
				break
			}
			m.state.animeIndex--

		case key.Matches(msg, ui.KeyMap.Right):
			if m.state.animeIndex == len(m.state.anime)-1 {
				break
			}
			m.state.animeIndex++

		}

	case ui.MenuIndexMsg:
		switch msg {
		case MenuDrop:
			m.state.view = LibraryDrop
		case MenuComplete:
			m.state.view = LibraryComplete
		case MenuDelete:
			m.state.view = LibraryDelete
		}
	}

	if m.ui.animeDisplay != nil {
		m.ui.animeDisplay.Update(msg)
	}
	m.ui.menu, cmd = m.ui.menu.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Library_Model) ViewMenu() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Library"),
		"",
		ui.DisplayText([]string{
			fmt.Sprintf(`There are ;g;%d ;x;anime in your local library.`, len(m.state.anime)),
		}),
		"",
		m.ui.animeDisplay.View(m.state.anime[m.state.animeIndex]),
		ui.DisplayTitle("Manage"),
		"",
		ui.DisplayText([]string{
			`Select an action to apply to the above entry.`,
		}),
		"",
		m.ui.menu.View(),
	))
}

func (m Library_Model) UpdateSearch(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Left), key.Matches(msg, ui.KeyMap.Right):
			// Do not accidentally pass these key events to the UpdateMenu method
			if m.state.searchAnimeResult.Found {
				return m, nil
			}

		case key.Matches(msg, ui.KeyMap.Submit):
			if m.state.searchAnimeResult.Found {
				break
			}
			if utils.RuneCount(m.ui.input.Value()) >= m.minInputLen {
				return m, m.searchLibrary(m.ui.input.Value())
			}

		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.state.searchAnimeResult.Found {
				m.state.searchAnimeResult.Found = false
				m.ui.input.Reset()
				return m, nil
			}
		}

	case LibraryAnimeSearchMsg:
		m.state.searchAnimeResult = msg
	}

	if m.state.searchAnimeResult.Found {
		m.ui.animeDisplay.Update(msg)
		m, cmd = m.UpdateMenu(msg)
		cmds = append(cmds, cmd)
	} else {
		m.ui.input, cmd = m.ui.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Library_Model) ViewSearch() tea.View {
	v := tea.NewView("")
	v.Cursor = m.ui.input.Cursor()
	v.Cursor.Shape = tea.CursorBar
	v.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Library", "Search"),
		ui.DisplayText([]string{
			"Quick search your library for top result.",
		}, 0, 1, 1),
		ui.Style.Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.ui.input.View(),
			ui.DisplayCharLimit(m.minInputLen, m.ui.input.Value()),
		)),
	)
	v.Cursor.Y = lipgloss.Height(v.Content) - 1

	if m.state.searchAnimeResult.Found {
		v.Content = lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle("Library", "Search Results"),
			"",
			m.ui.animeDisplay.View(m.state.searchAnimeResult.Value),
			ui.DisplayTitle("Manage"),
			"",
			ui.DisplayText([]string{
				`Select an action to apply to the above entry.`,
			}),
			"",
			m.ui.menu.View(),
		)
		v.Cursor = nil
	}
	return v
}

func (m Library_Model) UpdateReload(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = LibraryMenu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Reloading Library")
			return m, tea.Batch(cmd, m.reloadLibrary)
		}

	case LibraryReloadedMsg:
		m.state = Library_State{}
		m.state.anime = msg.Value
		m.ui.loader.Stop()
	}
	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m Library_Model) ViewReload() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Library", "Reloading"),
		"",
		ui.DisplayText(
			[]string{
				`This will ;r;overwrite;x; your current database with updated
information from ;dg;Kitsu;x;. This should only be necessary if ;dg;Kitsu;x; goes
out of sync with ;db;Koshime;x;'s database.`,
				`;dy;Getting ;y;out of sync ;dy;is only likely to happen if you
manually update your Kitsu library from the website.`,
			},
			1, 0, 1,
		),
		ui.TextStyle.Render(
			m.ui.consent.View(utils.ColorText(";b;Are you sure you want to reload?")),
		),
	))
}

func (m Library_Model) UpdateFeed(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = LibraryMenu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Updating Library Feeds")
			return m, tea.Batch(cmd, m.updateFeeds)
		}

	case LibraryFeedMsg:
		m.state = Library_State{}
		m.state.anime = msg.Value
		m.ui.loader.Stop()
	}
	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m Library_Model) ViewFeed() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Library", "Update Feeds"),
		"",
		ui.DisplayText(
			[]string{
				`This will update your current library with any feeds that
have not been assigned yet. You only need to do this if you deleted or lost
your database.`,
			},
			1, 0, 1,
		),
		ui.TextStyle.Render(
			m.ui.consent.View(
				utils.ColorText(";b;Are you sure you want to update your libraries feeds?"),
			),
		),
	))
}

func (m Library_Model) UpdateDelete(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = LibraryMenu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Deleting Entry")
			return m, tea.Batch(cmd, m.deleteAnime)
		}

	case LibraryReloadedMsg:
		m.state = Library_State{}
		m.state.anime = msg.Value
		if msg.LastIndex > 0 {
			m.state.animeIndex = msg.LastIndex - 1
		}
		m.ui.loader.Stop()
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m Library_Model) ViewDeleting() tea.View {
	anime := m.state.anime[m.state.animeIndex]
	if m.state.searchAnimeResult.Found {
		anime = m.state.searchAnimeResult.Value
	}

	title := anime.ENG_Title
	if title == "" {
		title = anime.JPN_Title
	}

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Library", "Deleting Entry"),
		"",
		ui.DisplayText(
			[]string{
				fmt.Sprintf(
					`;dc;%s ;x;will be deleted from your Kitsu library and
the local database.`, title,
				),
				`;y;[this action cannot be undone]`,
			}, 1, 0, 1,
		),
		ui.TextStyle.Render(
			m.ui.consent.View(utils.ColorText(";b;Are you sure?")),
		),
	))
}

func (m Library_Model) UpdateDrop(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = LibraryMenu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Dropping Anime")
			return m, tea.Batch(cmd, m.dropAnime)
		}

	case LibraryReloadedMsg:
		m.state = Library_State{}
		m.state.anime = msg.Value
		if msg.LastIndex > 0 {
			m.state.animeIndex = msg.LastIndex - 1
		}
		m.ui.loader.Stop()
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m Library_Model) ViewDrop() tea.View {
	anime := m.state.anime[m.state.animeIndex]
	if m.state.searchAnimeResult.Found {
		anime = m.state.searchAnimeResult.Value
	}

	title := anime.ENG_Title
	if title == "" {
		title = anime.JPN_Title
	}

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Library", "Drop"),
		"",
		ui.DisplayText([]string{
			fmt.Sprintf(`;dc;%s ;x;is about to be ;m;dropped;x;.`, title),
			`This is not the same as deletion. Dropping an anime sets its status to
;dm;dropped;x;, which stores it under the dropped tab in ;db;Kitsu;x;.`,
			`Dropping an anime is often times better than deletion because it keeps
track of the episodes you ;w;did;x; watch. This makes your watch time stats more
accurate.`,
		}, 1, 0, 1),
		ui.TextStyle.Render(m.ui.consent.View(utils.ColorText(";b;Are you sure?"))),
	))
}

func (m Library_Model) UpdateCompleted(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = LibraryMenu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Completing Anime")
			return m, tea.Batch(cmd, m.completeAnime)
		}

	case LibraryReloadedMsg:
		m.state = Library_State{}
		m.state.anime = msg.Value
		if msg.LastIndex > 0 {
			m.state.animeIndex = msg.LastIndex - 1
		}
		m.ui.loader.Stop()
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m Library_Model) ViewCompleted() tea.View {
	anime := m.state.anime[m.state.animeIndex]
	if m.state.searchAnimeResult.Found {
		anime = m.state.searchAnimeResult.Value
	}

	title := anime.ENG_Title
	if title == "" {
		title = anime.JPN_Title
	}

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Library", "Complete"),
		"",
		ui.DisplayText([]string{
			fmt.Sprintf(`;dc;%s ;x;is about to be ;b;completed;x;.`, title),
			`This should only be necessary if ;dg;Kitsu;x; messes up episode counts.
Updating the progress of an anime will auto-complete it on the last episode of a season.`,
		}, 1, 0, 1),
		ui.TextStyle.Render(m.ui.consent.View(utils.ColorText(";b;Are you sure?"))),
	))
}

func (m Library_Model) searchLibrary(search string) tea.Cmd {
	anime := m.db.Anime()
	return func() tea.Msg {
		a, found := app.FuzzyFindAnime(anime, search)
		return LibraryAnimeSearchMsg{
			Value: a,
			Found: found,
		}
	}
}

func (m Library_Model) reloadLibrary() tea.Msg {
	profile := m.db.Profile()
	anime, err := kitsu.GetUserAnime(profile.ID, kitsu.LibAnimeWatching)
	if err != nil {
		return err
	}
	err = m.db.UpdateAllAnime(anime)
	if err != nil {
		return fmt.Errorf("failed to reload library: %w", err)
	}
	err = app.UpdateFeeds(m.db)
	if err != nil {
		return fmt.Errorf("failed to reload library: %w", err)
	}
	slices.SortFunc(anime, app.CompareAnime)
	return LibraryReloadedMsg{anime, 0}
}

func (m Library_Model) updateFeeds() tea.Msg {
	err := app.UpdateFeeds(m.db)
	if err != nil {
		return err
	}
	anime := m.db.Anime()
	slices.SortFunc(anime, app.CompareAnime)
	return LibraryFeedMsg{m.db.Anime()}
}

func (m Library_Model) deleteAnime() tea.Msg {
	index := m.state.animeIndex
	libID := m.state.anime[index].LibID
	if m.state.searchAnimeResult.Found {
		index = 0
		libID = m.state.searchAnimeResult.Value.LibID
	}
	if err := app.DeleteAnime(m.db, libID); err != nil {
		return err
	}
	anime := m.db.Anime()
	slices.SortFunc(anime, app.CompareAnime)
	return LibraryReloadedMsg{anime, index}
}

func (m Library_Model) dropAnime() tea.Msg {
	index := m.state.animeIndex
	libID := m.state.anime[index].LibID
	if m.state.searchAnimeResult.Found {
		index = 0
		libID = m.state.searchAnimeResult.Value.LibID
	}
	if err := app.DropAnime(m.db, libID); err != nil {
		return err
	}
	anime := m.db.Anime()
	slices.SortFunc(anime, app.CompareAnime)
	return LibraryReloadedMsg{anime, index}
}

func (m Library_Model) completeAnime() tea.Msg {
	index := m.state.animeIndex
	libID := m.state.anime[index].LibID
	if m.state.searchAnimeResult.Found {
		index = 0
		libID = m.state.searchAnimeResult.Value.LibID
	}
	if err := app.CompleteAnime(m.db, libID); err != nil {
		return err
	}
	anime := m.db.Anime()
	slices.SortFunc(anime, app.CompareAnime)
	return LibraryReloadedMsg{anime, index}
}

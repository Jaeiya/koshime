package views

import (
	"fmt"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type WatchList_View int

const (
	WatchList_Menu = WatchList_View(iota)
	WatchList_Reload
	WatchList_Delete
)

type WatchListReloadedMsg = []kitsu.LibraryEntry

type WatchList_Model struct {
	windowSize tea.WindowSizeMsg
	ui         struct {
		loader       ui.LoaderModel
		consent      ui.ConsentModel
		animeDisplay *AnimeDisplayModel
	}
	keys struct {
		reload key.Binding
	}
	db        *database.Database
	menuItems []string
	state     WatchList_State
}

type WatchList_State struct {
	err        error
	view       WatchList_View
	anime      []ui.AnimeInfo
	animeIndex int
	menuIndex  int
}

func newWatchListModel(db *database.Database) WatchList_Model {
	m := WatchList_Model{db: db}
	m.menuItems = []string{
		"Delete",
	}
	m.ui.loader = ui.NewLoader()
	m.ui.animeDisplay = NewAnimeDisplayModel()
	m.keys.reload = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload"))
	m.state.anime = ui.ToAnimeInfo(db.Anime())
	return m
}

func (m WatchList_Model) Init() tea.Cmd {
	return nil
}

func (m WatchList_Model) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.reload):
			// Never execute reload in another view
			if m.state.view > WatchList_Menu {
				break
			}
			m.state.view = WatchList_Reload

		case key.Matches(msg, ui.KeyMap.MainMenu):
			return m, exitToMenu

		}

	case error:
		m.state.err = msg
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case WatchList_Menu:
		m, cmd = m.UpdateMenu(msg)
		cmds = append(cmds, cmd)

	case WatchList_Reload:
		m, cmd = m.UpdateReload(msg)
		cmds = append(cmds, cmd)

	case WatchList_Delete:
		m, cmd = m.UpdateDelete(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m WatchList_Model) View() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.err != nil {
		return ui.DisplayError(m.state.err), nil
	}

	switch m.state.view {
	case WatchList_Reload:
		return m.ViewReload(), nil
	case WatchList_Delete:
		return m.ViewDeleting(), nil
	}

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Watch List"),
		"",
		ui.DisplayText([]string{
			fmt.Sprintf(`There are ;g;%d ;x;anime in your watch list.`, len(m.state.anime)),
		}),
		"",
		m.ui.animeDisplay.View(m.state.anime[m.state.animeIndex]),
		ui.DisplayTitle("Manage"),
		"",
		ui.DisplayText([]string{
			`Select an action to apply to the above entry.`,
		}),
		"",
		ui.DisplayMenuItems(m.menuItems, 0),
	)
	return view, nil
}

func (m WatchList_Model) UpdateMenu(msg tea.Msg) (WatchList_Model, tea.Cmd) {
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

		case key.Matches(msg, ui.KeyMap.Up):
			if m.state.menuIndex == 0 {
				break
			}
			m.state.menuIndex--

		case key.Matches(msg, ui.KeyMap.Down):
			if m.state.menuIndex == len(m.menuItems)-1 {
				break
			}
			m.state.menuIndex++

		case key.Matches(msg, ui.KeyMap.Select):
			if m.state.menuIndex == 0 {
				m.state.view = WatchList_Delete
			}
		}
	}
	return m, nil
}

func (m WatchList_Model) UpdateReload(msg tea.Msg) (WatchList_Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = WatchList_Menu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Reloading Watch List")
			return m, tea.Batch(cmd, m.reloadLibrary)
		}

	case WatchListReloadedMsg:
		m.state = WatchList_State{}
		m.state.anime = ui.ToAnimeInfo(msg)
		m.ui.loader.Stop()

	}
	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m WatchList_Model) ViewReload() string {
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Watch List", "Reloading"),
		"",
		ui.DisplayText(
			[]string{
				`This will ;r;overwrite;x; your current database with updated
information from ;dg;Kitsu;x;. This should only be necessary if ;dg;Kitsu;x; goes
out of sync with ;db;Koshime;x;'s database.`,
				`;dy;Getting ;y;out of sync ;dy;is only likely to happen if you
manually update your Kitsu watch list from the website.`,
			},
			1,
		),
		ui.TextStyle.Render(
			m.ui.consent.View(utils.ColorText(";b;Are you sure you want to reload?")),
		),
	)
	return view
}

func (m WatchList_Model) UpdateDelete(msg tea.Msg) (WatchList_Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = WatchList_Menu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Deleting Entry")
			return m, tea.Batch(cmd, m.deleteEntry)
		}

	case WatchListReloadedMsg:
		m.state = WatchList_State{}
		m.state.anime = ui.ToAnimeInfo(msg)
		m.ui.loader.Stop()
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m WatchList_Model) ViewDeleting() string {
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Watch List", "Deleting Entry"),
		"",
		ui.DisplayText(
			[]string{
				fmt.Sprintf(
					`;dc;%s ;x;will be deleted from your Kitsu library and
the local database.`, m.state.anime[m.state.animeIndex].EngTitle),
				`;y;[this action cannot be undone]`,
			}, 1,
		),
		ui.TextStyle.Render(
			m.ui.consent.View(utils.ColorText(";b;Are you sure?")),
		),
	)
	return view
}

func (m WatchList_Model) ShortHelp() []key.Binding {
	if m.state.view == WatchList_Reload {
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select, ui.KeyMap.EscBack}
	}

	keys := []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down}

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
	keys = append(keys, ui.KeyMap.HelpMore)
	return keys
}

func (m WatchList_Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Prev, ui.KeyMap.Next, ui.KeyMap.Select},
		{m.keys.reload, ui.KeyMap.MainMenu, ui.KeyMap.HelpLess},
	}
}

func (m WatchList_Model) reloadLibrary() tea.Msg {
	profile := m.db.Profile()
	watchList, err := kitsu.GetLibraryAnime(profile.ID, kitsu.LibAnimeWatching)
	if err != nil {
		return err
	}

	err = m.db.LoadLibrary(watchList)
	if err != nil {
		return fmt.Errorf("failed to load data into database library: %w", err)
	}

	return watchList
}

func (m WatchList_Model) deleteEntry() tea.Msg {
	p := m.db.Profile()
	libID := m.state.anime[m.state.animeIndex].LibID
	_, err := kitsu.DeleteLibAnime(libID, p.AccessToken)
	if err != nil {
		return err
	}
	err = m.db.DeleteAnimeById(libID)
	if err != nil {
		return fmt.Errorf("failed to delete anime from database: %w", err)
	}

	return m.db.Anime()
}

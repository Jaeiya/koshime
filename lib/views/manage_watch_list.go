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
	db    *database.Database
	state WatchList_State
}

type WatchList_State struct {
	err         error
	anime       []ui.AnimeInfo
	animeIndex  int
	isReloading bool
}

func newWatchListModel(db *database.Database) WatchList_Model {
	m := WatchList_Model{db: db}
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
	if m.state.isReloading {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.reload):
			m.state.isReloading = true

		case key.Matches(msg, ui.KeyMap.MainMenu):
			return m, exitToMenu

		case key.Matches(msg, ui.KeyMap.Select):
			if m.state.isReloading {
				m.state = WatchList_State{}
				if m.ui.consent.Select() == ui.No {
					return m, nil
				}
				m.ui.loader, cmd = m.ui.loader.Start("Reloading Watch List")
				return m, tea.Batch(cmd, m.reloadLibrary)
			}

		case key.Matches(msg, ui.KeyMap.Left):
			if m.state.animeIndex == 0 {
				return m, nil
			}
			m.state.animeIndex--

		case key.Matches(msg, ui.KeyMap.Right):
			if m.state.animeIndex == len(m.state.anime)-1 {
				return m, nil
			}
			m.state.animeIndex++
		}

	case []kitsu.LibraryEntry:
		m.state.anime = ui.ToAnimeInfo(msg)
		m.ui.loader.Stop()

	case error:
		m.state.err = msg
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
	}

	return m, cmd
}

func (m WatchList_Model) View() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.err != nil {
		return ui.DisplayError(m.state.err), nil
	}

	if m.state.isReloading {
		return m.ViewReloading(), nil
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
	)
	return view, nil
}

func (m WatchList_Model) ViewReloading() string {
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

func (m WatchList_Model) ShortHelp() []key.Binding {
	if m.state.isReloading {
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select, ui.KeyMap.EscBack}
	}
	if len(m.state.anime) > 0 {
		if m.state.animeIndex == len(m.state.anime)-1 {
			return []key.Binding{m.keys.reload, ui.KeyMap.Last, ui.KeyMap.MainMenu}
		}
		if m.state.animeIndex > 0 {
			return []key.Binding{
				m.keys.reload,
				ui.KeyMap.Last,
				ui.KeyMap.Next,
				ui.KeyMap.MainMenu,
			}
		}
	}

	return []key.Binding{m.keys.reload, ui.KeyMap.Next, ui.KeyMap.MainMenu}
}

func (m WatchList_Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
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

package views

import (
	"fmt"

	"github.com/Jaeiya/koshime/lib"
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type WatchList_View int

const (
	WatchList_Menu = WatchList_View(iota)
	WatchList_Reload
	WatchList_Drop
	WatchList_FileBinding
	WatchList_Complete
	WatchList_Delete
)

type BindingMode int

const (
	SelectFile = BindingMode(iota)
	SelectAnime
	ConfirmSelection
)

type WatchListReloadedMsg = []kitsu.LibraryEntry

type WatchList_Model struct {
	windowSize tea.WindowSizeMsg
	ui         struct {
		loader       ui.LoaderModel
		consent      ui.ConsentModel
		animeDisplay *AnimeDisplayModel
		menu         ui.MenuModel
		list         list.Model
	}
	keys struct {
		reload key.Binding
	}
	db    *database.Database
	state WatchList_State
}

type WatchList_State struct {
	err               error
	view              WatchList_View
	anime             []ui.AnimeInfo
	animeIndex        int
	selectedFileTitle string
	selectedAnime     ui.AnimeInfo
	bindingMode       BindingMode
}

func newWatchListModel(db *database.Database) WatchList_Model {
	m := WatchList_Model{db: db}
	m.ui.loader = ui.NewLoader()
	m.ui.animeDisplay = NewAnimeDisplayModel()
	m.ui.menu = ui.NewMenuModel([]string{
		"Drop",
		"Bind",
		"Complete",
		"Delete",
	}, ui.WithMenuRotation(), ui.WithMenuDescriptions([]string{
		`Drops the selected anime above.`,
		`Binds a file name to a specific anime in your watch list.`,
		`Sets status of selected anime above, to completed.`,
		`Deletes the selected anime above.`,
	}))
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
			if m.ui.list.FilterState() > list.Unfiltered {
				break
			}
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

	case WatchList_Drop:
		m, cmd = m.UpdateDrop(msg)
		cmds = append(cmds, cmd)

	case WatchList_FileBinding:
		m, cmd = m.UpdateFileBinding(msg)
		cmds = append(cmds, cmd)

	case WatchList_Complete:
		m, cmd = m.UpdateCompleted(msg)
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
	case WatchList_Menu:
		return m.ViewMenu(), nil
	case WatchList_Reload:
		return m.ViewReload(), nil
	case WatchList_Drop:
		return m.ViewDrop(), nil
	case WatchList_Delete:
		return m.ViewDeleting(), nil
	case WatchList_FileBinding:
		return m.ViewFileBinding(), nil
	case WatchList_Complete:
		return m.ViewCompleted(), nil
	default:
		return "unknown view", nil
	}
}

func (m WatchList_Model) ShortHelp() []key.Binding {
	keys := []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down}

	if m.state.bindingMode < ConfirmSelection {
		return nil
	}

	if m.state.view > WatchList_Menu {
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select, ui.KeyMap.EscBack}
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
	keys = append(keys, ui.KeyMap.HelpMore)
	return keys
}

func (m WatchList_Model) FullHelp() [][]key.Binding {
	// Prevent conflicts with list component
	if m.state.view == WatchList_FileBinding {
		return nil
	}
	return [][]key.Binding{
		{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Prev, ui.KeyMap.Next, ui.KeyMap.Select},
		{m.keys.reload, ui.KeyMap.MainMenu, ui.KeyMap.HelpLess},
	}
}

func (m WatchList_Model) UpdateMenu(msg tea.Msg) (WatchList_Model, tea.Cmd) {
	var cmd tea.Cmd

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
		case 0:
			m.state.view = WatchList_Drop
		case 1:
			m.state.view = WatchList_FileBinding
		case 2:
			m.state.view = WatchList_Complete
		case 3:
			m.state.view = WatchList_Delete
		}
	}

	m.ui.menu, cmd = m.ui.menu.Update(msg)
	return m, cmd
}

func (m WatchList_Model) ViewMenu() string {
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
		m.ui.menu.View(),
	)
	return view
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

func (m WatchList_Model) UpdateDrop(msg tea.Msg) (WatchList_Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = WatchList_Menu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Dropping Anime")
			return m, tea.Batch(cmd, m.dropAnime)
		}

	case WatchListReloadedMsg:
		m.state = WatchList_State{}
		m.state.anime = ui.ToAnimeInfo(msg)
		m.ui.loader.Stop()
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m WatchList_Model) ViewDrop() string {
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Watch List", "Drop"),
		"",
		ui.DisplayText([]string{
			fmt.Sprintf(
				`;dc;%s ;x;is about to be ;m;dropped ;x;from your watch list.`,
				m.state.anime[m.state.animeIndex].EngTitle,
			),
			`This is not the same as deletion. Dropping an anime sets its status to
;dm;dropped;x;, which stores it under the dropped tab in ;db;Kitsu;x;.`,
			`Dropping an anime is often times better than deletion because it keeps
track of the episodes you ;w;did;x; watch. This makes your watch time stats more
accurate.`,
		}, 1),
		ui.TextStyle.Render(m.ui.consent.View(utils.ColorText(";b;Are you sure?"))),
	)
	return view
}

func (m WatchList_Model) UpdateFileBinding(msg tea.Msg) (WatchList_Model, tea.Cmd) {
	var cmd tea.Cmd

	if len(m.ui.list.VisibleItems()) == 0 {
		var ff lib.FansubFilter
		fileStream, err := fileSys.NewFilenameStream(fileSys.GetWorkingDir())
		if err != nil {
			m.state.err = err
			return m, nil
		}

		fansubs, err := ff.All(fileStream)
		if err != nil {
			m.state.err = err
			return m, nil
		}

		listItems := make([]list.Item, len(fansubs))
		for i, item := range fansubs {
			listItems[i] = ui.NewListItem(item.Title, item.Filename, i)
		}

		m.ui.list = ui.NewList(ui.ListOptions{
			Items:        listItems,
			Width:        m.windowSize.Width - 5,
			MaxHeight:    m.windowSize.Height,
			ItemsPerPage: 5,
			EnableFilter: true,
		})
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Submit):
			if m.ui.list.FilterState() == list.Filtering {
				break
			}

			switch m.state.bindingMode {
			case SelectFile:
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.selectedFileTitle = item.Title()
				listItems := make([]list.Item, len(m.state.anime))
				for i, item := range m.state.anime {
					if item.EngTitle == "" {
						item.EngTitle = item.JpnTitle
						item.JpnTitle = ""
					}
					listItems[i] = ui.NewListItem(item.EngTitle, item.JpnTitle, i)
				}

				m.ui.list = ui.NewList(ui.ListOptions{
					Items:        listItems,
					Width:        m.windowSize.Width - 5,
					MaxHeight:    m.windowSize.Height,
					ItemsPerPage: 5,
					EnableFilter: true,
				})
				m.state.bindingMode = SelectAnime

			case SelectAnime:
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.selectedAnime = m.state.anime[item.Index()]
				m.state.bindingMode = ConfirmSelection

			case ConfirmSelection:
				if m.ui.consent.Select() == ui.No {
					m.ui.list = list.Model{}
					m.state.bindingMode = SelectFile
					m.state.view = WatchList_Menu
					return m, nil
				}

				m.db.AddFileBinding(m.state.selectedFileTitle, m.state.selectedAnime.LibID)
				m.ui.list = list.Model{}
				m.state.bindingMode = SelectFile
				m.state.view = WatchList_Menu
				return m, nil
			}
		}
	}

	if m.state.bindingMode == ConfirmSelection {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	m.ui.list, cmd = m.ui.list.Update(msg)
	return m, cmd
}

func (m WatchList_Model) ViewFileBinding() string {
	switch m.state.bindingMode {
	case SelectFile:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("File Binding"),
			ui.DisplayText([]string{";m;Select a file you want to bind"}, 0, 1),
			m.ui.list.View(),
		)

	case SelectAnime:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("File Binding"),
			ui.DisplayText([]string{";m;Select the anime you'd like to bind"}, 0, 1),
			m.ui.list.View(),
		)

	case ConfirmSelection:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("File Binding"),
			ui.DisplayText([]string{
				"Selected File: ;y;" + m.state.selectedFileTitle,
				"   Binding To: ;g;" + m.state.selectedAnime.JpnTitle + ";x;",
			}, 0, 1, 1),
			ui.TextStyle.Render(
				m.ui.consent.View(utils.ColorText(";b;Is the above correct?")),
			),
		)
	}

	return "missing file binding view"
}

func (m WatchList_Model) UpdateCompleted(msg tea.Msg) (WatchList_Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = WatchList_Menu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Completing Anime")
			return m, tea.Batch(cmd, m.completeAnime)
		}

	case WatchListReloadedMsg:
		m.state = WatchList_State{}
		m.state.anime = ui.ToAnimeInfo(msg)
		m.ui.loader.Stop()
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m WatchList_Model) ViewCompleted() string {
	animeTitle := m.state.anime[m.state.animeIndex].EngTitle
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Watch List", "Complete"),
		"",
		ui.DisplayText([]string{
			fmt.Sprintf(`;dc;%s ;x;is about to be ;b;completed;x;.`, animeTitle),
			`This should only be necessary if ;dg;Kitsu;x; messes up episode counts.
Updating the progress of an anime will auto-complete it on the last episode of a season.`,
		}, 1),
		ui.TextStyle.Render(m.ui.consent.View(utils.ColorText(";b;Are you sure?"))),
	)
	return view
}

func (m WatchList_Model) reloadLibrary() tea.Msg {
	profile := m.db.Profile()
	watchList, err := kitsu.GetUserAnime(profile.ID, kitsu.LibAnimeWatching)
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
	_, err := kitsu.DeleteAnime(libID, p.AccessToken)
	if err != nil {
		return err
	}
	err = m.db.DeleteAnimeById(libID)
	if err != nil {
		return fmt.Errorf("failed to delete anime from database: %w", err)
	}

	return m.db.Anime()
}

func (m WatchList_Model) dropAnime() tea.Msg {
	p := m.db.Profile()
	libID := m.state.anime[m.state.animeIndex].LibID
	if err := kitsu.DropAnime(libID, p.AccessToken); err != nil {
		return err
	}

	if err := m.db.DeleteAnimeById(libID); err != nil {
		return fmt.Errorf("failed to delete anime from database: %w", err)
	}

	return m.db.Anime()
}

func (m WatchList_Model) completeAnime() tea.Msg {
	p := m.db.Profile()
	libID := m.state.anime[m.state.animeIndex].LibID
	_, err := kitsu.SetAnimeStatus(libID, p.AccessToken, kitsu.LibAnimeCompleted)
	if err != nil {
		return err
	}

	if err := m.db.DeleteAnimeById(libID); err != nil {
		return fmt.Errorf("failed to delete anime from database: %w", err)
	}

	return m.db.Anime()
}

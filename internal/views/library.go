package views

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
)

type Library_View int

const (
	Library_Menu = Library_View(iota)
	Library_Reload
	Library_Drop
	Library_FileBinding
	Library_Complete
	Library_Delete
)

type BindingMode int

const (
	SelectFile = BindingMode(iota)
	SelectAnime
	ConfirmSelection
)

type LibraryReloadedMsg = []kitsu.Anime

type Library_Model struct {
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
	state Library_State
}

type Library_State struct {
	err               error
	view              Library_View
	anime             []kitsu.Anime
	animeIndex        int
	selectedFileTitle string
	selectedAnime     kitsu.Anime
	filesNotFound     bool
	bindingMode       BindingMode
}

func newLibraryModel(db *database.Database) Library_Model {
	m := Library_Model{db: db}
	m.ui.list = ui.NewList(ui.ListOptions{})
	m.ui.loader = ui.NewLoader()
	m.ui.animeDisplay = NewAnimeDisplayModel()
	m.ui.menu = ui.NewMenuModel([]string{
		"Drop",
		"Bind",
		"Complete",
		"Delete",
	}, ui.WithMenuRotation(), ui.WithMenuDescriptions([]string{
		`Drops the selected anime above and removes it from local database.`,
		`Binds a file name to a specific anime in your library.`,
		`Sets status of selected anime above, to completed.`,
		`Deletes the selected anime above from Kitsu and local database.`,
	}))
	m.keys.reload = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload"))
	m.state.anime = db.Anime()
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
		case key.Matches(msg, m.keys.reload):
			// Never execute reload in another view
			if m.state.view > Library_Menu {
				break
			}
			m.state.view = Library_Reload

		case key.Matches(msg, ui.KeyMap.MainMenu):
			if m.state.filesNotFound {
				break
			}
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
	case Library_Menu:
		m, cmd = m.UpdateMenu(msg)
		cmds = append(cmds, cmd)

	case Library_Reload:
		m, cmd = m.UpdateReload(msg)
		cmds = append(cmds, cmd)

	case Library_Delete:
		m, cmd = m.UpdateDelete(msg)
		cmds = append(cmds, cmd)

	case Library_Drop:
		m, cmd = m.UpdateDrop(msg)
		cmds = append(cmds, cmd)

	case Library_FileBinding:
		m, cmd = m.UpdateFileBinding(msg)
		cmds = append(cmds, cmd)

	case Library_Complete:
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
	case Library_Menu:
		return m.ViewMenu()
	case Library_Reload:
		return m.ViewReload()
	case Library_Drop:
		return m.ViewDrop()
	case Library_Delete:
		return m.ViewDeleting()
	case Library_FileBinding:
		return m.ViewFileBinding()
	case Library_Complete:
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

	// List has its own keymap help
	if m.state.view == Library_FileBinding {
		return nil
	}

	if m.state.view > Library_Menu {
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
	keys = append(keys, ui.KeyMap.HelpMore)
	return keys
}

func (m Library_Model) FullHelp() [][]key.Binding {
	// Prevent conflicts with list component
	if m.state.view == Library_FileBinding {
		return nil
	}
	return [][]key.Binding{
		{
			ui.KeyMap.Up,
			ui.KeyMap.Down,
			ui.KeyMap.Prev,
			ui.KeyMap.Next,
			m.ui.animeDisplay.ShortHelp()[0],
			ui.KeyMap.Select,
		},
		{m.keys.reload, ui.KeyMap.MainMenu, ui.KeyMap.HelpLess},
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
		case 0:
			m.state.view = Library_Drop
		case 1:
			m.state.view = Library_FileBinding
			m, cmd = m.UpdateFileBinding(nil)
			cmds = append(cmds, cmd)
		case 2:
			m.state.view = Library_Complete
		case 3:
			m.state.view = Library_Delete
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

func (m Library_Model) UpdateReload(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = Library_Menu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Reloading Library")
			return m, tea.Batch(cmd, m.reloadLibrary)
		}

	case LibraryReloadedMsg:
		m.state = Library_State{}
		m.state.anime = msg
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
			1,
		),
		ui.TextStyle.Render(
			m.ui.consent.View(utils.ColorText(";b;Are you sure you want to reload?")),
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
				m.state.view = Library_Menu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Deleting Entry")
			return m, tea.Batch(cmd, m.deleteAnime)
		}

	case LibraryReloadedMsg:
		m.state = Library_State{}
		m.state.anime = msg
		m.ui.loader.Stop()
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m Library_Model) ViewDeleting() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Library", "Deleting Entry"),
		"",
		ui.DisplayText(
			[]string{
				fmt.Sprintf(
					`;dc;%s ;x;will be deleted from your Kitsu library and
the local database.`, m.state.anime[m.state.animeIndex].ENG_Title,
				),
				`;y;[this action cannot be undone]`,
			}, 1,
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
				m.state.view = Library_Menu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Dropping Anime")
			return m, tea.Batch(cmd, m.dropAnime)
		}

	case LibraryReloadedMsg:
		m.state = Library_State{}
		m.state.anime = msg
		m.ui.loader.Stop()
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m Library_Model) ViewDrop() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Library", "Drop"),
		"",
		ui.DisplayText([]string{
			fmt.Sprintf(
				`;dc;%s ;x;is about to be ;m;dropped;x;.`,
				m.state.anime[m.state.animeIndex].ENG_Title,
			),
			`This is not the same as deletion. Dropping an anime sets its status to
;dm;dropped;x;, which stores it under the dropped tab in ;db;Kitsu;x;.`,
			`Dropping an anime is often times better than deletion because it keeps
track of the episodes you ;w;did;x; watch. This makes your watch time stats more
accurate.`,
		}, 1),
		ui.TextStyle.Render(m.ui.consent.View(utils.ColorText(";b;Are you sure?"))),
	))
}

func (m Library_Model) UpdateFileBinding(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd

	if len(m.ui.list.VisibleItems()) == 0 && !m.state.filesNotFound {
		var ff app.FansubFilter
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

		if len(fansubs) > 0 {
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
		} else {
			m.state.filesNotFound = true
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.state.filesNotFound {
				m.state.filesNotFound = false
				m.state.view = Library_Menu
				return m, nil
			}

		case key.Matches(msg, ui.KeyMap.Submit):
			if m.ui.list.FilterState() == list.Filtering {
				break
			}

			switch m.state.bindingMode {
			case SelectFile:
				if m.state.filesNotFound {
					return m, nil
				}
				//nolint:errcheck // will ALWAYS be a list
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.selectedFileTitle = item.Title()
				listItems := make([]list.Item, len(m.state.anime))
				for i, item := range m.state.anime {
					if item.ENG_Title == "" {
						item.ENG_Title = item.JPN_Title
						item.JPN_Title = ""
					}
					listItems[i] = ui.NewListItem(item.ENG_Title, item.JPN_Title, i)
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
				//nolint:errcheck // will ALWAYS be a list
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.selectedAnime = m.state.anime[item.Index()]
				m.state.bindingMode = ConfirmSelection

			case ConfirmSelection:
				if m.ui.consent.Select() == ui.No {
					m.ui.list = list.Model{}
					m.state.bindingMode = SelectFile
					m.state.view = Library_Menu
					return m, nil
				}

				err := m.db.AddFileBinding(m.state.selectedFileTitle, m.state.selectedAnime.LibID)
				if err != nil {
					return m, func() tea.Msg { return DefaultErrorMsg{err} }
				}
				m.ui.list = list.Model{}
				m.state.bindingMode = SelectFile
				m.state.view = Library_Menu
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

func (m Library_Model) ViewFileBinding() tea.View {
	switch m.state.bindingMode {
	case SelectFile:
		if m.state.filesNotFound {
			return tea.NewView(lipgloss.JoinVertical(
				lipgloss.Left,
				ui.DisplayTitle("File Binding"),
				ui.DisplayText([]string{";y;No files found for binding"}, 0, 1),
			))
		}
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("File Binding"),
			ui.DisplayText([]string{";m;Select a file you want to bind"}, 0, 1),
			m.ui.list.View(),
		))

	case SelectAnime:
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("File Binding"),
			ui.DisplayText([]string{";m;Select the anime you'd like to bind"}, 0, 1),
			m.ui.list.View(),
		))

	case ConfirmSelection:
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("File Binding"),
			ui.DisplayText([]string{
				"Selected File: ;y;" + m.state.selectedFileTitle,
				"   Binding To: ;g;" + m.state.selectedAnime.JPN_Title + ";x;",
			}, 0, 1, 1),
			ui.TextStyle.Render(
				m.ui.consent.View(utils.ColorText(";b;Is the above correct?")),
			),
		))
	}

	return tea.NewView("missing FileBinding view")
}

func (m Library_Model) UpdateCompleted(msg tea.Msg) (Library_Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.state.view = Library_Menu
				return m, nil
			}
			m.ui.loader, cmd = m.ui.loader.Start("Completing Anime")
			return m, tea.Batch(cmd, m.completeAnime)
		}

	case LibraryReloadedMsg:
		m.state = Library_State{}
		m.state.anime = msg
		m.ui.loader.Stop()
	}

	m.ui.consent = m.ui.consent.Update(msg)
	return m, nil
}

func (m Library_Model) ViewCompleted() tea.View {
	animeTitle := m.state.anime[m.state.animeIndex].ENG_Title
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Library", "Complete"),
		"",
		ui.DisplayText([]string{
			fmt.Sprintf(`;dc;%s ;x;is about to be ;b;completed;x;.`, animeTitle),
			`This should only be necessary if ;dg;Kitsu;x; messes up episode counts.
Updating the progress of an anime will auto-complete it on the last episode of a season.`,
		}, 1),
		ui.TextStyle.Render(m.ui.consent.View(utils.ColorText(";b;Are you sure?"))),
	))
}

func (m Library_Model) reloadLibrary() tea.Msg {
	profile := m.db.Profile()
	anime, err := kitsu.GetUserAnime(profile.ID, kitsu.LibAnimeWatching)
	if err != nil {
		return err
	}
	err = m.db.LoadLibrary(anime)
	if err != nil {
		return fmt.Errorf("failed to load data into database library: %w", err)
	}
	return anime
}

func (m Library_Model) deleteAnime() tea.Msg {
	libID := m.state.anime[m.state.animeIndex].LibID
	if err := app.DeleteAnime(m.db, libID); err != nil {
		return err
	}
	return m.db.Anime()
}

func (m Library_Model) dropAnime() tea.Msg {
	libID := m.state.anime[m.state.animeIndex].LibID
	if err := app.DropAnime(m.db, libID); err != nil {
		return err
	}
	return m.db.Anime()
}

func (m Library_Model) completeAnime() tea.Msg {
	libID := m.state.anime[m.state.animeIndex].LibID
	if err := app.CompleteAnime(m.db, libID); err != nil {
		return err
	}
	return m.db.Anime()
}

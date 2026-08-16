package views

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
)

type UIView int

const (
	SetupUser = UIView(iota)
	Menu
	Abort
	Exit
)

var fileSys = utils.FileSys{}

type FetchErrorMsg struct {
	Msg string
}

func (e FetchErrorMsg) Error() string {
	return e.Msg
}

type DefaultErrorMsg struct {
	err error
}

func (e DefaultErrorMsg) Error() string {
	return e.err.Error()
}

type ViewModel interface {
	help.KeyMap
	Update(msg tea.Msg) (ViewModel, tea.Cmd)
	View() tea.View
	Init() tea.Cmd
}

type (
	AbortMsg struct{}
	ExitMsg  struct{}
)

type Model struct {
	db         *database.Database
	windowSize tea.WindowSizeMsg
	menu       MenuModel
	help       help.Model
	setupUser  SetupUserModel
	view       UIView
}

func New() (Model, error) {
	m := Model{}

	m.help = help.New()
	m.help.Styles.ShortKey = ui.HelpKeyStyle
	m.help.Styles.FullKey = m.help.Styles.ShortKey
	m.help.Styles.ShortDesc = ui.HelpDescStyle
	m.help.Styles.FullDesc = m.help.Styles.ShortDesc

	// Initialize empty database
	db, _ := database.NewDatabase(nil)
	m.db = db

	if !db.Exists() {
		m.setupUser = newSetupUserModel()
		return m, nil
	}

	err := db.Load()
	if err != nil {
		return m, err
	}

	m.CreateMenu()
	m.view = Menu
	return m, nil
}

func (ui Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Send size to all menu view models
		m.windowSize = msg
		m.menu, _ = m.menu.Update(m.windowSize)

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.view = Abort
			return m, tea.Quit
		}

	case SetupUserFinishedMsg:
		err := m.db.LoadData(msg.Value)
		if err != nil {
			// This should never happen
			panic(fmt.Errorf("failed to load new user data: %w", err))
		}
		m.CreateMenu()
		m.menu, _ = m.menu.Update(m.windowSize)
		m.view = Menu

	case AbortMsg:
		m.view = Abort
		return m, tea.Quit

	case ExitMsg:
		m.view = Exit
		return m, tea.Quit

	}

	if m.view == SetupUser {
		m.setupUser, cmd = m.setupUser.Update(msg)
		return m, cmd
	}

	m.menu, cmd = m.menu.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	switch m.view {
	case Menu:
		return m.menu.View()

	case SetupUser:
		v := m.setupUser.View()
		v.Content = lipgloss.JoinVertical(
			lipgloss.Left,
			v.Content,
			ui.HelpStyle.Render(m.help.View(m.setupUser)),
		)
		return v

	case Exit:
		return tea.NewView("")

	case Abort:
		return tea.NewView(ui.AbortStyle.Render(
			utils.ColorText(";g;>>> ;y;User Aborted Operation ;g;<<<"),
		))

	default:
		return tea.NewView("missing view")

	}
}

func (m *Model) CreateMenu() {
	m.menu = NewMenuModel([]MenuView{
		{
			Name:      "Watch",
			ModelFunc: func() ViewModel { return newWatchModel(m.db) },
			Desc:      "Finds downloaded anime and coordinates with your watch list to execute the file.",
		},
		{
			Name:      "Add",
			ModelFunc: func() ViewModel { return newAddAnimeModel(m.db) },
			Desc:      "Add an airing or completed anime to your watch list.",
		},
		{
			Name:      "RSS Lookup",
			ModelFunc: func() ViewModel { return newRssModel(m.db) },
			Desc:      "Search for fansub feeds",
		},
		{
			Name:      "Library",
			ModelFunc: func() ViewModel { return newLibraryModel(m.db) },
			Desc:      "View & manage your local library.",
		},
		{
			Name:      "Find",
			ModelFunc: func() ViewModel { return newFindAnimeModel(m.db) },
			Desc:      "Lookup an anime from Kitsu or your local watch list.",
		},
		{Name: "Maintenance", SubViews: []MenuView{
			{
				Name:      "Token",
				ModelFunc: func() ViewModel { return newTokenModel(m.db) },
				Desc:      "Refresh, reset, or view your Kitsu access token.",
			},
			{
				Name:      "Clean",
				ModelFunc: func() ViewModel { return newWatchDirModel() },
				Desc:      "View & manage your watched anime files.",
			},
		}, Desc: "Submenu for managing Koshime functionality."},
		{
			Name:      "About",
			ModelFunc: func() ViewModel { return newAboutModel() },
			Desc:      "View the nitty-gritty details of Koshime",
		},
	}, m.db)
}

func abort() tea.Msg {
	return AbortMsg{}
}

func exit() tea.Msg {
	return ExitMsg{}
}

package views

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/logger"
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

type FatalErrorMsg struct {
	Msg  string
	Desc string
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
	FatalErr   FatalErrorMsg
	HasAborted bool
}

func New() (Model, error) {
	m := Model{}
	var err error

	m.help = help.New()
	m.help.Styles.ShortKey = ui.HelpKeyStyle
	m.help.Styles.FullKey = m.help.Styles.ShortKey
	m.help.Styles.ShortDesc = ui.HelpDescStyle
	m.help.Styles.FullDesc = m.help.Styles.ShortDesc

	logger.Log(logger.Info, "loading database")
	m.db, err = database.NewDatabase(nil)
	if err != nil {
		return m, fmt.Errorf("failed to initialize database: %w", err)
	}

	if !m.db.Exists() {
		logger.Log(logger.Info, "Database not found: begin user setup")
		m.setupUser = newSetupUserModel()
		return m, nil
	}

	m.CreateMenu()
	m.view = Menu
	return m, nil
}

func (m Model) Init() tea.Cmd {
	if m.db.Exists() {
		logger.Log(logger.Debug, "Init(): initializing menu")
		return m.menu.Init()
	}
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

	case FatalErrorMsg:
		logger.Log(logger.Debug, "Update(): received fatal error message")
		m.FatalErr = msg
		return m, abort

	case tea.KeyPressMsg:
		// This is typically a forced abort so we assume the user
		// wanted to abort, rather than just exit.
		if msg.String() == "ctrl+c" {
			logger.Log(logger.Debug, "Update(): user aborted application with ctrl+c")
			m.view = Abort
			return m, abort
		}

		// Allows easy exit from any view
		if msg.String() == "ctrl+x" {
			logger.Log(logger.Debug, "Update(): user exited application using ctrl+x")
			m.view = Exit
			return m, exit
		}

	case SetupUserFinishedMsg:
		logger.Log(logger.Debug, "Update(): finishing user setup")
		if err := m.db.Overwrite(msg.Value); err != nil {
			return m, sendFatal(
				fmt.Errorf("failed to load new user data: %w", err).Error(),
				"Failed to save data to database after user setup",
			)
		}
		m.CreateMenu()
		m.menu, _ = m.menu.Update(m.windowSize)
		m.view = Menu
		return m, m.menu.Init()

	case AbortMsg:
		logger.Log(logger.Debug, "Update(): aborting application")
		m.HasAborted = true
		m.view = Abort
		return m, tea.Quit

	case ExitMsg:
		logger.Log(logger.Debug, "Update(): exiting application")
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

	case Abort, Exit:
		return tea.NewView("")

	default:
		return tea.NewView("missing view")

	}
}

func (m *Model) CreateMenu() {
	logger.Log(logger.Info, "creating main menu")
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
			ModelFunc: func() ViewModel { return newRssMainModel(m.db) },
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

func sendFatal(errMsg, desc string) tea.Cmd {
	logger.Log(logger.Debug, "sending fatal error message")
	return func() tea.Msg {
		return FatalErrorMsg{
			Msg:  errMsg,
			Desc: desc,
		}
	}
}

func abort() tea.Msg {
	return AbortMsg{}
}

func exit() tea.Msg {
	return ExitMsg{}
}

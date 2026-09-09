package views

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/logger"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
)

type UIView int

const (
	SetupUser = UIView(iota)
	Menu
	TokenExpiring
	Abort
	Exit
)

var fileSys = utils.FileSys{}

type FetchErrorMsg struct {
	Value string
}

func (e FetchErrorMsg) Error() string {
	return e.Value
}

type DefaultErrorMsg struct {
	err error
}

func (e DefaultErrorMsg) Error() string {
	return e.err.Error()
}

type FatalErrorMsg struct {
	Msg  string
	Desc string
}

type viewRefreshTokenMsg struct {
	err error
}

type ViewModel interface {
	help.KeyMap
	Update(msg tea.Msg) (ViewModel, tea.Cmd)
	View() tea.View
	Init() tea.Cmd
}

type ViewErrModel struct{}

func (m ViewErrModel) ShortHelp() []key.Binding {
	return []key.Binding{ui.KeyMap.EscBack}
}

func (m ViewErrModel) FullHelp() [][]key.Binding {
	return nil
}

type (
	AbortMsg struct{}
	ExitMsg  struct{}
)

type Model struct {
	db         *database.Database
	windowSize tea.WindowSizeMsg
	loader     ui.LoaderModel
	consent    ui.ConsentModel
	menu       MenuModel
	help       help.Model
	setupUser  SetupUserModel
	view       UIView
	FatalErr   FatalErrorMsg
	err        error
	HasAborted bool
}

func New() (Model, error) {
	m := Model{}
	var err error
	m.loader = ui.NewLoader()

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

	secPerDay := 86400.0
	days := float64(m.db.Profile().TokenExpirationSec-time.Now().Unix()) / secPerDay
	logger.Log(logger.Debug, "NewView(): token expires in: %0.2f days", days)

	m.view = Menu
	if days < 7 {
		m.view = TokenExpiring
	}

	m.CreateMenu()
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
		switch {
		// This is typically a forced abort so we assume the user
		// wanted to abort, rather than just exit.
		case key.Matches(msg, ui.KeyMap.AbortApp):
			logger.Log(logger.Debug, "Update(): user aborted application with ctrl+c")
			m.view = Abort
			return m, abort

		// Allows easy exit from any view
		case key.Matches(msg, ui.KeyMap.ExitApp):
			logger.Log(logger.Debug, "Update(): user exited application using ctrl+x")
			m.view = Exit
			return m, exit

		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.err != nil {
				m.view = Menu
				m.err = nil
				return m, nil
			}

		// Manage token expiration consent
		case key.Matches(msg, ui.KeyMap.Select):
			if m.loader.IsLoading() {
				return m, nil
			}

			if m.view == TokenExpiring {
				if m.consent.Select() == ui.No {
					m.view = Menu
					return m, nil
				}
				m.loader, cmd = m.loader.Start("Refreshing Token")
				return m, tea.Batch(cmd, m.refreshToken())
			}

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

	case viewRefreshTokenMsg:
		m.loader.Stop()
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.view = Menu

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

	if m.loader.IsLoading() {
		m.loader, cmd = m.loader.Update(msg)
		return m, cmd
	}

	if m.view == SetupUser {
		m.setupUser, cmd = m.setupUser.Update(msg)
		return m, cmd
	}

	if m.view == TokenExpiring {
		m.consent = m.consent.Update(msg)
		return m, nil
	}

	m.menu, cmd = m.menu.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	if m.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.loader.View()))
	}

	if m.err != nil {
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("Error Updating Token"),
			ui.DisplayError(m.err),
			ui.HelpStyle.Render(m.help.View(ViewErrModel{})),
		))
	}

	switch m.view {
	case Menu:
		return m.menu.View()

	case TokenExpiring:
		return m.ViewExpiration()

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

func (m Model) ViewExpiration() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		m.consent.View(
			ui.ConsentStyle.Render(
				"Your Kitsu token is about to expire, would you like to refresh it now?",
			),
		),
		ui.HelpStyle.Render(m.help.View(m.consent)),
	))
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
				ModelFunc: func() ViewModel { return newManDirModel() },
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

func (m Model) refreshToken() tea.Cmd {
	return func() tea.Msg {
		logger.Log(logger.Debug, "refreshing token")
		data, err := kitsu.RefreshToken(m.db.Profile().RefreshToken)
		if err != nil {
			logger.Log(logger.Error, err.Error())
			return viewRefreshTokenMsg{err}
		}
		// TODO: this should just take the AuthTokenData struct as an arg
		logger.Log(logger.Debug, "saving token data to database")
		if err = m.db.SaveTokenData(data.Token, data.RefreshToken, data.ExpiresIn); err != nil {
			logger.Log(logger.Error, err.Error())
			return viewRefreshTokenMsg{err}
		}
		return viewRefreshTokenMsg{}
	}
}

func abort() tea.Msg {
	return AbortMsg{}
}

func exit() tea.Msg {
	return ExitMsg{}
}

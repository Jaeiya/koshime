package views

import (
	"fmt"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/help"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type UIView int

const (
	SetupUser = UIView(iota)
	Menu
	Abort
	Exit
)

var fileSys = utils.FileSys{}

type (
	FetchErrorMsg error
)

type ViewModel interface {
	tea.CursorModel
	help.KeyMap
	Update(msg tea.Msg) (ViewModel, tea.Cmd)
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

func New(dbPath string) (Model, error) {
	m := Model{}

	m.help = help.New()
	m.help.Styles.ShortKey = ui.HelpKeyStyle
	m.help.Styles.FullKey = m.help.Styles.ShortKey
	m.help.Styles.ShortDesc = ui.HelpDescStyle
	m.help.Styles.FullDesc = m.help.Styles.ShortDesc

	// Initialize empty database
	db, _ := database.NewDatabase(nil)
	m.db = db
	fs := utils.FileSys{}

	if !fs.FileExists(dbPath) {
		m.setupUser = newSetupUserModel()
		return m, nil
	}

	err := db.Load()
	if err != nil {
		return m, err
	}

	m.CreateMenuItems()
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
		err := m.db.LoadData(msg)
		if err != nil {
			// This should never happen
			panic(fmt.Errorf("failed to load new user data: %w", err))
		}
		m.CreateMenuItems()
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

func (m Model) View() (string, *tea.Cursor) {
	switch m.view {
	case Menu:
		return m.menu.View()

	case SetupUser:
		v, c := m.setupUser.View()
		return lipgloss.JoinVertical(
			lipgloss.Left,
			v,
			ui.HelpStyle.Render(m.help.View(m.setupUser)),
		), c

	case Exit:
		return "", nil

	case Abort:
		return ui.AbortStyle.Render(
			utils.ColorText(";g;>>> ;y;User Aborted Operation ;g;<<<"),
		), nil

	default:
		return "missing view", nil

	}
}

func (m *Model) CreateMenuItems() {
	m.menu = NewMenuModel([]MenuView{
		{
			Name:      "Watch",
			ModelFunc: func() ViewModel { return newWatchAnimeModel(m.db) },
			Desc:      "Finds downloaded anime and coordinates with your watch list to execute the file.",
		},
		{
			Name:      "Find",
			ModelFunc: func() ViewModel { return newFindAnimeModel(m.db) },
			Desc:      "Lookup an anime from Kitsu or your local watch list.",
		},
		{
			Name:      "Add",
			ModelFunc: func() ViewModel { return newAddAnimeModel(m.db) },
			Desc:      "Add an airing or completed anime to your watch list.",
		},
		{
			Name:      "Delete",
			ModelFunc: func() ViewModel { return newDelAnimeModel(m.db) },
			Desc:      "Delete an anime from your watch list.",
		},
		{Name: "Maintenance", SubViews: []MenuView{
			{
				Name:      "Token",
				ModelFunc: func() ViewModel { return newTokenModel(m.db) },
				Desc:      "Refresh, reset, or view your Kitsu access token.",
			},
			{
				Name:      "Watch List",
				ModelFunc: func() ViewModel { return newManageWatchModel(m.db) },
				Desc:      "View or Reload your entire watch list.",
			},
			// {
			// 	Name: "Clean",
			// 	Desc: "View & manage your watched anime files.",
			// },
		}, Desc: "Submenu for managing Koshime functionality."},
		{
			Name:      "About",
			ModelFunc: func() ViewModel { return newAboutModel() },
			Desc:      "View the nitty-gritty details of Koshime",
		},
	}, m.db.Profile())
}

func abort() tea.Msg {
	return AbortMsg{}
}

func exit() tea.Msg {
	return ExitMsg{}
}

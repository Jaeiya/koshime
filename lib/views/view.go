package views

import (
	"fmt"
	"time"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type UIView int

const (
	None = UIView(iota)
	SetupUserView
	MenuView
	ExitView
	WatchAnimeView
	AddAnimeView
	FindAnimeView
	DropAnimeView
	RSSView
	MaintenanceView
	AbortView
)

type MenuItem int

const (
	Find = MenuItem(iota)
	Watch
	Update
	Delete
	Drop
	Rss
	Clean
	Add
)

type AnimeSource int

const (
	Kitsu = AnimeSource(iota)
	Local
)

func (s AnimeSource) String() string {
	switch s {
	case Kitsu:
		return "Kitsu"
	case Local:
		return "Local"
	default:
		return "Unknown"
	}
}

type (
	FetchErrorMsg              error
	FetchedNoResultsMsg        struct{}
	FetchedListItemsMsg[T any] struct {
		items   []list.Item
		results []T
	}
)

type ViewModel interface {
	tea.CursorModel
	help.KeyMap
	Update(msg tea.Msg) (ViewModel, tea.Cmd)
}

var MenuItemMap = map[MenuItem]string{
	Find:   "Find",
	Add:    "Add",
	Watch:  "Watch",
	Update: "Update",
	Delete: "Delete",
	Drop:   "Drop",
	Rss:    "RSS",
	Clean:  "Clean",
}

type MainKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Submit   key.Binding
	Abort    key.Binding
	MainMenu key.Binding
	HelpMore key.Binding
	HelpLess key.Binding
	Exit     key.Binding
}

var keyMap = MainKeyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Submit:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	HelpMore: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more help")),
	HelpLess: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "less help")),
	Abort:    key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "abort")),
	MainMenu: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "menu")),
	Exit:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "exit")),
}

type (
	AbortMsg      struct{}
	WindowSizeMsg struct {
		Width  int
		Height int
	}
)

type viewState struct {
	view       UIView
	consentPos ui.Consent
	menuPos    int
	height     int
	width      int
	loading    struct {
		active bool
		text   string
	}
}

type Model struct {
	db          *database.Database
	menuItems   []MenuItem
	menuViewMap map[UIView]ViewModel
	state       viewState
	help        help.Model
	input       textinput.Model
	spinner     spinner.Model
}

func New(dbPath string) (Model, error) {
	h := help.New()
	h.Styles.ShortKey = ui.HelpKeyStyle
	h.Styles.FullKey = h.Styles.ShortKey

	h.Styles.ShortDesc = ui.HelpDescStyle
	h.Styles.FullDesc = h.Styles.ShortDesc

	input := ui.NewTextInput()
	input.SetWidth(20)
	input.Focus()

	s := spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"},
		FPS:    time.Second / 10,
	}))

	model := Model{help: h, input: input, spinner: s}

	model.menuItems = append(
		model.menuItems,
		Find,
		Watch,
		Update,
		Drop,
		Rss,
		Delete,
		Clean,
	)

	// Initialize empty database
	db, _ := database.NewDatabase(nil)
	model.db = db

	model.menuViewMap = map[UIView]ViewModel{
		SetupUserView: NewUserSetupModel(),
		FindAnimeView: NewFindMenuModel(db),
	}

	if !utils.FileExists(dbPath) {
		model.SetViewState(SetupUserView)
		return model, nil
	}

	err := db.Load()
	if err != nil {
		return model, err
	}

	model.SetViewState(MenuView)

	return model, nil
}

func (ui Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	var currentModel ViewModel

	if model, exists := m.menuViewMap[m.state.view]; exists {
		currentModel = model
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.width = msg.Width
		m.state.height = msg.Height
		// Send size to all menu view models
		for key, model := range m.menuViewMap {
			m.menuViewMap[key], _ = model.Update(msg)
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.HelpMore):
			if (currentModel != nil && len(currentModel.FullHelp()) == 0) || m.state.view == MenuView {
				break
			}
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, keyMap.MainMenu):
			if m.state.view == MenuView {
				m.state.view = ExitView
				return m, tea.Quit
			}
		}

		if msg.String() == "ctrl+c" {
			m.state.view = AbortView
			return m, tea.Quit
		}

	case SetupUserFinishedMsg:
		m.SetViewState(MenuView)
		err := m.db.LoadData(msg)
		if err != nil {
			// FIX  Do not panic, we should display a message
			panic(err)
		}

	case AbortMsg:
		// Do not abort application when inside of sub-menu
		switch m.state.view {
		case FindAnimeView, AddAnimeView, DropAnimeView:
			m.SetViewState(MenuView)
			return m, nil
		}
		m.SetViewState(AbortView)
		return m, tea.Quit

	}

	if currentModel != nil {
		// We do not want to add new models to the map
		if _, exists := m.menuViewMap[m.state.view]; exists {
			m.menuViewMap[m.state.view], cmd = currentModel.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	switch m.state.view {
	case MenuView:
		m, cmd = m.UpdateMenu(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) SetViewState(view UIView) {
	m.state.view = view
}

func (m Model) View() (string, *tea.Cursor) {
	if model, exists := m.menuViewMap[m.state.view]; exists {
		view, c := model.View()
		return lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			ui.HelpStyle.Render(m.help.View(model)),
		), c
	}

	switch m.state.view {
	case AbortView:
		return ui.AbortStyle.Render(
			utils.ColorText(";g;>>> ;y;User Aborted Operation ;g;<<<"),
		), nil

	case MenuView:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.viewMenu(),
			ui.TextStyle.Render(m.help.View(m)),
		), nil

	case ExitView:
		return "", nil

	case None:
		anime := m.db.GetAllAnime()
		return ui.Style.PaddingTop(1).PaddingLeft(3).Render(
			utils.ColorText(fmt.Sprintf(";g;%d ;w;Library Entries Loaded", len(anime))),
		), nil
	}

	return "missing view", nil
}

func (m Model) UpdateMenu(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	itemLen := len(m.menuItems)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.Select):
			switch m.menuItems[m.state.menuPos] {
			case Find:
				m.SetViewState(FindAnimeView)
				cmd = textinput.Blink
			}

		case key.Matches(msg, keyMap.Up):
			m.state.menuPos = (m.state.menuPos - 1 + itemLen) % itemLen

		case key.Matches(msg, keyMap.Down):
			m.state.menuPos = (m.state.menuPos + 1) % itemLen

		}
	}

	return m, cmd
}

func (m Model) viewMenu() string {
	items := make([]string, len(m.menuItems)+1)
	s := ui.TextStyle.MarginLeft(5).Width(12).PaddingLeft(1).PaddingRight(3)

	header := ui.TextStyle.MarginTop(1).
		MarginBottom(1).
		Render(utils.ColorText(";b;What would you like to do?"))
	items[0] = header

	for i, item := range m.menuItems {
		if m.state.menuPos == i {
			items[i+1] = s.Foreground(ansi.BrightGreen).
				Background(ansi.Black).
				Render("> " + MenuItemMap[item])
			continue
		}
		items[i+1] = s.Render("  " + MenuItemMap[item])
	}

	// Add empty line between help list
	items = append(items, "")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		items...,
	)
}

func (m Model) ShortHelp() []key.Binding {
	switch m.state.view {
	case MenuView:
		return []key.Binding{
			keyMap.Up,
			keyMap.Down,
			keyMap.Select,
			keyMap.Exit,
		}
	}
	return []key.Binding{}
}

func (m Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func abort() tea.Msg {
	return AbortMsg{}
}

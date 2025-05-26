package views

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type UIView int

const (
	None = UIView(iota)
	SetupUser
	Menu
	Exit
	WatchAnimeView
	AddAnime
	DelAnime
	FindAnime
	DropAnime
	RSS
	Maintenance
	Abort
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

type (
	FetchErrorMsg       error
	FetchedNoResultsMsg struct{}
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

type (
	AbortMsg      struct{}
	WindowSizeMsg struct {
		Width  int
		Height int
	}
)

type Model struct {
	db          *database.Database
	menuItems   []MenuItem
	menuViewMap map[UIView]ViewModel
	help        help.Model
	input       textinput.Model
	spinner     spinner.Model
	state       struct {
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
		Add,
		Delete,
	)

	// Initialize empty database
	db, _ := database.NewDatabase(nil)
	model.db = db

	model.menuViewMap = map[UIView]ViewModel{
		SetupUser: newUserSetupModel(),
		FindAnime: newFindAnimeModel(db),
		AddAnime:  newAddAnimeModel(db),
		DelAnime:  newDelAnimeModel(db),
	}

	if !utils.FileExists(dbPath) {
		model.SetViewState(SetupUser)
		return model, nil
	}

	err := db.Load()
	if err != nil {
		return model, err
	}

	model.SetViewState(Menu)

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
		case key.Matches(msg, ui.KeyMap.HelpMore):
			if (currentModel != nil && len(currentModel.FullHelp()) == 0) || m.state.view == Menu {
				break
			}
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, ui.KeyMap.MainMenu):
			if m.state.view == Menu {
				m.state.view = Exit
				return m, tea.Quit
			}
		}

		if msg.String() == "ctrl+c" {
			m.state.view = Abort
			return m, tea.Quit
		}

	case SetupUserFinishedMsg:
		m.SetViewState(Menu)
		err := m.db.LoadData(msg)
		if err != nil {
			// FIX  Do not panic, we should display a message
			panic(err)
		}

	case AbortMsg:
		// Do not abort application when inside of sub-menu
		switch m.state.view {
		case FindAnime, AddAnime, DelAnime:
			m.SetViewState(Menu)
			return m, nil
		}
		m.SetViewState(Abort)
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
	case Menu:
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
	case Abort:
		return ui.AbortStyle.Render(
			utils.ColorText(";g;>>> ;y;User Aborted Operation ;g;<<<"),
		), nil

	case Menu:
		profile := m.db.GetProfile()
		header := ui.TextStyle.
			MarginTop(1).
			MarginBottom(1).
			Render(utils.ColorText(fmt.Sprintf(";dy;%s's;b; profile stats:", profile.Username)))

		expStyle := ui.Style
		tokenExpiration := utils.NewRelativeTimeUnits(profile.TokenExpirationSec)
		switch {
		case tokenExpiration.Weeks < 1:
			expStyle = expStyle.Foreground(ansi.BrightRed)
		case tokenExpiration.Weeks < 2:
			expStyle = expStyle.Foreground(ansi.BrightYellow)
		default:
			expStyle = expStyle.Foreground(ansi.BrightGreen)
		}

		d := newPropValDisplay([]string{
			utils.ColorText(";dc;Completed Anime"),
			utils.ColorText(";dc;Time Watched"),
			utils.ColorText(";dc;Token Expiration"),
			utils.ColorText(";dc;Last Updated"),
		}, []string{
			strconv.Itoa(profile.CompletedSeries),
			utils.NewDurationUnits(time.Second * time.Duration(profile.SecondsWatched)).
				ToShortString(),
			expStyle.Render(tokenExpiration.ToPrecisionString(utils.Days)),
			utils.NewRelativeTimeUnits(profile.LastUpdateSec).String(),
		})

		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			d,
			m.viewMenu(),
			ui.TextStyle.Render(m.help.View(m)),
		), nil

	case Exit:
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
		case key.Matches(msg, ui.KeyMap.Select):
			switch m.menuItems[m.state.menuPos] {
			case Find:
				m.SetViewState(FindAnime)
				cmd = textinput.Blink
			case Add:
				m.SetViewState(AddAnime)
				cmd = textinput.Blink
			case Delete:
				m.SetViewState(DelAnime)
				cmd = textinput.Blink
			}

		case key.Matches(msg, ui.KeyMap.Up):
			m.state.menuPos = (m.state.menuPos - 1 + itemLen) % itemLen

		case key.Matches(msg, ui.KeyMap.Down):
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
	case Menu:
		return []key.Binding{
			ui.KeyMap.Up,
			ui.KeyMap.Down,
			ui.KeyMap.Select,
			ui.KeyMap.Exit,
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

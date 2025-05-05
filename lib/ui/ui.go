package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jaeiya/koshime/lib/database"
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
	SetupUserView
	MenuView
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

var keyMap = userSetupKeyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Submit:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	HelpMore: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more help")),
	HelpLess: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "less help")),
	Abort:    key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "abort")),
}

type (
	AbortMsg             struct{}
	SetupUserMsg         struct{}
	SetupUserFinishedMsg struct{}
)

type Consent int

const (
	No = Consent(iota)
	Yes
)

type uiState struct {
	view       UIView
	consentPos Consent
	menuPos    int
	loading    struct {
		active bool
		text   string
	}
}

type UIModel struct {
	db        *database.Database
	menuItems []MenuItem
	state     struct {
		internal  uiState
		userSetup userSetupState
	}
	help    help.Model
	input   textinput.Model
	spinner spinner.Model
}

func NewUI(dbPath string) (UIModel, error) {
	h := help.New()
	h.Styles.ShortKey = helpKeyStyle
	h.Styles.FullKey = h.Styles.ShortKey

	h.Styles.ShortDesc = helpDescStyle
	h.Styles.FullDesc = h.Styles.ShortDesc

	input := textinput.New()
	input.SetWidth(20)
	input.Focus()
	input.CharLimit = 30
	input.Prompt = "   > "
	input.EchoCharacter = '•'
	input.Styles.Focused.Prompt = inputPromptStyle
	input.Styles.Focused.Text = inputTextStyle

	s := spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"},
		FPS:    time.Second / 10,
	}))

	model := UIModel{help: h, input: input, spinner: s}

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

	if !utils.FileExists(dbPath) {
		model.SetViewState(SetupUserView)
		return model, nil
	}

	db, _ := database.NewDatabase(nil)
	err := db.Load()
	if err != nil {
		return model, err
	}

	model.db = db
	model.SetViewState(MenuView)

	return model, nil
}

func (ui UIModel) Init() tea.Cmd {
	return nil
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var err error
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg { return AbortMsg{} }
		}

	case SetupUserFinishedMsg:
		m.SetViewState(None)
		m.db, err = database.NewDatabase(&m.state.userSetup.data)
		if err != nil {
			panic(err)
		}
		return m, tea.Quit

	case AbortMsg:
		// Do not abort application when inside of sub-menu
		switch m.state.internal.view {
		case FindAnimeView, AddAnimeView, DropAnimeView:
			m.SetViewState(MenuView)
			return m, nil
		}
		m.SetViewState(AbortView)
		return m, tea.Quit

	}

	if m.isLoading() {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.internal.view {
	case SetupUserView:
		m, cmd = m.UpdateUserSetup(msg)
		cmds = append(cmds, cmd)

	case FindAnimeView:
		m, cmd = m.UpdateFindAnime(msg)
		cmds = append(cmds, cmd)

	case MenuView:
		m = m.UpdateMenu(msg)

	// Temporary
	case None:
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

func (m *UIModel) SetViewState(view UIView) {
	m.state.internal.view = view
}

func (m UIModel) View() (string, *tea.Cursor) {
	switch m.state.internal.view {
	case SetupUserView:
		return m.ViewUserSetup()
	case FindAnimeView:
		return m.ViewFindAnime()

	case AbortView:
		return abortStyle.Render(
			utils.ColorText(";g;>>> ;y;User Aborted Operation ;g;<<<"),
		), nil

	case MenuView:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.viewMenu(),
			textStyle.Render(m.help.View(m)),
		), nil

	case None:
		anime := m.db.GetAllAnime()
		return style.PaddingTop(1).PaddingLeft(3).Render(
			utils.ColorText(fmt.Sprintf(";g;%d ;w;Library Entries Loaded", len(anime))),
		), nil
	}

	return "missing view", nil
}

func (m UIModel) UpdateMenu(msg tea.Msg) UIModel {
	itemLen := len(m.menuItems)
	state := &m.state.internal
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.Select):
			switch m.menuItems[state.menuPos] {
			case Find:
				m.SetViewState(FindAnimeView)
			}

		case key.Matches(msg, keyMap.Up):
			state.menuPos = (state.menuPos - 1 + itemLen) % itemLen

		case key.Matches(msg, keyMap.Down):
			state.menuPos = (state.menuPos + 1) % itemLen
		}
	}
	return m
}

func (m UIModel) viewMenu() string {
	items := make([]string, len(m.menuItems)+1)
	s := textStyle.MarginLeft(5).Width(12).PaddingLeft(1).PaddingRight(3)

	header := textStyle.MarginTop(1).
		MarginBottom(1).
		Render(utils.ColorText(";b;What would you like to do?"))
	items[0] = header

	for i, item := range m.menuItems {
		if m.state.internal.menuPos == i {
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

func (m UIModel) ShortHelp() []key.Binding {
	switch m.state.internal.view {
	case SetupUserView:
		return m.userSetupShortHelp()
	case MenuView:
		return []key.Binding{
			keyMap.Up,
			keyMap.Down,
			keyMap.Select,
		}

	}
	return []key.Binding{}
}

func (m UIModel) FullHelp() [][]key.Binding {
	switch m.state.internal.view {
	case SetupUserView:
		return m.userSetupFullHelp()
	}
	return [][]key.Binding{}
}

func (m UIModel) isConsenting() bool {
	return m.state.internal.consentPos == Yes
}

func (m *UIModel) setConsentStartPos(c Consent) {
	m.state.internal.consentPos = c
}

func (m UIModel) updateConsent(msg tea.Msg) UIModel {
	state := &m.state.internal
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.Down):
			state.consentPos = Yes
		case key.Matches(msg, keyMap.Up):
			state.consentPos = No
		}
	}
	return m
}

func (m UIModel) viewConsent(msg ...string) string {
	state := m.state.internal
	var yes, no string
	if state.consentPos == No {
		no = selectNoStyle.Render("> No")
		yes = textStyle.Render("  Yes")
	} else {
		yes = selectYesStyle.Render("> Yes")
		no = textStyle.MarginTop(1).Render("  No")
	}

	msg = append(msg, no, yes)
	return lipgloss.JoinVertical(lipgloss.Left, msg...)
}

func (m UIModel) isLoading() bool {
	return m.state.internal.loading.active
}

func (m *UIModel) setLoadingState(s bool) {
	m.state.internal.loading.active = s
}

func (m *UIModel) setLoadingText(s string) {
	m.state.internal.loading.text = s
}

func (m UIModel) viewLoading() string {
	spinnerStr := spinnerStyle.Render(strings.Repeat(m.spinner.View(), 3))
	return textStyle.Render(
		fmt.Sprintf(
			"%s %s %s",
			spinnerStr,
			loadingStyle.Render(m.state.internal.loading.text),
			spinnerStr,
		),
	)
}

func (m UIModel) abort() tea.Msg {
	return AbortMsg{}
}

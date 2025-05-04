package ui

import (
	"fmt"
	"time"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type UIView int

const (
	None = UIView(iota)
	SetupUserView
	AbortView
)

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

type uiState struct {
	view UIView
}

type UIModel struct {
	db    *database.Database
	state struct {
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

	if !utils.FileExists(dbPath) {
		model.state.internal.view = SetupUserView
		return model, nil
	}

	db, _ := database.NewDatabase(nil)
	err := db.Load()
	if err != nil {
		return model, err
	}

	model.db = db

	return model, nil
}

func (ui UIModel) Init() tea.Cmd {
	return nil
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var err error
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg { return AbortMsg{} }
		}

	case SetupUserFinishedMsg:
		m.state.internal.view = None
		m.db, err = database.NewDatabase(&m.state.userSetup.userData)
		if err != nil {
			panic(err)
		}
		return m, tea.Quit

	case AbortMsg:
		m.state.internal.view = AbortView
		return m, tea.Quit

	}

	switch m.state.internal.view {
	case SetupUserView:
		return m.UpdateUserSetup(msg)
	}
	return m, nil
}

func (m UIModel) View() (string, *tea.Cursor) {
	switch m.state.internal.view {
	case SetupUserView:
		return m.ViewUserSetup()

	case AbortView:
		return abortStyle.Render(
			utils.ColorText(";g;>>> ;y;User Aborted Operation ;g;<<<"),
		), nil

	case None:
		anime := m.db.GetAllAnime()
		return style.PaddingTop(1).PaddingLeft(3).Render(
			utils.ColorText(fmt.Sprintf(";g;%d ;w;Library Entries Loaded", len(anime))),
		), nil
	}

	return "missing view", nil
}

func (m UIModel) ShortHelp() []key.Binding {
	state := m.state.userSetup

	switch state.view {
	case SetupConsentView:
		return []key.Binding{
			keyMap.Up,
			keyMap.Down,
			keyMap.Select,
			keyMap.HelpMore,
		}

	case SetupUsernameView:
		if state.username.failed || state.username.passed {
			return []key.Binding{keyMap.Up, keyMap.Down, keyMap.Select}
		}
		return []key.Binding{keyMap.Submit, keyMap.Abort}

	case SetupPasswordView:
		if state.password.failed {
			return []key.Binding{keyMap.Up, keyMap.Down, keyMap.Select}
		}
		return []key.Binding{keyMap.Submit, keyMap.Abort}

	case SetupLibraryView:
		if state.libAnime.passed {
			return []key.Binding{keyMap.Submit, keyMap.Abort}
		}
		return []key.Binding{keyMap.Up, keyMap.Down, keyMap.Select}
	}

	return []key.Binding{}
}

func (m UIModel) FullHelp() [][]key.Binding {
	switch m.state.userSetup.view {
	case SetupConsentView:
		return [][]key.Binding{
			{keyMap.Up, keyMap.Down, keyMap.Select},
			{keyMap.Abort, keyMap.HelpLess},
		}
	}
	return [][]key.Binding{}
}

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

type Consent int

const (
	No = Consent(iota)
	Yes
)

type uiState struct {
	view       UIView
	consentPos Consent
	loading    struct {
		active bool
		text   string
	}
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
		model.SetViewState(SetupUserView)
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
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg { return AbortMsg{} }
		}

	case SetupUserFinishedMsg:
		m.SetViewState(None)
		m.db, err = database.NewDatabase(&m.state.userSetup.userData)
		if err != nil {
			panic(err)
		}
		return m, tea.Quit

	case AbortMsg:
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
	switch m.state.internal.view {
	case SetupUserView:
		return m.userSetupShortHelp()
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

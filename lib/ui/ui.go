package ui

import (
	"time"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/help"
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
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg { return AbortMsg{} }
		}

	case AbortMsg:
		m.state.internal.view = AbortView
		return m, tea.Quit

	}

	return m.UpdateUserSetup(msg)
}

func (m UIModel) View() (string, *tea.Cursor) {
	switch m.state.internal.view {
	case AbortView:
		return abortStyle.Render(
			utils.ColorText(";g;>>> ;y;User Aborted Operation ;g;<<<"),
		), nil
	}
	return m.ViewUserSetup()
}

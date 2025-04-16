package ui

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jaeiya/koshime/lib"
	"github.com/jaeiya/koshime/lib/utils"
)

var (
	questionStyle = defaultTextStyle.Foreground(ansi.BrightWhite)
	selectedStyle = defaultTextStyle.Foreground(ansi.BrightMagenta)

	selectStyle = lipgloss.NewStyle().PaddingLeft(3)
	selectedYes = selectStyle.Foreground(ansi.BrightGreen).Render("> Yes")
	selectedNo  = selectStyle.MarginTop(1).Foreground(ansi.BrightMagenta).Render("> No")
)

type ViewState int

const (
	ConsentView = ViewState(iota)
	UsernameView
	PasswordView
	AbortView
)

type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
	Help  key.Binding
}

func (km keyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Quit, km.Help}
}

func (km keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.Up, km.Down, km.Enter},
		{km.Help, km.Quit},
	}
}

var keys = keyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
	Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
	Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Quit:  key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
	Help:  key.NewBinding(key.WithKeys("shift+/"), key.WithHelp("?", "toggle help")),
}

type User struct{}

func (User) Init() lib.DBData {
	h := help.New()
	h.Styles.ShortKey = h.Styles.ShortKey.Foreground(lipgloss.Color("#787897"))
	h.Styles.FullKey = h.Styles.ShortKey

	h.Styles.ShortDesc = h.Styles.ShortDesc.Foreground(lipgloss.Color("#56566B"))
	h.Styles.FullDesc = h.Styles.ShortDesc

	input := textinput.New()
	input.SetWidth(20)
	input.Focus()
	input.CharLimit = 30
	input.Prompt = "   > "
	input.Styles.Focused.Prompt = lipgloss.NewStyle().Foreground(ansi.BrightGreen)
	input.Styles.Focused.Text = lipgloss.NewStyle().Foreground(ansi.BrightWhite)

	p := tea.NewProgram(userModel{
		isConsenting: true,
		input:        input,
		keys:         keys,
		help:         h,
	})
	m, err := p.Run()
	if err != nil {
		panic(err)
	}

	return m.(userModel).Data
}

type userModel struct {
	keys         keyMap
	help         help.Model
	input        textinput.Model
	isConsenting bool
	consentPos   int
	userName     string
	password     string
	viewState    ViewState
	blinks       int
	Data         lib.DBData
}

func (m userModel) Init() tea.Cmd {
	return nil
}

func (m userModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width

	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		if key.Matches(msg, m.keys.Help) {
			m.help.ShowAll = !m.help.ShowAll
		}

	}

	switch m.viewState {
	case ConsentView:
		m, cmd = m.UpdateUserConsent(msg)
	case UsernameView:
		m, cmd = m.UpdateUserName(msg)
	case PasswordView:
		m, cmd = m.UpdatePassword(msg)
	}

	return m, cmd
}

func (m userModel) UpdateUserConsent(msg tea.Msg) (userModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Down):
			m.consentPos = utils.AbsInt(m.consentPos-1) % 2

		case key.Matches(msg, m.keys.Up):
			m.consentPos = utils.AbsInt(m.consentPos+1) % 2

		case key.Matches(msg, m.keys.Enter):
			if m.consentPos == 0 {
				m.viewState = AbortView
				return m, tea.Quit
			}
			m.viewState = UsernameView
			return m, textinput.Blink

		}
	}
	return m, nil
}

func (m userModel) UpdateUserName(msg tea.Msg) (userModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			m.userName = m.input.Value()
			m.viewState = PasswordView
			m.input.Reset()
			m.input.EchoCharacter = '•'
			m.input.EchoMode = textinput.EchoPassword
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m userModel) UpdatePassword(msg tea.Msg) (userModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			m.password = m.input.Value()
			m.input.Reset()
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m userModel) View() (string, *tea.Cursor) {
	var c *tea.Cursor
	var view string
	switch m.viewState {
	case ConsentView:
		view = m.ConsentView()
	case UsernameView:
		view, c = m.UsernameView()
	case PasswordView:
		view, c = m.PasswordView()
	case AbortView:
		view = m.AbortView()
	}

	if m.viewState >= UsernameView {
		return view, c
	}

	helpView := defaultTextStyle.Height(3).PaddingTop(1).Render(m.help.View(m.keys))
	return lipgloss.JoinVertical(lipgloss.Left, view, helpView), c
}

func (m userModel) ConsentView() string {
	yes, no := m.GetYesNo(m.consentPos)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		UserWelcomeMsg,
		UserConsentMsg,
		no,
		yes,
	)
}

func (m userModel) UsernameView() (string, *tea.Cursor) {
	c := m.input.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		UserNameMsg,
		lipgloss.NewStyle().Render(m.input.View()),
	)
	c.Y += lipgloss.Height(view)
	return view, c
}

func (m userModel) PasswordView() (string, *tea.Cursor) {
	c := m.input.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		PasswordMsg,
		lipgloss.NewStyle().Render(m.input.View()),
	)
	c.Y += lipgloss.Height(view)
	return view, c
}

func (m userModel) AbortView() string {
	return lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Render(utils.ColorText(";g;>>> ;y;Koshime Setup Aborted ;g;<<<;x;"))
}

func (m userModel) GetYesNo(state int) (yes string, no string) {
	if state == 0 {
		no = selectedNo
		yes = selectStyle.Render(" Yes")
	} else {
		yes = selectedYes
		no = selectStyle.MarginTop(1).Render(" No")
	}
	return yes, no
}

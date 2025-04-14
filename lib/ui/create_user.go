package ui

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jaeiya/koshime/lib"
	"github.com/jaeiya/koshime/lib/utils"
)

var (
	questionStyle = defaultTextStyle.Foreground(ansi.BrightWhite)
	selectedStyle = defaultTextStyle.Foreground(ansi.BrightMagenta)
)

type ViewState int

const (
	ConsentView = ViewState(iota)
	UsernameView
	PasswordView
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

	p := tea.NewProgram(userModel{
		isConsenting: true,
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
	isConsenting bool
	consentPos   int
	viewState    ViewState
	Data         lib.DBData
}

func (m userModel) Init() tea.Cmd {
	return nil
}

func (m userModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	return m.UpdateUserConsent(msg)
}

func (m userModel) UpdateUserConsent(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Down):
			m.consentPos = utils.AbsInt(m.consentPos-1) % 2
		case key.Matches(msg, m.keys.Up):
			m.consentPos = utils.AbsInt(m.consentPos+1) % 2
		case key.Matches(msg, m.keys.Enter):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m userModel) View() string {
	var view string
	switch m.viewState {
	case ConsentView:
		view = m.ConsentView()
	}

	helpView := defaultTextStyle.Height(3).PaddingTop(1).Render(m.help.View(m.keys))

	return lipgloss.JoinVertical(lipgloss.Left, view, helpView)
}

func (m userModel) ConsentView() string {
	selStyle := lipgloss.NewStyle().PaddingLeft(3)
	var yes, no string
	if m.consentPos == 0 {
		no = selStyle.MarginTop(1).Foreground(ansi.BrightMagenta).Render("> No")
		yes = selStyle.Render("  Yes")
	} else {
		yes = selStyle.Foreground(ansi.BrightGreen).Render("> Yes")
		no = selStyle.MarginTop(1).Render("  No")
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		UserWelcomeMsg,
		UserConsentMsg,
		no,
		yes,
	)
}

func (m userModel) NextView() {
	m.viewState += 1
}

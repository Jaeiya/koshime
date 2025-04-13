package ui

import (
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

type User struct{}

func (User) Init() lib.DBData {
	p := tea.NewProgram(userModel{
		isConsenting: true,
	})
	m, err := p.Run()
	if err != nil {
		panic(err)
	}

	return m.(userModel).Data
}

type userModel struct {
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
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEsc:
			return m, tea.Quit
		}

		// Exit with Ctrl+C
		switch msg.Mod {
		case tea.ModCtrl:
			switch msg.Code {
			case 'c':
				return m, tea.Quit
			}
		}
	}

	return m.UpdateUserConsent(msg)
}

func (m userModel) UpdateUserConsent(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case 'j', tea.KeyDown:
			m.consentPos = utils.AbsInt(m.consentPos-1) % 2
		case 'k', tea.KeyUp:
			m.consentPos = utils.AbsInt(m.consentPos+1) % 2
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

	footer := defaultTextStyle.Render(
		utils.ColorText(";bk;Koshime ;dg;Alpha;x; ;bk;- Hit Esc or Ctrl+C to quit;x;"),
	)

	return lipgloss.JoinVertical(lipgloss.Left, view, footer)
}

func (m userModel) ConsentView() string {
	selStyle := lipgloss.NewStyle().PaddingLeft(3)
	var yes, no string
	if m.consentPos == 1 {
		no = selStyle.MarginTop(1).Foreground(ansi.BrightMagenta).Render("> No")
		yes = selStyle.Render("  Yes")
	} else {
		yes = selStyle.Foreground(ansi.BrightMagenta).Render("> Yes")
		no = selStyle.MarginTop(1).Render("  No")
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		UserWelcomeMsg,
		UserConsentMsg,
		no,
		yes,
		lipgloss.NewStyle().Render(""),
	)
}

func (m userModel) NextView() {
	m.viewState += 1
}

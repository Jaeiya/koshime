package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jaeiya/koshime/lib"
	"github.com/jaeiya/koshime/lib/kitsu"
	"github.com/jaeiya/koshime/lib/utils"
)

var (
	questionStyle = defaultTextStyle.Foreground(ansi.BrightWhite)
	selectedStyle = defaultTextStyle.Foreground(ansi.BrightMagenta)

	selectStyle = lipgloss.NewStyle().PaddingLeft(3)
	selectedYes = selectStyle.Foreground(ansi.BrightGreen).Render("> Yes")
	selectedNo  = selectStyle.MarginTop(1).Foreground(ansi.BrightMagenta).Render("> No")
)

type (
	FetchedAuthTokenMsg kitsu.AuthToken
	AuthTokenErrorMsg   error
)

type ViewState int

const (
	ConsentView = ViewState(iota)
	UsernameView
	ConfirmUsernameView
	UsernameFailedView
	PasswordView
	PasswordFailedView
	Completed
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
	Quit:  key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "quit")),
	Help:  key.NewBinding(key.WithKeys("shift+/"), key.WithHelp("?", "toggle help")),
}

type User struct{}

func (User) NewUser() (lib.DBData, bool) {
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
	input.EchoCharacter = '•'
	input.Styles.Focused.Prompt = lipgloss.NewStyle().Foreground(ansi.BrightGreen)
	input.Styles.Focused.Text = lipgloss.NewStyle().Foreground(ansi.BrightWhite)

	p := tea.NewProgram(userModel{
		input:     input,
		keys:      keys,
		help:      h,
		db:        lib.DBData{},
		viewState: ConsentView,
	})
	m, err := p.Run()
	if err != nil {
		panic(err)
	}

	return m.(userModel).db, m.(userModel).isAborted
}

type userModel struct {
	keys           keyMap
	help           help.Model
	input          textinput.Model
	consentPos     int
	db             lib.DBData
	password       string
	isLoading      bool
	isAborted      bool
	fetchedProfile kitsu.ProfileData
	authError      error
	viewState      ViewState
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
			m.isAborted = true
			m.viewState = AbortView
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
	case UsernameFailedView:
		m, cmd = m.UpdateUsernameFailed(msg)
	case ConfirmUsernameView:
		m, cmd = m.UpdateConfirmUsername(msg)
	case PasswordView:
		m, cmd = m.UpdatePassword(msg)
	case PasswordFailedView:
		m, cmd = m.UpdatePasswordFailed(msg)
	}

	return m, cmd
}

func (m userModel) UpdateUserConsent(msg tea.Msg) (userModel, tea.Cmd) {
	m = m.updateConsent(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Enter):
			if m.consentPos == 0 {
				m.viewState = AbortView
				m.isAborted = true
				return m, tea.Quit
			}
			m.viewState = UsernameView
			m.consentPos = 0 // Reset for future
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
			if !m.isLoading {
				m.isLoading = true
				return m, getProfile(m.input.Value())
			}
		}

	case kitsu.ProfileData:
		m.isLoading = false
		m.fetchedProfile = msg
		if len(m.fetchedProfile.Data) == 0 {
			// Start on Yes
			m.consentPos = 1
			m.viewState = UsernameFailedView
		} else {
			m.viewState = ConfirmUsernameView
		}

	// If getting the profile returns an error
	case error:
		panic(msg)

	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func getProfile(userName string) func() tea.Msg {
	return func() tea.Msg {
		p, err := kitsu.GetProfile(userName)
		if err != nil {
			return err
		}
		return p
	}
}

func (m userModel) UpdateUsernameFailed(msg tea.Msg) (userModel, tea.Cmd) {
	m = m.updateConsent(msg)
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Enter):
			if m.consentPos == 0 {
				m.isAborted = true
				m.viewState = AbortView
				return m, tea.Quit
			}
			m.consentPos = 0 // reset for future use
			m.input.Reset()
			m.viewState = UsernameView
		}
	}
	return m, nil
}

func (m userModel) UpdateConfirmUsername(msg tea.Msg) (userModel, tea.Cmd) {
	m = m.updateConsent(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Enter):
			if m.consentPos == 0 {
				m.input.Reset()
				m.viewState = UsernameView
				return m, nil
			}

			m.consentPos = 0 // Reset for future
			m.viewState = PasswordView
			m.input.Reset()
			m.input.EchoMode = textinput.EchoPassword
			return m, nil
		}
	}

	return m, nil
}

func (m userModel) UpdatePassword(msg tea.Msg) (userModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			m.isLoading = true
			return m, m.GetAuthToken
		}

	case FetchedAuthTokenMsg:
		m.isLoading = false
		m.db.Profile.AccessToken = msg.Token
		m.db.Profile.RefreshToken = msg.RefreshToken
		m.db.Profile.TokenExpiration = msg.ExpiresIn

		userData := m.fetchedProfile.Data[0]
		userStats := m.fetchedProfile.Included[0]

		m.db.Profile.ID = userData.ID
		m.db.Profile.Username = userData.Attributes.Name
		m.db.Profile.CompletedSeries = userStats.Attributes.Stats.CompletedAnime
		m.db.Profile.SecondsWatched = userStats.Attributes.Stats.SecondsWatched

		m.viewState = Completed
		return m, tea.Quit

	case AuthTokenErrorMsg:
		m.isLoading = false
		m.input.Reset()
		m.input.EchoMode = textinput.EchoNormal
		m.viewState = PasswordFailedView
		m.authError = msg

	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m userModel) UpdatePasswordFailed(msg tea.Msg) (userModel, tea.Cmd) {
	m = m.updateConsent(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Enter):
			if m.consentPos == 0 {
				m.viewState = AbortView
				m.isAborted = true
				return m, tea.Quit
			}
			m.viewState = UsernameView
			m.consentPos = 0 // Reset for future
			return m, nil
		}
	}

	return m, nil
}

func (m userModel) View() (string, *tea.Cursor) {
	var c *tea.Cursor
	var view string
	switch m.viewState {
	case ConsentView:
		view = m.ConsentView()
	case UsernameView:
		view, c = m.UsernameView()
	case UsernameFailedView:
		view = m.UsernameFailedView()
	case ConfirmUsernameView:
		view = m.ConfirmUsernameView()
	case PasswordView:
		view, c = m.PasswordView()
	case PasswordFailedView:
		view = m.PasswordFailedView()
	case AbortView:
		view = m.AbortView()
	}

	if m.viewState == UsernameView || m.viewState == PasswordView || m.viewState == AbortView {
		return view, c
	}

	helpView := defaultTextStyle.Height(3).PaddingTop(1).Render(m.help.View(m.keys))
	return lipgloss.JoinVertical(lipgloss.Left, view, helpView), c
}

func (m userModel) ConsentView() string {
	yes, no := m.GetYesNo(m.consentPos)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		userWelcomeTxt,
		userConsentTxt,
		no,
		yes,
	)
}

func (m userModel) UsernameView() (string, *tea.Cursor) {
	if m.isLoading {
		return defaultTextStyle.MarginTop(1).Render("Loading..."), nil
	}
	c := m.input.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		userNameTxt,
		lipgloss.NewStyle().Render(m.input.View()),
	)
	c.Y += lipgloss.Height(view)
	return view, c
}

func (m userModel) UsernameFailedView() string {
	yes, no := m.GetYesNo(m.consentPos)
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		usernameFailedTxt,
		no,
		yes,
	)
	return view
}

func (m userModel) ConfirmUsernameView() string {
	yes, no := m.GetYesNo(m.consentPos)
	userID := m.fetchedProfile.Data[0].ID
	attr := m.fetchedProfile.Data[0].Attributes

	createdDate, err := time.Parse(time.RFC3339, attr.CreatedAt)
	if err != nil {
		panic(err)
	}

	profileStr := defaultTextStyle.PaddingLeft(5).MarginTop(1).
		Width(60).
		Render(utils.ColorText(strings.Trim((fmt.Sprintf(`
    ;w;Name:;x; ;g;%s;x;
   ;w;About:;x; %s
  ;w;Gender:;x; %s
;w;BirthDay:;x; %s
;w;Location:;x; %s
 ;w;Created:;x; %s
 ;w;Profile:;x; %s`, attr.Name, attr.About, attr.Gender, attr.Birthday, attr.Location, createdDate.Local().Format("01/02/2006 3:04 PM"), kitsu.GetProfileLink(userID))), "\n")))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		confirmUsernamePreTxt,
		profileStr,
		confirmUsernameConsentTxt,
		no,
		yes,
	)
}

func (m userModel) PasswordView() (string, *tea.Cursor) {
	if m.isLoading {
		return defaultTextStyle.MarginTop(1).Render("Loading..."), nil
	}

	c := m.input.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		passwordTxt,
		lipgloss.NewStyle().Render(m.input.View()),
	)
	c.Y += lipgloss.Height(view)
	return view, c
}

func (m userModel) PasswordFailedView() string {
	yes, no := m.GetYesNo(m.consentPos)
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		passwordFailedTxt,
		no,
		yes,
	)
	return view
}

func (m userModel) CompletedView() string {
	return ""
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

func (m userModel) GetAuthToken() tea.Msg {
	userData := m.fetchedProfile.Data[0]
	tokenData, err := kitsu.GetAuthToken(userData.Attributes.Name, m.input.Value())
	if err != nil {
		return AuthTokenErrorMsg(err)
	}
	return FetchedAuthTokenMsg(tokenData)
}

func (m userModel) updateConsent(msg tea.Msg) userModel {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Down):
			m.consentPos = utils.AbsInt(m.consentPos-1) % 2

		case key.Matches(msg, m.keys.Up):
			m.consentPos = utils.AbsInt(m.consentPos+1) % 2

		}
	}

	return m
}

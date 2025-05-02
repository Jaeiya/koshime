package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type (
	FetchedAuthTokenMsg = kitsu.AuthToken
	FetchProfileMsg     = database.Profile
	FetchedLibAnimeMsg  = []database.LibraryEntry
	FetchErrorMsg       error
)

type ViewState int

const (
	ConsentView = ViewState(iota)
	UsernameView
	PasswordView
	LibraryAnimeView
	AbortView
	Completed
)

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Submit   key.Binding
	Abort    key.Binding
	HelpMore key.Binding
	HelpLess key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Submit:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	HelpMore: key.NewBinding(key.WithKeys("shift+/"), key.WithHelp("?", "more help")),
	HelpLess: key.NewBinding(key.WithKeys("shift+/"), key.WithHelp("?", "less help")),
	Abort:    key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "abort")),
}

func NewUser() (database.Data, bool) {
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

	p := tea.NewProgram(userModel{
		input:   input,
		help:    h,
		spinner: s,
	})

	m, err := p.Run()
	if err != nil {
		panic(err)
	}

	model := m.(userModel)
	return model.state.userData, model.state.isAborted
}

type state struct {
	consentPos int
	isAborted  bool
	userData   database.Data
	view       ViewState
	loading    struct {
		active bool
		text   string
	}
	username struct {
		failed bool
		passed bool
	}
	password struct {
		failed bool
		passed bool
	}
	libAnime struct {
		failed bool
		passed bool
	}
}

type userModel struct {
	help       help.Model
	input      textinput.Model
	spinner    spinner.Model
	fetchError error
	state      state
}

func (m userModel) Init() tea.Cmd {
	return nil
}

func (m userModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width

	case tea.KeyPressMsg:
		if key.Matches(msg, keys.Abort) {
			m.state.isAborted = true
			m.state.view = AbortView
			return m, tea.Quit
		}

		if key.Matches(msg, keys.HelpMore) {
			switch m.state.view {
			// Full help not implemented for input fields
			case UsernameView, PasswordView:
			default:
				m.help.ShowAll = !m.help.ShowAll
			}
		}

	}

	switch m.state.view {
	case ConsentView:
		m, cmd = m.UpdateUserConsent(msg)
	case UsernameView:
		m, cmd = m.UpdateUserName(msg)
	case PasswordView:
		m, cmd = m.UpdatePassword(msg)
	case LibraryAnimeView:
		m, cmd = m.UpdateLibAnime(msg)
	}

	cmds = append(cmds, cmd)

	if m.state.loading.active {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m userModel) View() (string, *tea.Cursor) {
	var c *tea.Cursor
	var view string
	switch m.state.view {
	case ConsentView:
		view = m.consentView()
	case UsernameView:
		view, c = m.usernameView()
	case PasswordView:
		view, c = m.passwordView()
	case LibraryAnimeView:
		view = m.libAnimeView()
	case Completed:
		view = ""
	case AbortView:
		view = m.abortView()
	}

	if c != nil {
		// Adjust for help text
		c.Y -= 1
	}

	helpView := textStyle.Height(3).PaddingTop(1).Render(m.help.View(m))
	return lipgloss.JoinVertical(lipgloss.Left, view, helpView), c
}

func (m userModel) ShortHelp() []key.Binding {
	switch m.state.view {
	case ConsentView:
		return []key.Binding{keys.Up, keys.Down, keys.Select, keys.HelpMore}

	case UsernameView:
		if m.state.username.failed || m.state.username.passed {
			return []key.Binding{keys.Up, keys.Down, keys.Select}
		}
		return []key.Binding{keys.Submit, keys.Abort}

	case PasswordView:
		if m.state.password.failed {
			return []key.Binding{keys.Up, keys.Down, keys.Select}
		}
		return []key.Binding{keys.Submit, keys.Abort}

	case LibraryAnimeView:
		if m.state.libAnime.passed {
			return []key.Binding{keys.Submit, keys.Abort}
		}
		return []key.Binding{keys.Up, keys.Down, keys.Select}
	}

	return []key.Binding{}
}

func (m userModel) FullHelp() [][]key.Binding {
	switch m.state.view {
	case ConsentView:
		return [][]key.Binding{
			{keys.Up, keys.Down, keys.Select},
			{keys.Abort, keys.HelpLess},
		}
	}
	return [][]key.Binding{}
}

func (m userModel) UpdateUserConsent(msg tea.Msg) (userModel, tea.Cmd) {
	m = m.updateConsent(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Select):
			if !m.isConsenting() {
				return m.abort()
			}
			m.state.view = UsernameView
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m userModel) UpdateUserName(msg tea.Msg) (userModel, tea.Cmd) {
	var cmd tea.Cmd
	state := &m.state.username

	if state.failed || state.passed {
		m = m.updateConsent(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			if !state.failed && !state.passed && !m.state.loading.active {
				m.state.loading.active = true
				m.state.loading.text = "Loading Profile"
				return m, tea.Batch(m.spinner.Tick, m.getProfile(m.input.Value()))
			}

			// User chooses to either abort or try again
			if state.failed {
				if !m.isConsenting() {
					return m.abort()
				}
				m.input.Reset()
				state.failed = false
			}

			// User chooses if profile is theirs or not
			if state.passed {
				if !m.isConsenting() {
					state.passed = false
					m.input.Reset()
					return m, nil
				}
				m.input.Reset()
				m.input.EchoMode = textinput.EchoPassword
				m.state.view = PasswordView
				return m, nil
			}
		}

	case FetchProfileMsg:
		m.state.loading.active = false
		m.state.userData.Profile = msg
		state.passed = true

	// If getting the profile returns an error
	case FetchErrorMsg:
		m.state.loading.active = false
		if strings.Contains(msg.Error(), "profile not found") {
			m.state.username.failed = true
			// Default to yes
			m.state.consentPos = 1
		} else {
			// FIX  we should display a proper error, not panic
			panic(msg)
		}

	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m userModel) UpdatePassword(msg tea.Msg) (userModel, tea.Cmd) {
	var cmd tea.Cmd
	state := &m.state.password

	if state.failed {
		m = m.updateConsent(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			if !state.failed && !state.passed && !m.state.loading.active {
				m.state.loading.active = true
				m.state.loading.text = "Getting Access Token"
				return m, tea.Batch(m.spinner.Tick, m.getAuthToken)
			}

			if state.failed {
				if !m.isConsenting() {
					return m.abort()
				}
				state.failed = false
				m.input.Reset()
			}

			if state.passed && !m.state.loading.active {
				m.state.loading.active = true
				m.state.loading.text = "Getting Library Anime"
				m.state.view = LibraryAnimeView
				return m, tea.Batch(m.spinner.Tick, m.getAnimeLibrary(m.state.userData.Profile.ID))
			}
		}

	case FetchedAuthTokenMsg:
		m.state.loading.active = false
		state.passed = true

		m.state.userData.Profile.AccessToken = msg.Token
		m.state.userData.Profile.RefreshToken = msg.RefreshToken
		m.state.userData.Profile.TokenExpiration = msg.ExpiresIn
		return m, nil

	case FetchErrorMsg:
		m.state.loading.active = false
		state.failed = true
		m.fetchError = msg
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m userModel) UpdateLibAnime(msg tea.Msg) (userModel, tea.Cmd) {
	state := &m.state.libAnime

	if state.failed {
		m = m.updateConsent(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, keys.Select) {
			if state.failed {
				if !m.isConsenting() {
					return m.abort()
				}
				m.state.loading.active = true
				state.failed = false
				return m, tea.Batch(m.spinner.Tick, m.getAnimeLibrary(m.state.userData.Profile.ID))
			}

			if state.passed {
				m.state.view = Completed
				return m, tea.Quit
			}
		}

	case FetchedLibAnimeMsg:
		m.state.userData.Library = msg
		state.passed = true
		m.state.loading.active = false

	case FetchErrorMsg:
		// Default to yes
		m.state.consentPos = 1
		m.state.loading.active = false
		state.failed = true
		m.fetchError = msg
	}
	return m, nil
}

func (m userModel) consentView() string {
	yes, no := m.getYesNo(m.state.consentPos)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		userWelcomeTxt,
		no,
		yes,
	)
}

func (m userModel) usernameView() (string, *tea.Cursor) {
	if m.state.loading.active {
		return m.loadingView(), nil
	}

	if m.state.username.failed {
		yes, no := m.getYesNo(m.state.consentPos)
		view := lipgloss.JoinVertical(
			lipgloss.Left,
			usernameFailedTxt,
			no,
			yes,
		)
		return view, nil
	}

	if m.state.username.passed {
		yes, no := m.getYesNo(m.state.consentPos)
		p := m.state.userData.Profile

		createdDate, err := time.Parse(time.RFC3339, p.CreatedAt)
		if err != nil {
			panic(err)
		}

		profileStr := createList([]string{
			"Name", "About", "Gender", "BirthDay", "Location", "Created", "Profile",
		}, []string{
			fmt.Sprintf(";g;%s", p.Username), p.About, p.Gender, p.Birthday, p.Location,
			createdDate.Local().Format("01/02/2006 3:04 PM"),
			kitsu.GetProfileLink(p.ID),
		})

		return lipgloss.JoinVertical(
			lipgloss.Left,
			confirmUsernamePreTxt,
			profileStr,
			confirmUsernameConsentTxt,
			no,
			yes,
		), nil
	}

	c := m.input.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		userNameTxt,
		style.MarginTop(1).Render(m.input.View()),
	)
	c.Y += lipgloss.Height(view)
	return view, c
}

func (m userModel) passwordView() (string, *tea.Cursor) {
	if m.state.loading.active {
		return m.loadingView(), nil
	}

	if m.state.password.failed {
		yes, no := m.getYesNo(m.state.consentPos)
		view := lipgloss.JoinVertical(
			lipgloss.Left,
			passwordFailedTxt,
			no,
			yes,
		)
		return view, nil
	}

	if m.state.password.passed {
		tokensStr := newText([]string{
			";c;Access Token",
			";bk;" + m.state.userData.Profile.AccessToken,
			"",
			";c;Refresh Token",
			";bk;" + m.state.userData.Profile.RefreshToken,
		}, false)
		header := lipgloss.NewStyle().Align(lipgloss.Center).
			Width(lipgloss.Width(tokensStr) - 3).
			Foreground(ansi.BrightBlue).
			PaddingBottom(1).
			Render("Your Token Credentials")

		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			textStyle.PaddingBottom(1).Render(tokensStr),
			textStyle.Foreground(ansi.BrightGreen).Render("> Continue"),
		), nil
	}

	c := m.input.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		passwordTxt,
		style.MarginTop(1).Render(m.input.View()),
	)
	c.Y += lipgloss.Height(view)
	return view, c
}

func (m userModel) libAnimeView() string {
	if m.state.loading.active {
		return m.loadingView()
	}

	if m.state.libAnime.failed {
		s := textStyle.MarginTop(1).Width(60).Render(m.fetchError.Error())
		yes, no := m.getYesNo(m.state.consentPos)
		view := lipgloss.JoinVertical(
			lipgloss.Left,
			s,
			libAnimeFetchFailedTxt,
			no,
			yes,
		)
		return view
	}

	if m.state.libAnime.passed {
		loadedStr := textStyle.PaddingBottom(1).
			Render(
				utils.ColorText(
					fmt.Sprintf(
						";b;Loaded ;w;%d ;b;Anime from your watch list",
						len(m.state.userData.Library),
					),
				),
			)
		continueStr := textStyle.
			Foreground(ansi.BrightGreen).
			Render("> Continue")
		return lipgloss.JoinVertical(lipgloss.Left, loadedStr, continueStr)
	}

	return ""
}

func (m userModel) loadingView() string {
	spinnerStr := spinnerStyle.Render(strings.Repeat(m.spinner.View(), 3))
	return textStyle.Render(
		fmt.Sprintf(
			"%s %s %s",
			spinnerStr,
			loadingStyle.Render(m.state.loading.text),
			spinnerStr,
		),
	)
}

func (m userModel) abortView() string {
	return abortStyle.Render(utils.ColorText(";g;>>> ;y;Koshime Setup Aborted ;g;<<<;x;"))
}

func (m userModel) abort() (userModel, tea.Cmd) {
	m.state.isAborted = true
	m.state.view = AbortView
	return m, tea.Quit
}

func (m userModel) getYesNo(state int) (yes string, no string) {
	if state == 0 {
		no = selectNoStyle.Render("> No")
		yes = textStyle.Render("  Yes")
	} else {
		yes = selectYesStyle.Render("> Yes")
		no = textStyle.MarginTop(1).Render("  No")
	}
	return yes, no
}

func (m *userModel) isConsenting() bool {
	hasConsented := m.state.consentPos == 1
	if hasConsented {
		m.state.consentPos = 0
	}
	return hasConsented
}

func (m userModel) updateConsent(msg tea.Msg) userModel {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Down):
			m.state.consentPos = utils.AbsInt(m.state.consentPos-1) % 2

		case key.Matches(msg, keys.Up):
			m.state.consentPos = utils.AbsInt(m.state.consentPos+1) % 2
		}
	}

	return m
}

func (m userModel) getProfile(userName string) func() tea.Msg {
	return func() tea.Msg {
		p, err := kitsu.GetProfile(userName)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return FetchProfileMsg(p)
	}
}

func (m userModel) getAuthToken() tea.Msg {
	tokenData, err := kitsu.GetAuthToken(m.state.userData.Profile.Username, m.input.Value())
	if err != nil {
		return FetchErrorMsg(err)
	}
	return tokenData
}

func (m userModel) getAnimeLibrary(userID string) func() tea.Msg {
	return func() tea.Msg {
		data, err := kitsu.GetLibraryAnime(userID, kitsu.LibAnimeWatching)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return data
	}
}

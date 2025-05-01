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

var (
	questionStyle = defaultTextStyle.Foreground(ansi.BrightWhite)
	selectedStyle = defaultTextStyle.Foreground(ansi.BrightMagenta)
	loadingStyle  = lipgloss.NewStyle().Foreground(ansi.BrightBlue)
	spinnerStyle  = lipgloss.NewStyle().Foreground(ansi.BrightGreen)

	selectedYes = defaultTextStyle.Foreground(ansi.BrightGreen).Render("> Yes")
	selectedNo  = defaultTextStyle.MarginTop(1).Foreground(ansi.BrightMagenta).Render("> No")
)

type (
	FetchedAuthTokenMsg kitsu.AuthToken
	FetchedLibAnimeMsg  []database.LibraryEntry
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

	s := spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"},
		FPS:    time.Second / 10,
	}))

	p := tea.NewProgram(userModel{
		input:     input,
		help:      h,
		db:        database.Data{},
		spinner:   s,
		viewState: ConsentView,
	})
	m, err := p.Run()
	if err != nil {
		panic(err)
	}

	return m.(userModel).db, m.(userModel).isAborted
}

type userModel struct {
	help       help.Model
	input      textinput.Model
	spinner    spinner.Model
	consentPos int
	state      struct {
		profile  kitsu.ProfileData
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
	db          database.Data
	password    string
	isLoading   bool
	loadingText string
	isAborted   bool
	fetchError  error
	viewState   ViewState
}

func (m userModel) ShortHelp() []key.Binding {
	switch m.viewState {
	case ConsentView:
		return []key.Binding{
			keys.Up, keys.Down, keys.Select, keys.HelpMore,
		}

	case UsernameView:
		state := m.state.username
		if state.failed || state.passed {
			return []key.Binding{
				keys.Up, keys.Down, keys.Select,
			}
		}
		return []key.Binding{
			keys.Submit, keys.Abort,
		}

	case PasswordView:
		state := m.state.password
		if state.failed {
			return []key.Binding{
				keys.Up, keys.Down, keys.Select,
			}
		}
		return []key.Binding{
			keys.Submit, keys.Abort,
		}

	case LibraryAnimeView:
		state := m.state.libAnime
		if state.passed {
			return []key.Binding{
				keys.Submit, keys.Abort,
			}
		}
		return []key.Binding{
			keys.Up, keys.Down, keys.Select,
		}
	}

	return []key.Binding{}
}

func (m userModel) FullHelp() [][]key.Binding {
	switch m.viewState {
	case ConsentView:
		return [][]key.Binding{
			{keys.Up, keys.Down, keys.Select},
			{keys.Abort, keys.HelpLess},
		}
	case UsernameView, PasswordView, LibraryAnimeView:
		// All help is already shown with short help
	}
	return [][]key.Binding{}
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
			m.isAborted = true
			m.viewState = AbortView
			return m, tea.Quit
		}

		if key.Matches(msg, keys.HelpMore) {
			switch m.viewState {
			// Full help not implemented for input fields
			case UsernameView, PasswordView:
			default:
				m.help.ShowAll = !m.help.ShowAll
			}
		}

	}

	switch m.viewState {
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

	if m.isLoading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m userModel) UpdateUserConsent(msg tea.Msg) (userModel, tea.Cmd) {
	m = m.updateConsent(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Select):
			hasConsent := m.getConsentSelection()
			if !hasConsent {
				m.viewState = AbortView
				m.isAborted = true
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
	state := &m.state.username

	if state.failed || state.passed {
		m = m.updateConsent(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			if !state.failed && !state.passed && !m.isLoading {
				m.isLoading = true
				m.loadingText = "Loading Profile"
				return m, tea.Batch(m.spinner.Tick, m.getProfile(m.input.Value()))
			}

			// User chooses to either abort or try again
			if state.failed {
				hasConsented := m.getConsentSelection()
				if !hasConsented {
					return m.abort()
				}
				m.input.Reset()
				state.failed = false
			}

			// User chooses if profile is theirs or not
			if state.passed {
				hasConsented := m.getConsentSelection()
				if !hasConsented {
					state.passed = false
					m.input.Reset()
					return m, nil
				}
				m.input.Reset()
				m.input.EchoMode = textinput.EchoPassword
				m.viewState = PasswordView
				return m, nil
			}
		}

	case kitsu.ProfileData:
		m.isLoading = false
		m.state.profile = msg
		if len(m.state.profile.Data) == 0 {
			// Start on Yes
			m.consentPos = 1
			state.failed = true
		} else {
			state.passed = true
		}

	// If getting the profile returns an error
	case FetchErrorMsg:
		panic(msg)

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
			if !state.failed && !state.passed && !m.isLoading {
				m.isLoading = true
				m.loadingText = "Getting Access Token"
				return m, tea.Batch(m.spinner.Tick, m.getAuthToken)
			}

			if state.failed {
				hasConsented := m.getConsentSelection()
				if !hasConsented {
					return m.abort()
				}
				state.failed = false
				m.input.Reset()
			}

			if state.passed && !m.isLoading {
				m.isLoading = true
				m.loadingText = "Getting Library Anime"
				m.viewState = LibraryAnimeView
				return m, tea.Batch(m.spinner.Tick, m.getAnimeLibrary(m.db.Profile.ID))
			}
		}

	case FetchedAuthTokenMsg:
		m.isLoading = false
		state.passed = true

		m.db.Profile.AccessToken = msg.Token
		m.db.Profile.RefreshToken = msg.RefreshToken
		m.db.Profile.TokenExpiration = msg.ExpiresIn

		userData := m.state.profile.Data[0]
		userStats := m.state.profile.Included[0]

		m.db.Profile.ID = userData.ID
		m.db.Profile.Username = userData.Attributes.Name
		m.db.Profile.CompletedSeries = userStats.Attributes.Stats.CompletedAnime
		m.db.Profile.SecondsWatched = userStats.Attributes.Stats.SecondsWatched
		return m, nil

	case FetchErrorMsg:
		m.isLoading = false
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
				hasConsented := m.getConsentSelection()
				if !hasConsented {
					return m.abort()
				}
				m.isLoading = true
				state.failed = false
				return m, tea.Batch(m.spinner.Tick, m.getAnimeLibrary(m.db.Profile.ID))
			}

			if state.passed {
				m.viewState = Completed
				return m, tea.Quit
			}
		}

	case FetchedLibAnimeMsg:
		m.db.Library = msg
		state.passed = true
		m.isLoading = false

	case FetchErrorMsg:
		// Default to yes
		m.consentPos = 1
		m.isLoading = false
		state.failed = true
		m.fetchError = msg
	}
	return m, nil
}

func (m userModel) View() (string, *tea.Cursor) {
	var c *tea.Cursor
	var view string
	switch m.viewState {
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

	helpView := defaultTextStyle.Height(3).PaddingTop(1).Render(m.help.View(m))
	return lipgloss.JoinVertical(lipgloss.Left, view, helpView), c
}

func (m userModel) consentView() string {
	yes, no := m.getYesNo(m.consentPos)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		userWelcomeTxt,
		userConsentTxt,
		no,
		yes,
	)
}

func (m userModel) usernameView() (string, *tea.Cursor) {
	if m.isLoading {
		return m.loadingView(), nil
	}

	profile := m.state.profile

	if m.state.username.failed {
		yes, no := m.getYesNo(m.consentPos)
		view := lipgloss.JoinVertical(
			lipgloss.Left,
			usernameFailedTxt,
			no,
			yes,
		)
		return view, nil
	}

	if m.state.username.passed {
		yes, no := m.getYesNo(m.consentPos)
		userID := profile.Data[0].ID
		attr := profile.Data[0].Attributes

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
		), nil
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

func (m userModel) passwordView() (string, *tea.Cursor) {
	if m.isLoading {
		return m.loadingView(), nil
	}

	if m.state.password.failed {
		yes, no := m.getYesNo(m.consentPos)
		view := lipgloss.JoinVertical(
			lipgloss.Left,
			passwordFailedTxt,
			no,
			yes,
		)
		return view, nil
	}

	if m.state.password.passed {
		content := utils.ColorText(fmt.Sprintf(
			";c;Access Token\n;bk;%s\n\n;c;Refresh Token\n;bk;%s",
			m.db.Profile.AccessToken,
			m.db.Profile.RefreshToken,
		))
		header := defaultTextStyle.Align(lipgloss.Center).
			Width(lipgloss.Width(content)).
			Foreground(ansi.BrightBlue).
			PaddingBottom(1).
			Render("Your Token Credentials")

		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			defaultTextStyle.Width(60).PaddingBottom(1).Render(content),
			defaultTextStyle.Foreground(ansi.BrightGreen).Render("> Continue"),
		), nil
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

func (m userModel) libAnimeView() string {
	if m.isLoading {
		return m.loadingView()
	}

	if m.state.libAnime.failed {
		s := defaultTextStyle.MarginTop(1).Width(60).Render(m.fetchError.Error())
		yes, no := m.getYesNo(m.consentPos)
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
		loadedStr := defaultTextStyle.PaddingBottom(1).
			Render(
				utils.ColorText(
					fmt.Sprintf(
						";b;Loaded ;w;%d ;b;Anime from your watch list",
						len(m.db.Library),
					),
				),
			)
		continueStr := defaultTextStyle.
			Foreground(ansi.BrightGreen).
			Render("> Continue")
		return lipgloss.JoinVertical(lipgloss.Left, loadedStr, continueStr)
	}

	return ""
}

func (m userModel) loadingView() string {
	spinnerStr := spinnerStyle.Render(strings.Repeat(m.spinner.View(), 3))
	return defaultTextStyle.Render(
		fmt.Sprintf("%s %s %s", spinnerStr, loadingStyle.Render(m.loadingText), spinnerStr),
	)
}

func (m userModel) abortView() string {
	return lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Render(utils.ColorText(";g;>>> ;y;Koshime Setup Aborted ;g;<<<;x;"))
}

func (m userModel) abort() (userModel, tea.Cmd) {
	m.isAborted = true
	m.viewState = AbortView
	return m, tea.Quit
}

func (m userModel) getYesNo(state int) (yes string, no string) {
	if state == 0 {
		no = selectedNo
		yes = defaultTextStyle.Render(" Yes")
	} else {
		yes = selectedYes
		no = defaultTextStyle.MarginTop(1).Render(" No")
	}
	return yes, no
}

func (m *userModel) getConsentSelection() bool {
	hasConsented := m.consentPos == 1
	if hasConsented {
		m.consentPos = 0
	}
	return hasConsented
}

func (m userModel) updateConsent(msg tea.Msg) userModel {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Down):
			m.consentPos = utils.AbsInt(m.consentPos-1) % 2

		case key.Matches(msg, keys.Up):
			m.consentPos = utils.AbsInt(m.consentPos+1) % 2
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
		return p
	}
}

func (m userModel) getAuthToken() tea.Msg {
	userData := m.state.profile.Data[0]
	tokenData, err := kitsu.GetAuthToken(userData.Attributes.Name, m.input.Value())
	if err != nil {
		return FetchErrorMsg(err)
	}
	return FetchedAuthTokenMsg(tokenData)
}

func (m userModel) getAnimeLibrary(userID string) func() tea.Msg {
	return func() tea.Msg {
		data, err := kitsu.GetLibraryAnime(userID, kitsu.LibAnimeWatching)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return FetchedLibAnimeMsg(data)
	}
}

package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
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

type userSetupView int

const (
	SetupConsentView = userSetupView(iota)
	SetupUsernameView
	SetupPasswordView
	SetupLibraryView
)

type userSetupKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Submit   key.Binding
	Abort    key.Binding
	HelpMore key.Binding
	HelpLess key.Binding
}

type userSetupState struct {
	data       database.Data
	view       userSetupView
	fetchError error
	username   struct {
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

func (m UIModel) UpdateUserSetup(msg tea.Msg) (UIModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	state := m.state.userSetup

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width

	case tea.KeyPressMsg:
		if key.Matches(msg, keyMap.HelpMore) {
			switch state.view {
			// Full help not implemented for input fields
			case SetupUsernameView, SetupPasswordView:
			default:
				m.help.ShowAll = !m.help.ShowAll
			}
		}

	}

	switch state.view {
	case SetupConsentView:
		m, cmd = m.updateUserConsent(msg)
	case SetupUsernameView:
		m, cmd = m.updateUserName(msg)
	case SetupPasswordView:
		m, cmd = m.updatePassword(msg)
	case SetupLibraryView:
		m, cmd = m.updateLibAnime(msg)
	}

	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m UIModel) ViewUserSetup() (string, *tea.Cursor) {
	var c *tea.Cursor
	var view string
	switch m.state.userSetup.view {
	case SetupConsentView:
		view = m.viewConsent(userWelcomeTxt)
	case SetupUsernameView:
		view, c = m.viewSetupUsername()
	case SetupPasswordView:
		view, c = m.viewSetupPassword()
	case SetupLibraryView:
		view = m.viewSetupLibrary()
	}

	// Always keep margin from prompt
	view = style.MarginTop(1).Render(view)
	helpView := textStyle.Height(3).PaddingTop(1).Render(m.help.View(m))

	if c != nil {
		// Adjust for margin and help text
		c.Y += 1
	}
	return lipgloss.JoinVertical(lipgloss.Left, style.MarginTop(1).Render(view), helpView), c
}

func (m UIModel) updateUserConsent(msg tea.Msg) (UIModel, tea.Cmd) {
	m = m.updateConsent(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.Select):
			if !m.isConsenting() {
				return m, m.abort
			}
			m.state.userSetup.view = SetupUsernameView
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m UIModel) updateUserName(msg tea.Msg) (UIModel, tea.Cmd) {
	var cmd tea.Cmd
	state := &m.state.userSetup

	if state.username.failed || state.username.passed {
		m = m.updateConsent(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			if !state.username.failed && !state.username.passed && !m.isLoading() {
				m.setLoadingState(true)
				m.setLoadingText("Loading Profile")
				return m, tea.Batch(m.spinner.Tick, m.getProfile(m.input.Value()))
			}

			// User chooses to either abort or try again
			if state.username.failed {
				if !m.isConsenting() {
					return m, m.abort
				}
				m.input.Reset()
				state.username.failed = false
			}

			// User chooses if profile is theirs or not
			if state.username.passed {
				if !m.isConsenting() {
					state.username.passed = false
					m.input.Reset()
					return m, nil
				}
				m.input.Reset()
				m.input.EchoMode = textinput.EchoPassword
				state.view = SetupPasswordView
				return m, nil
			}
		}

	case FetchProfileMsg:
		m.setLoadingState(false)
		state.data.Profile = msg
		state.username.passed = true

	// If getting the profile returns an error
	case FetchErrorMsg:
		m.setLoadingState(false)
		if strings.Contains(msg.Error(), "profile not found") {
			state.username.failed = true
			m.setConsentStartPos(Yes)
		} else {
			// FIX  we should display a proper error, not panic
			panic(msg)
		}

	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m UIModel) userSetupShortHelp() []key.Binding {
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

func (m UIModel) userSetupFullHelp() [][]key.Binding {
	switch m.state.userSetup.view {
	case SetupConsentView:
		return [][]key.Binding{
			{keyMap.Up, keyMap.Down, keyMap.Select},
			{keyMap.Abort, keyMap.HelpLess},
		}
	}
	return [][]key.Binding{}
}

func (m UIModel) updatePassword(msg tea.Msg) (UIModel, tea.Cmd) {
	var cmd tea.Cmd
	state := &m.state.userSetup

	if state.password.failed {
		m = m.updateConsent(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			if !state.password.failed && !state.password.passed && !m.isLoading() {
				m.setLoadingState(true)
				m.setLoadingText("Getting Access Token")
				return m, tea.Batch(m.spinner.Tick, m.getAuthToken)
			}

			if state.password.failed {
				if !m.isConsenting() {
					return m, m.abort
				}
				state.password.failed = false
				m.input.Reset()
			}

			if state.password.passed && !m.isLoading() {
				m.setLoadingState(true)
				m.setLoadingText("Getting Library Anime")
				state.view = SetupLibraryView
				return m, tea.Batch(m.spinner.Tick, m.getAnimeLibrary(state.data.Profile.ID))
			}
		}

	case FetchedAuthTokenMsg:
		m.setLoadingState(false)
		state.password.passed = true

		state.data.Profile.AccessToken = msg.Token
		state.data.Profile.RefreshToken = msg.RefreshToken
		state.data.Profile.TokenExpiration = msg.ExpiresIn
		return m, nil

	case FetchErrorMsg:
		m.setLoadingState(false)
		state.password.failed = true
		state.fetchError = msg
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m UIModel) updateLibAnime(msg tea.Msg) (UIModel, tea.Cmd) {
	state := &m.state.userSetup

	if state.libAnime.failed {
		m = m.updateConsent(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, keyMap.Select) {
			if state.libAnime.failed {
				if !m.isConsenting() {
					return m, m.abort
				}
				m.setLoadingState(true)
				state.libAnime.failed = false
				return m, tea.Batch(m.spinner.Tick, m.getAnimeLibrary(state.data.Profile.ID))
			}

			if state.libAnime.passed {
				return m, func() tea.Msg { return SetupUserFinishedMsg{} }
			}
		}

	case FetchedLibAnimeMsg:
		state.data.Library = msg
		state.libAnime.passed = true
		m.setLoadingState(false)

	case FetchErrorMsg:
		m.setConsentStartPos(Yes)
		m.setLoadingState(false)
		state.libAnime.failed = true
		state.fetchError = msg
	}
	return m, nil
}

func (m UIModel) viewSetupUsername() (string, *tea.Cursor) {
	state := m.state.userSetup
	if m.isLoading() {
		return m.viewLoading(), nil
	}

	if state.username.failed {
		return m.viewConsent(usernameFailedTxt), nil
	}

	if state.username.passed {
		p := state.data.Profile

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

		return m.viewConsent(confirmUsernamePreTxt, profileStr, confirmUsernameConsentTxt), nil
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

func (m UIModel) viewSetupPassword() (string, *tea.Cursor) {
	state := m.state.userSetup
	if m.isLoading() {
		return m.viewLoading(), nil
	}

	if state.password.failed {
		return m.viewConsent(passwordFailedTxt), nil
	}

	if state.password.passed {
		tokensStr := newText([]string{
			";c;Access Token",
			";bk;" + state.data.Profile.AccessToken,
			"",
			";c;Refresh Token",
			";bk;" + state.data.Profile.RefreshToken,
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

func (m UIModel) viewSetupLibrary() string {
	state := m.state.userSetup
	if m.isLoading() {
		return m.viewLoading()
	}

	if state.libAnime.failed {
		s := textStyle.MarginTop(1).Width(60).Render(state.fetchError.Error())
		return m.viewConsent(s, libAnimeFetchFailedTxt)
	}

	if state.libAnime.passed {
		loadedStr := textStyle.PaddingBottom(1).
			Render(
				utils.ColorText(
					fmt.Sprintf(
						";b;Loaded ;w;%d ;b;Anime from your watch list",
						len(state.data.Library),
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

func (m UIModel) getProfile(userName string) func() tea.Msg {
	return func() tea.Msg {
		p, err := kitsu.GetProfile(userName)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return FetchProfileMsg(p)
	}
}

func (m UIModel) getAuthToken() tea.Msg {
	tokenData, err := kitsu.GetAuthToken(
		m.state.userSetup.data.Profile.Username,
		m.input.Value(),
	)
	if err != nil {
		return FetchErrorMsg(err)
	}
	return tokenData
}

func (m UIModel) getAnimeLibrary(userID string) func() tea.Msg {
	return func() tea.Msg {
		data, err := kitsu.GetLibraryAnime(userID, kitsu.LibAnimeWatching)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return data
	}
}

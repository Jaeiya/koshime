package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type (
	FetchedAuthTokenMsg  = kitsu.AuthTokenData
	FetchProfileMsg      = kitsu.Profile
	FetchedLibAnimeMsg   = []kitsu.LibraryEntry
	SetupUserFinishedMsg = database.Data
)

type UserSetupView int

const (
	SetupConsentView = UserSetupView(iota)
	SetupUsernameView
	SetupPasswordView
	SetupLibraryView
)

type UserSetupModel struct {
	consent ui.ConsentModel
	loader  ui.LoaderModel
	input   textinput.Model
	state   struct {
		data       database.Data
		view       UserSetupView
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
}

func newUserSetupModel() UserSetupModel {
	return UserSetupModel{
		loader: ui.NewLoader(),
		input:  ui.NewTextInput(),
	}
}

func (m UserSetupModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Abort):
			return m, abort
		}
	}

	if m.loader.IsLoading() {
		m.loader, cmd = m.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case SetupConsentView:
		m, cmd = m.UpdateConsent(msg)
		cmds = append(cmds, cmd)
	case SetupUsernameView:
		m, cmd = m.UpdateUsername(msg)
		cmds = append(cmds, cmd)
	case SetupPasswordView:
		m, cmd = m.UpdatePassword(msg)
		cmds = append(cmds, cmd)
	case SetupLibraryView:
		m, cmd = m.UpdateLibAnime(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m UserSetupModel) View() (string, *tea.Cursor) {
	var view string
	var c *tea.Cursor

	switch m.state.view {
	case SetupConsentView:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			userSetupMsgs.header,
			userSetupMsgs.welcome,
			m.consent.View(
				ui.ConsentStyle.Render("Would you like to setup in this directory?"),
			),
		), nil
	case SetupUsernameView:
		view, c = m.ViewUsername()
	case SetupPasswordView:
		view, c = m.ViewPassword()
	case SetupLibraryView:
		view = m.ViewLibAnime()
	}

	view = ui.Style.MarginTop(1).Render(view)
	if c != nil {
		// -1 for the text input
		// (this obviously breaks for more than one input)
		c.Y += lipgloss.Height(view) - 1
	}
	return view, c
}

func (m UserSetupModel) UpdateConsent(msg tea.Msg) (UserSetupModel, tea.Cmd) {
	m.consent = m.consent.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if ui.No == m.consent.Select() {
				return m, abort
			}
			m.state.view = SetupUsernameView
			return m, textinput.Blink
		}
	}

	return m, nil
}

func (m UserSetupModel) UpdateUsername(msg tea.Msg) (UserSetupModel, tea.Cmd) {
	var cmd tea.Cmd
	usernameState := &m.state.username

	// Ignore all updates when we encounter an
	// unrecoverable error
	if m.state.fetchError != nil {
		return m, nil
	}

	if usernameState.failed || usernameState.passed {
		m.consent = m.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			if !usernameState.failed && !usernameState.passed && !m.loader.IsLoading() {
				m.loader, cmd = m.loader.Start("Loading Profile")
				return m, tea.Batch(cmd, m.getProfile(m.input.Value()))
			}

			// User chooses to either abort or try again
			if usernameState.failed {
				if ui.No == m.consent.Select() {
					return m, abort
				}
				m.input.Reset()
				usernameState.failed = false
			}

			// User chooses if profile is theirs or not
			if usernameState.passed {
				if ui.No == m.consent.Select() {
					usernameState.passed = false
					m.input.Reset()
					return m, nil
				}
				m.input.Reset()
				m.input.EchoMode = textinput.EchoPassword
				m.state.view = SetupPasswordView
				return m, nil
			}
		}

	case FetchProfileMsg:
		m.loader.Stop()
		m.state.data.Profile = msg
		usernameState.passed = true

	// If getting the profile returns an error
	case FetchErrorMsg:
		m.loader.Stop()
		if strings.Contains(msg.Error(), "profile not found") {
			usernameState.failed = true
			m.consent.SetConsentPos(ui.Yes)
		} else {
			m.state.fetchError = msg
		}

	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m UserSetupModel) ViewUsername() (string, *tea.Cursor) {
	if m.loader.IsLoading() {
		return m.loader.View(), nil
	}

	if m.state.username.failed {
		return m.consent.View(userSetupMsgs.username.failed), nil
	}

	if m.state.fetchError != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			ui.TextStyle.Foreground(ansi.BrightRed).Render("Fetch Error"),
			ui.TextStyle.PaddingLeft(2).
				Foreground(ansi.BrightYellow).
				Render(m.state.fetchError.Error()),
		), nil
	}

	if m.state.username.passed {
		p := m.state.data.Profile

		createdDate, err := time.Parse(time.RFC3339, p.CreatedAt)
		if err != nil {
			panic(err)
		}

		profileStr := ui.DisplayPropVal([]string{
			"Name", "About", "Gender", "BirthDay", "Location", "Created", "Profile",
		}, []string{
			fmt.Sprintf(";g;%s", p.Username), p.About, p.Gender, p.Birthday, p.Location,
			createdDate.Local().Format("01/02/2006 3:04 PM"),
			kitsu.GetProfileLink(p.ID),
		})

		return m.consent.View(
			userSetupMsgs.username.confirmHeader,
			profileStr,
			ui.Style.PaddingTop(1).Render(userSetupMsgs.username.confirmConsent),
		), nil
	}

	c := m.input.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		userSetupMsgs.username.enter,
		m.input.View(),
	)
	return view, c
}

func (m UserSetupModel) UpdatePassword(msg tea.Msg) (UserSetupModel, tea.Cmd) {
	var cmd tea.Cmd
	passwordState := &m.state.password
	if m.state.password.failed {
		m.consent = m.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			if !passwordState.failed && !passwordState.passed && !m.loader.IsLoading() {
				m.loader, cmd = m.loader.Start("Getting Access Token")
				return m, tea.Batch(cmd, m.getAuthToken)
			}

			if passwordState.failed {
				if ui.No == m.consent.Select() {
					return m, abort
				}
				passwordState.failed = false
				m.input.Reset()
			}

			if passwordState.passed && !m.loader.IsLoading() {
				m.loader, cmd = m.loader.Start("Getting Library Anime")
				m.state.view = SetupLibraryView
				return m, tea.Batch(cmd, m.getAnimeLibrary(m.state.data.Profile.ID))
			}
		}

	case FetchedAuthTokenMsg:
		m.loader.Stop()
		passwordState.passed = true
		p := &m.state.data.Profile
		p.AccessToken = msg.Token
		p.RefreshToken = msg.RefreshToken
		// Because this is a duration and not a time stamp, we stamp
		// it ourselves by adding the unix time.
		p.TokenExpirationSec = int64(msg.ExpiresIn) + time.Now().Unix()
		return m, nil

	case FetchErrorMsg:
		m.loader.Stop()
		passwordState.failed = true
		m.state.fetchError = msg
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m UserSetupModel) ViewPassword() (string, *tea.Cursor) {
	if m.loader.IsLoading() {
		return m.loader.View(), nil
	}

	if m.state.password.failed {
		return m.consent.View(userSetupMsgs.password.failed), nil
	}

	if m.state.password.passed {
		tokensStr := ui.DisplayText([]string{
			";c;Access Token",
			";bk;" + m.state.data.Profile.AccessToken,
			"",
			";c;Refresh Token",
			";bk;" + m.state.data.Profile.RefreshToken,
		})
		header := lipgloss.NewStyle().Align(lipgloss.Center).
			Width(lipgloss.Width(tokensStr) - 3).
			Foreground(ansi.BrightBlue).
			PaddingBottom(1).
			Render("Your Token Credentials")

		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			ui.TextStyle.PaddingBottom(1).Render(tokensStr),
			ui.TextStyle.Foreground(ansi.BrightGreen).Render("> Continue"),
		), nil
	}

	c := m.input.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		userSetupMsgs.password.enter,
		m.input.View(),
	)
	return view, c
}

func (m UserSetupModel) UpdateLibAnime(msg tea.Msg) (UserSetupModel, tea.Cmd) {
	state := &m.state.libAnime

	if state.failed {
		m.consent = m.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, ui.KeyMap.Select) {
			if state.failed {
				if ui.No == m.consent.Select() {
					return m, abort
				}
				// Retry getting library anime
				var cmd tea.Cmd
				m.loader, cmd = m.loader.Start("")
				state.failed = false
				return m, tea.Batch(cmd, m.getAnimeLibrary(m.state.data.Profile.ID))
			}

			if state.passed {
				return m, func() tea.Msg { return SetupUserFinishedMsg(m.state.data) }
			}
		}

	case FetchedLibAnimeMsg:
		m.loader.Stop()
		m.state.data.Library = msg
		state.passed = true

	case FetchErrorMsg:
		m.loader.Stop()
		m.consent.SetConsentPos(ui.Yes)
		state.failed = true
		m.state.fetchError = msg
	}
	return m, nil
}

func (m UserSetupModel) ViewLibAnime() string {
	if m.loader.IsLoading() {
		return m.loader.View()
	}

	if m.state.libAnime.failed {
		s := ui.TextStyle.MarginTop(1).Width(60).Render(m.state.fetchError.Error())
		return m.consent.View(s, userSetupMsgs.libAnime.failed)
	}

	if m.state.libAnime.passed {
		loadedStr := ui.TextStyle.PaddingBottom(1).
			Render(
				utils.ColorText(
					fmt.Sprintf(
						";b;Loaded ;w;%d ;b;Anime from your watch list",
						len(m.state.data.Library),
					),
				),
			)
		continueStr := ui.TextStyle.
			Foreground(ansi.BrightGreen).
			Render("> Continue")
		return lipgloss.JoinVertical(lipgloss.Left, loadedStr, continueStr)
	}

	return ""
}

func (m UserSetupModel) ShortHelp() []key.Binding {
	if m.loader.IsLoading() {
		return []key.Binding{}
	}

	if m.state.fetchError != nil {
		return []key.Binding{ui.KeyMap.Abort}
	}

	switch m.state.view {
	case SetupConsentView:
		return []key.Binding{
			ui.KeyMap.Up,
			ui.KeyMap.Down,
			ui.KeyMap.Select,
			ui.KeyMap.HelpMore,
		}

	case SetupUsernameView:
		if m.state.username.failed || m.state.username.passed {
			return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
		}
		return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.Abort}

	case SetupPasswordView:
		if m.state.password.failed {
			return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
		}
		return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.Abort}

	case SetupLibraryView:
		if m.state.libAnime.passed {
			return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.Abort}
		}
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
	}

	return []key.Binding{}
}

func (m UserSetupModel) FullHelp() [][]key.Binding {
	switch m.state.view {
	case SetupConsentView:
		return [][]key.Binding{
			{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select},
			{ui.KeyMap.Abort, ui.KeyMap.HelpLess},
		}
	}
	return [][]key.Binding{}
}

func (m UserSetupModel) getProfile(username string) tea.Cmd {
	return func() tea.Msg {
		p, err := kitsu.GetProfile(username)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return FetchProfileMsg(p)
	}
}

func (m UserSetupModel) getAnimeLibrary(userID string) func() tea.Msg {
	return func() tea.Msg {
		data, err := kitsu.GetLibraryAnime(userID, kitsu.LibAnimeWatching)
		if err != nil {
			return FetchErrorMsg(err)
		}
		return data
	}
}

func (m UserSetupModel) getAuthToken() tea.Msg {
	tokenData, err := kitsu.GetAuthToken(
		m.state.data.Profile.Username,
		m.input.Value(),
	)
	if err != nil {
		return FetchErrorMsg(err)
	}
	return tokenData
}

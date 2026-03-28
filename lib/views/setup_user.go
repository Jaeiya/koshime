package views

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	KitsuProfileMsg      = kitsu.Profile
	FetchedLibAnimeMsg   = []kitsu.LibraryEntry
	SetupUserFinishedMsg = database.Data
	SetupUserAbortMsg    struct{}
)

type SetupUserHelpMap map[SetupUserView]ui.KeyHelpInfo[SetupUserModel]

type SetupUserView int

const (
	SetupConsentView = SetupUserView(iota)
	SetupUsernameView
	SetupPasswordView
	SetupLibraryView
)

type SetupUserModel struct {
	ui struct {
		consent  ui.ConsentModel
		loader   ui.LoaderModel
		username textinput.Model
		password textinput.Model
	}
	helpMap SetupUserHelpMap
	err     error
	state   struct {
		data     database.Data
		view     SetupUserView
		err      error
		username struct {
			notFound bool
			found    bool
		}
		password struct {
			failed  bool
			success bool
		}
		libAnime struct {
			failed bool
			passed bool
		}
	}
}

func newSetupUserModel() SetupUserModel {
	m := SetupUserModel{}
	m.ui.loader = ui.NewLoader()

	m.ui.username = ui.NewTextInput()
	m.ui.username.Placeholder = "<profile-url-name>"

	m.ui.password = ui.NewTextInput()
	m.ui.password.EchoMode = textinput.EchoPassword
	m.ui.password.Placeholder = "<password>"

	m.helpMap = SetupUserHelpMap{
		SetupConsentView: {
			ShortHelp: func(usm SetupUserModel) []key.Binding {
				return []key.Binding{
					ui.KeyMap.Up,
					ui.KeyMap.Down,
					ui.KeyMap.Select,
					ui.KeyMap.Abort,
				}
			},
		},
		SetupUsernameView: {
			ShortHelp: func(usm SetupUserModel) []key.Binding {
				if m.state.username.notFound || m.state.username.found {
					return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
				}
				return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.Abort}
			},
		},
		SetupPasswordView: {
			ShortHelp: func(usm SetupUserModel) []key.Binding {
				if m.state.password.failed {
					return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
				}
				return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.Abort}
			},
		},
		SetupLibraryView: {
			ShortHelp: func(usm SetupUserModel) []key.Binding {
				if m.state.libAnime.passed {
					return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.Abort}
				}
				return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
			},
		},
	}

	return m
}

func (m SetupUserModel) Update(msg tea.Msg) (SetupUserModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Abort):
			// Do allow aborting on successful user setup
			if m.state.view == SetupLibraryView && m.state.libAnime.passed {
				break
			}
			m.deleteWatchDir()
			return m, abort
		}

	case SetupUserAbortMsg:
		m.deleteWatchDir()
		return m, abort

	case DefaultErrorMsg:
		m.err = msg
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
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

func (m SetupUserModel) View() (string, *tea.Cursor) {
	var view string
	var c *tea.Cursor

	if m.err != nil {
		view := lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle("User Setup", "Error"),
			ui.DisplayError(m.err),
		)
		return view, nil
	}

	switch m.state.view {
	case SetupConsentView:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("User Setup"),
			ui.DisplayText([]string{
				`Welcome to ;g;Koshime;x;!`,
				`You'll be able to easily ;dc;add;x;, ;dc;update;x;, and ;dc;watch;x;
your anime, from the command line.`,
				`Some operations can even be ;w;automated;x;, like ;dc;updating;x; your
;dc;progress;x; once you've finished watching an anime.`,
			}, 1, 1),
			m.ui.consent.View(
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

	view = ui.Style.Render(view)
	if c != nil {
		// -1 for the text input
		// (this obviously breaks for more than one input)
		c.Y += lipgloss.Height(view) - 1
	}
	return view, c
}

func (m SetupUserModel) ShortHelp() []key.Binding {
	if m.ui.loader.IsLoading() {
		return []key.Binding{}
	}

	if m.state.err != nil {
		return []key.Binding{ui.KeyMap.Abort}
	}

	return m.helpMap[m.state.view].ShortHelp(m)
}

func (m SetupUserModel) FullHelp() [][]key.Binding {
	if h, exists := m.helpMap[m.state.view]; exists && h.FullHelp != nil {
		return h.FullHelp(m)
	}
	return [][]key.Binding{}
}

func (m SetupUserModel) UpdateConsent(msg tea.Msg) (SetupUserModel, tea.Cmd) {
	m.ui.consent = m.ui.consent.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if ui.No == m.ui.consent.Select() {
				return m, m.abort
			}
			m.state.view = SetupUsernameView
			return m, tea.Batch(textinput.Blink, m.createWatchDir)
		}
	}

	return m, nil
}

func (m SetupUserModel) UpdateUsername(msg tea.Msg) (SetupUserModel, tea.Cmd) {
	var cmd tea.Cmd
	state := &m.state.username

	// Ignore all updates when we encounter an
	// unrecoverable error
	if m.state.err != nil {
		return m, nil
	}

	if state.notFound || state.found {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			if m.ui.loader.IsLoading() {
				return m, nil
			}
			// User can abort or try again
			if state.notFound {
				if ui.No == m.ui.consent.Select() {
					return m, m.abort
				}
				m.ui.username.Reset()
				state.notFound = false
				return m, nil
			}

			// User can reject or continue
			if state.found {
				if ui.No == m.ui.consent.Select() {
					state.found = false
					m.ui.username.Reset()
					return m, nil
				}
				m.state.view = SetupPasswordView
				// User will likely want to retry bad password
				m.ui.consent.SetConsentPos(ui.Yes)
				return m, nil
			}

			m.ui.loader, cmd = m.ui.loader.Start("User Setup")
			return m, tea.Batch(cmd, m.getProfile(m.ui.username.Value()))
		}

	case KitsuProfileMsg:
		m.state.data.Profile = msg
		state.found = true
		m.ui.loader.Stop()

	case FetchErrorMsg:
		if strings.Contains(msg.Error(), "profile not found") {
			state.notFound = true
			// Most users will want to try again quickly
			m.ui.consent.SetConsentPos(ui.Yes)
		} else {
			m.err = msg
		}
		m.ui.loader.Stop()

	}

	m.ui.username, cmd = m.ui.username.Update(msg)
	return m, cmd
}

func (m SetupUserModel) ViewUsername() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.username.notFound {
		return m.ui.consent.View(
			ui.DisplayText([]string{`;y;Profile not found; ;g;try again?`}, 0, 1),
		), nil
	}

	if m.state.err != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			ui.TextStyle.Foreground(ansi.BrightRed).Render("Fetch Error"),
			ui.TextStyle.PaddingLeft(2).
				Foreground(ansi.BrightYellow).
				Render(m.state.err.Error()),
		), nil
	}

	if m.state.username.found {
		p := m.state.data.Profile

		createdDate, err := time.Parse(time.RFC3339, p.CreatedAt)
		if err != nil {
			panic(err)
		}

		profileStr := ui.DisplayPropValue([]string{
			"Name", "About", "Gender", "BirthDay", "Location", "Created", "Profile",
		}, []string{
			fmt.Sprintf(";g;%s", p.Username), p.About, p.Gender, p.Birthday, p.Location,
			createdDate.Local().Format("01/02/2006 3:04 PM"),
			kitsu.GetProfileLink(p.ID),
		})

		return m.ui.consent.View(
			ui.DisplaySubTitle("Setup User", "Select User"),
			"",
			profileStr,
			ui.DisplayText([]string{`;b;Does that look like your profile?`}, 0, 1),
		), nil
	}

	c := m.ui.username.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("User Setup", "Username"),
		ui.DisplayText([]string{
			`A ;g;Kitsu;x; account is required. So make sure you've
created one at ;dc;http://kitsu.app;x;.`,
			`You'll also need to apply a profile URL to your account. You can do this
at the following link:`,
			`;dc;https://kitsu.app/settings/profile`,
			`You'll see a setting for ;w;Profile URL;x;. Enter a name for your profile URL
in that box and click ;w;update profile;x; at the bottom of the page.`,
			`;b;Enter the profile URL name you applied`,
		}, 1, 1),
		m.ui.username.View(),
	)
	return view, c
}

func (m SetupUserModel) UpdatePassword(msg tea.Msg) (SetupUserModel, tea.Cmd) {
	var cmd tea.Cmd
	state := &m.state.password
	if state.failed {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			if m.ui.loader.IsLoading() {
				return m, nil
			}

			// Allow user to retry or abort
			if state.failed {
				if ui.No == m.ui.consent.Select() {
					return m, m.abort
				}
				state.failed = false
				m.ui.password.Reset()
				return m, nil
			}

			if state.success {
				m.ui.loader, cmd = m.ui.loader.Start("User Setup")
				m.state.view = SetupLibraryView
				return m, tea.Batch(cmd, m.getAnimeLibrary(m.state.data.Profile.ID))
			}

			m.ui.loader, cmd = m.ui.loader.Start("User Setup")
			return m, tea.Batch(cmd, m.getAuthToken)
		}

	case FetchedAuthTokenMsg:
		state.success = true
		p := &m.state.data.Profile
		p.AccessToken = msg.Token
		p.RefreshToken = msg.RefreshToken
		// Because this is a duration and not a time stamp, we stamp
		// it ourselves by adding the unix time.
		p.TokenExpirationSec = int64(msg.ExpiresIn) + time.Now().Unix()
		m.ui.loader.Stop()
		return m, nil

	case FetchErrorMsg:
		if strings.Contains(msg.Error(), "provided authorization grant") {
			state.failed = true
			m.state.err = msg
		} else {
			m.err = msg
		}
		m.ui.loader.Stop()
	}

	m.ui.password, cmd = m.ui.password.Update(msg)
	return m, cmd
}

func (m SetupUserModel) ViewPassword() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.state.password.failed {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle("User Setup", "Password"),
			ui.DisplayText(
				[]string{
					`;r;Authorization Failed. ;x;You must have entered your password incorrectly.`,
				},
				1, 1,
			),
			m.ui.consent.View(ui.DisplayText([]string{";b;Would you like to try again?"}, 0)),
		), nil
	}

	if m.state.password.success {
		tokensStr := ui.DisplayText([]string{
			";c;Access Token",
			";bk;" + m.state.data.Profile.AccessToken,
			"",
			";c;Refresh Token",
			";bk;" + m.state.data.Profile.RefreshToken,
		})

		return lipgloss.JoinVertical(lipgloss.Left,
			ui.DisplaySubTitle("User Setup", "Credentials"),
			"",
			ui.DisplayText([]string{
				`By continuing, you acknowledge that ;g;Koshime;x; has the right to
use the following credentials to manage your ;w;Kitsu;x; account:`,
			}),
			"",
			ui.TextStyle.Render(tokensStr),
			ui.DisplayText([]string{
				`You ;m;do not;x; need to save these credentials; they are displayed for
transparency purposes only.`,
			}, 1, 1),
			ui.TextStyle.Foreground(ansi.BrightGreen).Render("> Continue"),
			"",
		), nil
	}

	c := m.ui.password.Cursor()
	c.Shape = tea.CursorBar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("User Setup", "Password"),
		ui.DisplayText([]string{
			`This process allows Koshime to gain an access token to your ;g;Kitsu;x; account.`,
			`;m;We do not save your password;x;. The token lasts for ;w;30 days;x; and can be
renewed any time within the ;w;30 days;x;, without a password.`,
			`;g;Koshime;x; tracks the time until token expiration and will let you know when
to renew it.`,
			`;b;Enter your password below to finish logging in.`,
		}, 1, 1),
		m.ui.password.View(),
	)
	return view, c
}

func (m SetupUserModel) UpdateLibAnime(msg tea.Msg) (SetupUserModel, tea.Cmd) {
	state := &m.state.libAnime

	if state.failed {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, ui.KeyMap.Select) {
			if m.ui.loader.IsLoading() {
				return m, nil
			}

			if state.failed {
				if ui.No == m.ui.consent.Select() {
					return m, m.abort
				}
				var cmd tea.Cmd
				m.ui.loader, cmd = m.ui.loader.Start("User Setup")
				state.failed = false
				return m, tea.Batch(cmd, m.getAnimeLibrary(m.state.data.Profile.ID))
			}

			if state.passed {
				return m, func() tea.Msg { return SetupUserFinishedMsg(m.state.data) }
			}
		}

	case FetchedLibAnimeMsg:
		m.state.data.Library = msg
		state.passed = true
		m.ui.loader.Stop()

	case FetchErrorMsg:
		state.failed = true
		m.state.err = msg
		m.ui.consent.SetConsentPos(ui.Yes)
		m.ui.loader.Stop()
	}
	return m, nil
}

func (m SetupUserModel) ViewLibAnime() string {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View())
	}

	if m.state.libAnime.failed {
		return m.ui.consent.View(
			ui.DisplayError(m.state.err),
			ui.DisplayText([]string{`;b;Would you like to try again?`}, 1, 1, 0),
		)
	}

	if m.state.libAnime.passed {
		loadedStr := ui.TextStyle.
			Render(
				utils.ColorText(
					fmt.Sprintf(
						"Loaded ;c;%d;x; Anime from your watch list",
						len(m.state.data.Library),
					),
				),
			)

		continueStr := ui.TextStyle.
			Foreground(ansi.BrightGreen).
			Render("> Continue")

		return lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle("User Setup", "Success"),
			"",
			loadedStr,
			"",
			continueStr,
		)
	}

	return ""
}

func (m SetupUserModel) getProfile(profileSlug string) tea.Cmd {
	return func() tea.Msg {
		// Convert to proper slug
		slug := strings.ReplaceAll(strings.TrimSpace(profileSlug), " ", "-")
		p, err := kitsu.GetProfile(slug)
		if err != nil {
			return FetchErrorMsg{Msg: err.Error()}
		}
		return KitsuProfileMsg(p)
	}
}

func (m SetupUserModel) getAnimeLibrary(userID string) func() tea.Msg {
	return func() tea.Msg {
		data, err := kitsu.GetUserAnime(userID, kitsu.LibAnimeWatching)
		if err != nil {
			return FetchErrorMsg{Msg: err.Error()}
		}
		return data
	}
}

func (m SetupUserModel) getAuthToken() tea.Msg {
	tokenData, err := kitsu.GetAuthToken(
		m.state.data.Profile.Slug,
		m.ui.password.Value(),
	)
	if err != nil {
		return FetchErrorMsg{Msg: err.Error()}
	}
	return tokenData
}

func (m SetupUserModel) createWatchDir() tea.Msg {
	wd := fileSys.GetWorkingDir()
	watchPath := filepath.Join(wd, "(watched)")
	_, err := os.Stat(watchPath)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(watchPath, 0o755); err != nil {
			return DefaultErrorMsg{err}
		}
		return nil
	}

	if err != nil {
		return DefaultErrorMsg{err}
	}

	return nil
}

func (m SetupUserModel) deleteWatchDir() {
	wd := fileSys.GetWorkingDir()
	err := os.Remove(filepath.Join(wd, "(watched)"))
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		panic(err)
	}
}

func (m SetupUserModel) abort() tea.Msg {
	return SetupUserAbortMsg{}
}

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
	"github.com/cli/browser"
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
	SetupBittorrentView
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
		data        database.Data
		view        SetupUserView
		err         error
		qBittorrent struct {
			accepted    bool
			tutorialPos int
			page0Sel    int
		}
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
		SetupBittorrentView: {
			ShortHelp: func(usm SetupUserModel) []key.Binding {
				return []key.Binding{
					ui.KeyMap.Up,
					ui.KeyMap.Down,
					ui.KeyMap.Select,
					ui.KeyMap.Back,
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
	case SetupBittorrentView:
		m, cmd = m.UpdateQBittorrent(msg)
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
	case SetupBittorrentView:
		view = m.ViewQBittorrent()
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
			m.state.view = SetupBittorrentView
			return m, tea.Batch(textinput.Blink, m.createWatchDir)
		}
	}

	return m, nil
}

func (m SetupUserModel) UpdateQBittorrent(msg tea.Msg) (SetupUserModel, tea.Cmd) {
	if !m.state.qBittorrent.accepted {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if !m.state.qBittorrent.accepted {
				if ui.No == m.ui.consent.Select() {
					m.state.view = SetupUsernameView
					return m, nil
				}
				m.state.qBittorrent.accepted = true
				return m, nil
			}

			switch m.state.qBittorrent.tutorialPos {
			case 0:
				/*
				 INFO: We ignore the error because the user will just move
				 on and search themselves if it doesn't work.
				*/
				switch m.state.qBittorrent.page0Sel {
				case 0:
					_ = browser.OpenURL("https://qbittorrent.org/download")
				case 1:
					_ = browser.OpenURL("https://github.com/qbittorrent/qBittorrent")
				case 2:
					m.state.qBittorrent.tutorialPos++
				}
			case 1, 2, 3:
				m.state.qBittorrent.tutorialPos++
			}

		case key.Matches(msg, ui.KeyMap.Down):
			if !m.state.qBittorrent.accepted {
				return m, nil
			}

			switch m.state.qBittorrent.tutorialPos {
			case 0:
				if m.state.qBittorrent.page0Sel+1 <= 2 {
					m.state.qBittorrent.page0Sel++
				}
			}

		case key.Matches(msg, ui.KeyMap.Up):
			if !m.state.qBittorrent.accepted {
				return m, nil
			}

			switch m.state.qBittorrent.tutorialPos {
			case 0:
				if m.state.qBittorrent.page0Sel-1 >= 0 {
					m.state.qBittorrent.page0Sel--
				}
			}

		case key.Matches(msg, ui.KeyMap.Back):
			if m.state.qBittorrent.tutorialPos-1 >= 0 {
				m.state.qBittorrent.tutorialPos--
			}

		}
	}

	return m, nil
}

func (m SetupUserModel) ViewQBittorrent() string {
	setupView := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("qBittorrent", "Setup"),
		ui.DisplayText([]string{
			`To make things easier, we can setup a torrent client called
;dg;qBittorrent;x;, to automatically download your anime for you. It is
a ;g;free;x; and ;g;open source;x; program.`,
			`It works on ;y;Linux;x;, ;c;MacOS;x;, and ;m;Windows;x;.`,
		}, 1, 1),
	)

	if !m.state.qBittorrent.accepted {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			setupView,
			m.ui.consent.View(ui.ConsentStyle.Render("Would you like to use qBittorrent?")),
		)
	}

	optStyle := ui.Style.MarginLeft(5)
	selStyle := ui.Style.MarginLeft(3).Foreground(ansi.BrightGreen)

	page1 := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("qBittorrent", "Tutorial Page 1"),
		ui.DisplayText([]string{
			`First you'll need to download the ;dg;qBittorrent;x; client
if you don't already have it installed. Once you have it downloaded,
you can install it using all its default settings.`,
			";b;Here are some options to get you started:;x;",
		}, 1, 1),
	)

	page2 := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("qBittorrent", "Tutorial Page 2"),
		ui.DisplayText([]string{
			`In order for ;g;Koshime;x; to connect to qBittorrent,
we need to enable its ;w;WebUI;x; component.`,
			`This can be done by opening ;w;qBittorrent;x; and clicking the
;c;gear icon;x;. This can also be done by opening its ;c;tools;x; menu and
selecting ;c;options;x; from the list.`,
			`The options window should now be visible. On the left side, find
the icon named ;w;WebUI;x; and click it. The icon will look like a ringed
planet.`,
		}, 1, 1),
	)

	page3 := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("qBittorrent", "Tutorial Page 3"),
		ui.DisplayText([]string{
			`You should now see all the options for the ;w;WebUI;x; on the right side
of the options window.`,
			`At the top, you'll see the text ;c;Web User Interface;x;, with a checkbox
next to it. Go ahead and click the checkbox. This will enable the ;w;WebUI;x;.`,
			`The line below that one, shows an ;c;IP Address;x; and ;c;Port;x;. The only
thing we care about is the ;c;Port;x;. The default is ;w;8080;x; but if you run other
services on that port, you'll need to change it.`,
			`If you want to make sure there are no conflicts, I would suggest changing the port
to ;g;8567;x;. This port is known to have virtually Zero conflicts.`,
		}, 1, 1),
	)

	page4 := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("qBittorrent", "Tutorial Page 4"),
		ui.DisplayText([]string{
			`Now that you have the ;c;Port;x; set, we need to set a ;c;Username;x; and
;c;Password;x;. Scan down a few lines from the port and you should see an ;w;Authentication;x;
and ;w;User;x; section.`,
			`The ;c;Username;x; should default to ;g;admin;x; which is fine. You can change this if you
want. The ;c;Password;x; however, definitely needs to be changed because it defaults to
;g;admin;x; as well.`,
			`Go ahead and click in the ;c;Password;x; box and change the password to something
you'll remember. Once you're done, click the ;w;Apply;x; button at the bottom right of the window.`,
		}, 1, 1),
	)

	page1Opts := [...]string{
		"Open Download Website",
		"Open Source-code Website",
		"Continue",
	}

	switch m.state.qBittorrent.tutorialPos {
	case 0:
		var sb strings.Builder
		sb.Grow(15 * 3)
		for i, opt := range page1Opts {
			if i > 0 {
				sb.WriteByte('\n')
			}
			if i == m.state.qBittorrent.page0Sel {
				sb.WriteString(selStyle.Render("> " + opt))
			} else {
				sb.WriteString(optStyle.Render(opt))
			}
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			page1,
			sb.String(),
		)

	case 1:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			page2,
			selStyle.Render("> Continue"),
		)

	case 2:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			page3,
			selStyle.Render("> Continue"),
		)

	case 3:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			page4,
			selStyle.Render("> Continue"),
		)
	}
	return ""
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

		return lipgloss.JoinVertical(
			lipgloss.Left,
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

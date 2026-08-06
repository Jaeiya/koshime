package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/x/ansi"
)

type TokenModel struct {
	db *database.Database
	ui struct {
		loader  ui.LoaderModel
		consent ui.ConsentModel
		input   textinput.Model
	}
	keys struct {
		v key.Binding
		r key.Binding
		x key.Binding
	}
	err         error
	data        kitsu.AuthTokenData
	showToken   bool
	isRenewing  bool
	isResetting bool
	triedRenew  bool
}

func newTokenModel(db *database.Database) TokenModel {
	m := TokenModel{db: db}
	m.ui.loader = ui.NewLoader()
	m.ui.input = ui.NewTextInput()
	m.ui.input.EchoMode = textinput.EchoPassword

	m.keys.r = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "renew"))
	m.keys.x = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "reset"))
	m.keys.v = key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "view"))
	return m
}

func (TokenModel) Init() tea.Cmd {
	return nil
}

func (m TokenModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			if m.err != nil {
				m.err = nil
				return m, nil
			}
			if m.isResetting {
				m.isResetting = false
				return m, nil
			}
			return m, exitToMenu

		case key.Matches(msg, ui.KeyMap.Back):
			if m.err != nil {
				m.err = nil
				return m, nil
			}

		case key.Matches(msg, m.keys.r):
			// Prevent key press when entering password
			if m.isResetting {
				break
			}
			if m.HasBeenRenewed() {
				m.triedRenew = true
				return m, nil
			}
			m.isRenewing = true

		case key.Matches(msg, m.keys.x):
			// Prevent key press when entering password
			if m.isResetting {
				break
			}
			m.isResetting = true
			return m, nil

		case key.Matches(msg, m.keys.v):
			m.showToken = !m.showToken

		case key.Matches(msg, ui.KeyMap.Submit):
			if m.isResetting {
				m.ui.loader, cmd = m.ui.loader.Start("Getting Auth Token")
				return m, tea.Batch(cmd, m.resetToken)
			}

			if m.isRenewing {
				if m.ui.consent.Select() == ui.No {
					m.isRenewing = false
					return m, nil
				}
				m.isRenewing = false
				m.ui.loader, cmd = m.ui.loader.Start("Refreshing Token")
				return m, tea.Batch(cmd, m.RefreshToken)
			}
		}

	case error:
		m.err = msg
		m.ui.loader.Stop()

	case kitsu.AuthTokenData:
		m.data = msg
		m.isResetting = false
		m.ui.loader.Stop()
	}

	if m.isRenewing {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.isResetting {
		m.ui.input, cmd = m.ui.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m TokenModel) View() tea.View {
	if m.ui.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.ui.loader.View()))
	}

	if m.err != nil {
		return tea.NewView(ui.DisplayError(m.err))
	}

	if m.isResetting {
		return m.ViewReset()
	}

	if m.isRenewing {
		return m.ViewRenew()
	}

	return m.ViewDefault()
}

func (m TokenModel) ShortHelp() []key.Binding {
	if m.err != nil {
		return []key.Binding{ui.KeyMap.EscBack}
	}
	if m.isResetting {
		return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.MainMenu}
	}
	if m.isRenewing {
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, m.keys.v, ui.KeyMap.MainMenu}
	}
	return []key.Binding{m.keys.r, m.keys.v, m.keys.x, ui.KeyMap.MainMenu}
}

func (m TokenModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m TokenModel) ViewDefault() tea.View {
	p := m.db.Profile()

	tokenExpiration := utils.NewRelativeTimeUnits(p.TokenExpirationSec)
	expStyle := ui.ExpireStyle(tokenExpiration)

	at := p.AccessToken
	rt := p.RefreshToken
	if !m.showToken {
		at = at[:len(at)/2] + strings.Repeat("◆", len(at)/2+1)
		rt = rt[:len(rt)/2] + strings.Repeat("◆", len(rt)/2+1)
	}
	tokenStyle := ui.TextStyle.PaddingLeft(3).Foreground(ansi.BrightBlack)
	expirationStr := ui.TextStyle.Render(
		utils.ColorText(fmt.Sprintf(
			";w;Expiration: %s",
			expStyle.Render(tokenExpiration.ToPrecisionString(utils.Days)),
		)),
	)
	if m.HasBeenRenewed() {
		expirationStr = ui.DisplayText([]string{
			";w;Expiration: ;g;in a Month",
		})
		if m.triedRenew {
			expirationStr = ui.DisplayText([]string{
				";m;[cannot renew a new token]",
			})
		}
	}

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Manage Token"),
		"",
		expirationStr,
		"",
		ui.TextStyle.Foreground(ansi.Green).Render("Access Token"),
		tokenStyle.Render(at),
		ui.TextStyle.Foreground(ansi.Green).Render("Refresh Token"),
		tokenStyle.Render(rt),
	))
}

func (m TokenModel) ViewReset() tea.View {
	view := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Manage Token", "Reset"),
		"",
		ui.DisplayText(
			[]string{
				`You only need to do this if your token has ;dc;expired;x; or some other
service/app is using it.`,
				";m;[your password will not be saved]",
			},
			1, 0, 0,
		),
		"",
	))

	view.Cursor = m.ui.input.Cursor()
	view.Cursor.Shape = tea.CursorBar
	view.Cursor.Y = lipgloss.Height(view.Content)
	view.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		view.Content,
		m.ui.input.View(),
	)

	return view
}

func (m TokenModel) ViewRenew() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Manage Token", "Renew"),
		"",
		ui.DisplayText([]string{
			`This will refresh access to your ;c;Kitsu;x; account, so that ;c;Koshime;x;
can continue to update your watch list.`,
			`If at any point you ;m;miss;x; the expiration, you'll have to use the ;y;reset;x;
option, which requires your password.`,
		}, 1),
		ui.TextStyle.Render(
			m.ui.consent.View(utils.ColorText(";b;Are you sure you want to renew?")),
		),
	))
}

func (m TokenModel) RefreshToken() tea.Msg {
	data, err := kitsu.RefreshToken(m.db.Profile().RefreshToken)
	if err != nil {
		return err
	}

	err = m.db.SaveTokenData(data.Token, data.RefreshToken, data.ExpiresIn)
	if err != nil {
		return err
	}
	return data
}

func (m TokenModel) resetToken() tea.Msg {
	p := m.db.Profile()
	data, err := kitsu.GetAuthToken(p.Slug, m.ui.input.Value())
	if err != nil {
		return fmt.Errorf(
			"failed to reset token with %s:%s: %w",
			p.Slug,
			m.ui.input.Value(),
			err,
		)
	}

	err = m.db.SaveTokenData(data.Token, data.RefreshToken, data.ExpiresIn)
	if err != nil {
		return err
	}
	return data
}

func (m TokenModel) HasBeenRenewed() bool {
	timeUnits := utils.NewRelativeTimeUnits(m.db.Profile().TokenExpirationSec)
	return timeUnits.Weeks == 4 && timeUnits.Days > 0
}

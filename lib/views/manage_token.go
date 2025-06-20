package views

import (
	"fmt"
	"strings"

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
			if m.HasBeenRenewed() {
				m.triedRenew = true
				return m, nil
			}
			m.isRenewing = true

		case key.Matches(msg, m.keys.x):
			m.isResetting = true
			return m, nil

		case key.Matches(msg, m.keys.v):
			m.showToken = !m.showToken

		case key.Matches(msg, ui.KeyMap.Submit):
			if m.isResetting {
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

func (m TokenModel) View() (string, *tea.Cursor) {
	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	if m.err != nil {
		return ui.DisplayError(m.err), nil
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

func (m TokenModel) ViewDefault() (string, *tea.Cursor) {
	p := m.db.GetProfile()

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

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Manage Token"),
		"",
		expirationStr,
		"",
		ui.TextStyle.Foreground(ansi.Green).Render("Access Token"),
		tokenStyle.Render(at),
		ui.TextStyle.Foreground(ansi.Green).Render("Refresh Token"),
		tokenStyle.Render(rt),
	)

	return view, nil
}

func (m TokenModel) ViewReset() (string, *tea.Cursor) {
	view := lipgloss.JoinVertical(
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
	)
	c := m.ui.input.Cursor()
	c.Shape = tea.CursorBar
	c.Y = lipgloss.Height(view)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		view,
		m.ui.input.View(),
	), c
}

func (m TokenModel) ViewRenew() (string, *tea.Cursor) {
	return lipgloss.JoinVertical(
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
	), nil
}

func (m TokenModel) RefreshToken() tea.Msg {
	data, err := kitsu.RefreshToken(m.db.GetProfile().RefreshToken)
	if err != nil {
		return err
	}

	err = m.db.SaveTokenData(data.Token, data.RefreshToken, data.ExpiresIn)
	if err != nil {
		return err
	}
	return data
}

func (m TokenModel) HasBeenRenewed() bool {
	rtu := utils.NewRelativeTimeUnits(m.db.GetProfile().TokenExpirationSec)
	return rtu.Weeks == 4 && rtu.Days > 0
}

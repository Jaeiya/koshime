package views

import (
	"fmt"
	"strings"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type TokenModel struct {
	db *database.Database
	ui struct {
		loader  ui.LoaderModel
		consent ui.ConsentModel
	}
	keys struct {
		v key.Binding
		r key.Binding
	}
	err        error
	data       kitsu.AuthTokenData
	showToken  bool
	renewToken bool
}

func NewTokenModel(db *database.Database) TokenModel {
	m := TokenModel{db: db}
	m.ui.loader = ui.NewLoader()

	m.keys.r = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "renew"))
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
			return m, exitToMenu

		case key.Matches(msg, ui.KeyMap.Back):
			if m.err != nil {
				m.err = nil
				return m, nil
			}

		case key.Matches(msg, m.keys.r):
			m.renewToken = true

		case key.Matches(msg, m.keys.v):
			m.showToken = !m.showToken

		case key.Matches(msg, ui.KeyMap.Submit):
			if m.renewToken {
				if m.ui.consent.Select() == ui.No {
					m.renewToken = false
					return m, nil
				}
				m.renewToken = false
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

	if m.renewToken {
		m.ui.consent = m.ui.consent.Update(msg)
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
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

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Manage Token"),
		"",
		ui.TextStyle.Render(
			utils.ColorText(fmt.Sprintf(
				";w;Expiration: %s",
				expStyle.Render(tokenExpiration.ToPrecisionString(utils.Days)),
			)),
		),
		"",
		ui.TextStyle.Foreground(ansi.Green).Render("Access Token"),
		tokenStyle.Render(at),
		ui.TextStyle.Foreground(ansi.Green).Render("Refresh Token"),
		tokenStyle.Render(rt),
	)

	if m.renewToken {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			"",
			ui.TextStyle.Render(
				m.ui.consent.View(utils.ColorText(";b;Are you sure you want to renew?")),
			),
		), nil
	}
	return view, nil
}

func (m TokenModel) ShortHelp() []key.Binding {
	if m.err != nil {
		return []key.Binding{ui.KeyMap.EscBack}
	}
	if m.renewToken {
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, m.keys.v, ui.KeyMap.MainMenu}
	}
	return []key.Binding{m.keys.r, m.keys.v, ui.KeyMap.MainMenu}
}

func (m TokenModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
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

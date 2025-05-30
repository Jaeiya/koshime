package views

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type ExitToMenuMsg struct{}

type MenuView struct {
	Name  string
	Model ViewModel
	Desc  string
}

type MenuModel struct {
	menuItems     []MenuView
	selectedModel ViewModel
	help          help.Model
	menuPos       int
	profile       kitsu.Profile
}

func NewMenuModel(views []MenuView, p kitsu.Profile) MenuModel {
	m := MenuModel{}
	m.profile = p

	m.help = help.New()
	m.help.Styles.ShortKey = ui.HelpKeyStyle
	m.help.Styles.FullKey = m.help.Styles.ShortKey
	m.help.Styles.ShortDesc = ui.HelpDescStyle
	m.help.Styles.FullDesc = m.help.Styles.ShortDesc

	m.menuItems = views
	return m
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	var cmd tea.Cmd

	if m.selectedModel != nil {
		m.menuItems[m.menuPos].Model, cmd = m.selectedModel.Update(msg)
		switch msg.(type) {
		case ExitToMenuMsg:
			m.selectedModel = nil
		}
		return m, cmd
	}

	itemLen := len(m.menuItems)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		for _, v := range m.menuItems {
			v.Model, _ = v.Model.Update(msg)
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Exit):
			return m, exit

		case key.Matches(msg, ui.KeyMap.Select):
			m.selectedModel = m.menuItems[m.menuPos].Model

		case key.Matches(msg, ui.KeyMap.HelpMore):
			if m.selectedModel == nil || len(m.selectedModel.FullHelp()) == 0 {
				break
			}
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, ui.KeyMap.Up):
			m.menuPos = (m.menuPos - 1 + itemLen) % itemLen

		case key.Matches(msg, ui.KeyMap.Down):
			m.menuPos = (m.menuPos + 1) % itemLen

		}
	}

	return m, cmd
}

func (m MenuModel) View() (string, *tea.Cursor) {
	if m.selectedModel != nil {
		v, c := m.selectedModel.View()
		return lipgloss.JoinVertical(
			lipgloss.Left,
			v,
			ui.HelpStyle.Render(m.help.View(m.selectedModel)),
		), c
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.DisplayProfile(),
		m.DisplayMenu(),
		ui.HelpStyle.Render(m.help.View(m)),
	), nil
}

func (m MenuModel) ShortHelp() []key.Binding {
	return []key.Binding{
		ui.KeyMap.Up,
		ui.KeyMap.Down,
		ui.KeyMap.Select,
		ui.KeyMap.Exit,
	}
}

func (m MenuModel) FullHelp() [][]key.Binding {
	if m.selectedModel != nil {
		return m.menuItems[m.menuPos].Model.FullHelp()
	}
	return [][]key.Binding{}
}

func (m MenuModel) DisplayMenu() string {
	items := make([]string, len(m.menuItems)+1)
	menuStyle := ui.TextStyle.MarginLeft(5).Width(12).PaddingLeft(1).PaddingRight(3)

	// Header
	items[0] = ui.TextStyle.
		MarginTop(1).
		MarginBottom(1).
		Render(utils.ColorText(";b;What would you like to do?"))

	for i, v := range m.menuItems {
		if m.menuPos == i {
			items[i+1] = menuStyle.Foreground(ansi.BrightGreen).
				Background(ansi.Black).
				Render("> " + v.Name)
			continue
		}
		items[i+1] = menuStyle.Render("  " + v.Name)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		items...,
	)
}

func (m MenuModel) DisplayProfile() string {
	p := m.profile
	header := ui.TextStyle.
		MarginTop(1).
		MarginBottom(1).
		Render(utils.ColorText(fmt.Sprintf(";dy;%s's;b; profile stats:", p.Username)))

	expStyle := ui.Style
	tokenExpiration := utils.NewRelativeTimeUnits(p.TokenExpirationSec)
	switch {
	case tokenExpiration.Weeks < 1:
		expStyle = expStyle.Foreground(ansi.BrightRed)
	case tokenExpiration.Weeks < 2:
		expStyle = expStyle.Foreground(ansi.BrightYellow)
	default:
		expStyle = expStyle.Foreground(ansi.BrightGreen)
	}

	d := ui.DisplayPropVal([]string{
		utils.ColorText(";dc;Completed Anime"),
		utils.ColorText(";dc;Time Watched"),
		utils.ColorText(";dc;Token Expiration"),
		utils.ColorText(";dc;Last Updated"),
	}, []string{
		strconv.Itoa(p.CompletedSeries),
		utils.NewDurationUnits(time.Second * time.Duration(p.SecondsWatched)).
			ToShortString(),
		expStyle.Render(tokenExpiration.ToPrecisionString(utils.Days)),
		utils.NewRelativeTimeUnits(p.LastUpdateSec).String(),
	})

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		d,
	)
}

func exitToMenu() tea.Msg {
	return ExitToMenuMsg{}
}

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
	Name     string
	Model    ViewModel
	Desc     string
	SubViews []MenuView
}

type MenuModel struct {
	windowSize    tea.WindowSizeMsg
	menuItems     []MenuView
	activeItems   []MenuView
	selectedModel ViewModel
	help          help.Model
	menuIndex     int
	activeIndex   int
	profile       kitsu.Profile
	inSubMenu     bool
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
	m.activeItems = views
	return m
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	var cmd tea.Cmd

	if m.selectedModel != nil {
		m.selectedModel, cmd = m.selectedModel.Update(msg)
		if _, ok := msg.(ExitToMenuMsg); ok {
			m.selectedModel = nil
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Exit):
			if m.inSubMenu {
				m.inSubMenu = false
				m.activeIndex = m.menuIndex
				m.activeItems = m.menuItems
				return m, nil
			}
			return m, exit

		case key.Matches(msg, ui.KeyMap.Back):
			if m.inSubMenu {
				m.inSubMenu = false
				m.activeIndex = m.menuIndex
				m.activeItems = m.menuItems
				return m, nil
			}
			return m, nil

		case key.Matches(msg, ui.KeyMap.Select):
			chosen := m.activeItems[m.activeIndex]
			if chosen.SubViews != nil {
				m.inSubMenu = true
				m.menuIndex = m.activeIndex
				m.activeItems = chosen.SubViews
				m.activeIndex = 0
			} else {
				m.selectedModel = chosen.Model
				m.selectedModel, _ = m.selectedModel.Update(m.windowSize)
			}

		case key.Matches(msg, ui.KeyMap.HelpMore):
			if m.selectedModel == nil || len(m.selectedModel.FullHelp()) == 0 {
				break
			}
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, ui.KeyMap.Up):
			itemLen := len(m.activeItems)
			m.activeIndex = (m.activeIndex - 1 + itemLen) % itemLen

		case key.Matches(msg, ui.KeyMap.Down):
			m.activeIndex = (m.activeIndex + 1) % len(m.activeItems)

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

	title := ui.DisplayTitle("Koshime Menu")
	if m.inSubMenu {
		title = ui.DisplaySubTitle("Koshime Menu", m.menuItems[m.menuIndex].Name)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.DisplayProfile(),
		title,
		"",
		m.DisplayMenu(),
		ui.HelpStyle.Render(m.help.View(m)),
	), nil
}

func (m MenuModel) ShortHelp() []key.Binding {
	exitKey := ui.KeyMap.Exit
	if m.inSubMenu {
		exitKey = ui.KeyMap.MainMenu
	}
	return []key.Binding{
		ui.KeyMap.Up,
		ui.KeyMap.Down,
		ui.KeyMap.Select,
		exitKey,
	}
}

func (m MenuModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m MenuModel) DisplayMenu() string {
	lines := make([]string, len(m.activeItems))
	menuStyle := ui.TextStyle.MarginLeft(5).Width(17).PaddingLeft(1).PaddingRight(3)

	for i, item := range m.activeItems {
		if m.activeIndex == i {
			lines[i] = menuStyle.Foreground(ansi.BrightGreen).
				Background(ansi.Black).
				Render("> " + item.Name)
			continue
		}
		lines[i] = menuStyle.Render("  " + item.Name)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m MenuModel) DisplayProfile() string {
	p := m.profile
	header := ui.TextStyle.
		MarginTop(1).
		MarginBottom(1).
		Render(utils.ColorText(fmt.Sprintf(";dy;%s's;b; profile stats:", p.Username)))

	tokenExpiration := utils.NewRelativeTimeUnits(p.TokenExpirationSec)
	expStyle := ui.ExpireStyle(tokenExpiration)

	props := []string{
		utils.ColorText(";dc;Completed Anime"),
		utils.ColorText(";dc;Time Watched"),
		utils.ColorText(";dc;Token Expiration"),
		utils.ColorText(";dc;Last Updated"),
	}
	values := []string{
		strconv.Itoa(p.CompletedSeries),
		utils.NewDurationUnits(time.Second * time.Duration(p.SecondsWatched)).
			ToShortString(),
		expStyle.Render(tokenExpiration.ToPrecisionString(utils.Days)),
		utils.NewRelativeTimeUnits(p.LastUpdateSec).String(),
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		ui.DisplayPropVal(props, values),
	)
}

func exitToMenu() tea.Msg {
	return ExitToMenuMsg{}
}

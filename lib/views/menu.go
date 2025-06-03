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
	menuItems     []MenuView
	subItems      []MenuView
	selectedModel ViewModel
	help          help.Model
	menuPos       int
	subMenuPos    int
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
		item := &m.menuItems[m.menuPos]
		if m.subItems != nil {
			item = &m.subItems[m.subMenuPos]
		}
		item.Model, cmd = item.Model.Update(msg)
		switch msg.(type) {
		case ExitToMenuMsg:
			// Display main menu
			m.subItems = nil
			m.selectedModel = nil
		}
		return m, cmd
	}

	itemLen := len(m.menuItems)
	if m.subItems != nil {
		itemLen = len(m.subItems)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		for _, v := range m.menuItems {
			if v.Model != nil {
				v.Model, _ = v.Model.Update(msg)
			}
			// Send window size to sub-menu views
			if v.Model == nil {
				for _, v := range v.SubViews {
					if v.Model == nil {
						continue
					}
					v.Model, _ = v.Model.Update(msg)
				}
			}
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Exit):
			if m.subItems != nil {
				m.subItems = nil
				return m, nil
			}
			return m, exit

		case key.Matches(msg, ui.KeyMap.Back):
			if m.subItems != nil {
				m.subItems = nil
			}
			return m, nil

		case key.Matches(msg, ui.KeyMap.Select):
			menuItem := m.menuItems[m.menuPos]
			if m.subItems == nil && menuItem.SubViews != nil {
				m.subItems = menuItem.SubViews
			} else if menuItem.SubViews != nil {
				m.selectedModel = m.subItems[m.subMenuPos].Model
			} else {
				m.selectedModel = m.menuItems[m.menuPos].Model
			}

		case key.Matches(msg, ui.KeyMap.HelpMore):
			if m.selectedModel == nil || len(m.selectedModel.FullHelp()) == 0 {
				break
			}
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, ui.KeyMap.Up):
			if m.subItems != nil {
				m.subMenuPos = (m.subMenuPos - 1 + itemLen) % itemLen
			} else {
				m.menuPos = (m.menuPos - 1 + itemLen) % itemLen
			}

		case key.Matches(msg, ui.KeyMap.Down):
			if m.subItems != nil {
				m.subMenuPos = (m.subMenuPos + 1) % itemLen
			} else {
				m.menuPos = (m.menuPos + 1) % itemLen
			}

		}
	}

	return m, cmd
}

func (m MenuModel) View() (string, *tea.Cursor) {
	if m.selectedModel != nil {
		item := m.menuItems[m.menuPos]
		if m.subItems != nil {
			item = m.subItems[m.subMenuPos]
		}
		v, c := item.Model.View()
		return lipgloss.JoinVertical(
			lipgloss.Left,
			v,
			ui.HelpStyle.Render(m.help.View(item.Model)),
		), c
	}

	title := ui.DisplayTitle("Koshime Menu")
	if m.subItems != nil {
		title = ui.DisplaySubTitle("Koshime Menu", m.menuItems[m.menuPos].Name)
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
	var items []string
	if m.subItems != nil {
		items = make([]string, len(m.subItems))
	} else {
		items = make([]string, len(m.menuItems))
	}

	menuStyle := ui.TextStyle.MarginLeft(5).Width(17).PaddingLeft(1).PaddingRight(3)

	menuItems := m.menuItems
	menuPos := m.menuPos
	if m.subItems != nil {
		menuItems = m.subItems
		menuPos = m.subMenuPos
	}

	for i, v := range menuItems {
		if menuPos == i {
			items[i] = menuStyle.Foreground(ansi.BrightGreen).
				Background(ansi.Black).
				Render("> " + v.Name)
			continue
		}
		items[i] = menuStyle.Render("  " + v.Name)
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

	tokenExpiration := utils.NewRelativeTimeUnits(p.TokenExpirationSec)
	expStyle := ui.ExpireStyle(tokenExpiration)

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

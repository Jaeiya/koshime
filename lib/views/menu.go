package views

import (
	"fmt"
	"strconv"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/qbittorrent"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/x/ansi"
)

type (
	ExitToMenuMsg struct{}
	QbtStateMsg   struct{ Value QbtState }
)

type QbtState uint8

const (
	None = QbtState(iota)
	Pending
	Offline
	Online
)

type MenuView struct {
	Name      string
	ModelFunc func() ViewModel
	Desc      string
	SubViews  []MenuView
}

type MenuModel struct {
	windowSize    tea.WindowSizeMsg
	menuItems     []MenuView
	activeItems   []MenuView
	selectedModel ViewModel
	help          help.Model
	menu          ui.MenuModel
	menuIndex     int
	activeIndex   int
	qbtState      QbtState
	isQbtInit     bool
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
	m.qbtState = Pending

	m.menuItems = views
	m.activeItems = views

	m.updateMenu()
	return m
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.selectedModel != nil {
		m.selectedModel, cmd = m.selectedModel.Update(msg)
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch {
			case key.Matches(msg, ui.KeyMap.HelpMore):
				if m.selectedModel == nil || len(m.selectedModel.FullHelp()) == 0 {
					break
				}
				m.help.ShowAll = !m.help.ShowAll
				return m, nil
			}

		case ExitToMenuMsg:
			// Short help should always be default
			m.help.ShowAll = false
			m.selectedModel = nil
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.inSubMenu {
				m.inSubMenu = false
				m.activeIndex = m.menuIndex
				m.activeItems = m.menuItems
				m.updateMenu()
				return m, nil
			}
			return m, exit

		case key.Matches(msg, ui.KeyMap.Back):
			if m.inSubMenu {
				m.inSubMenu = false
				m.activeIndex = m.menuIndex
				m.activeItems = m.menuItems
				m.updateMenu()
				return m, nil
			}
			return m, nil

		}

	case ui.MenuIndexMsg:
		chosen := m.activeItems[msg]
		if chosen.SubViews != nil {
			m.inSubMenu = true
			m.activeIndex = int(msg)
			m.activeItems = chosen.SubViews
			m.menuIndex = int(msg)
			m.updateMenu()
		} else {
			m.selectedModel = chosen.ModelFunc()
			m.selectedModel, cmd = m.selectedModel.Update(m.windowSize)
			cmds = append(cmds, cmd, m.selectedModel.Init())
		}

	case QbtStateMsg:
		m.qbtState = msg.Value

	}

	if !m.isQbtInit {
		m, cmd = m.initQbtState()
		cmds = append(cmds, cmd)
	}

	m.menu, cmd = m.menu.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m MenuModel) View() tea.View {
	if m.selectedModel != nil {
		mv := m.selectedModel.View()
		v := tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			mv.Content,
			ui.HelpStyle.Render(m.help.View(m.selectedModel)),
		))
		v.Cursor = mv.Cursor
		return v
	}

	title := ui.DisplayTitle("Menu")
	if m.inSubMenu {
		title = ui.DisplaySubTitle("Menu", m.menuItems[m.menuIndex].Name)
	}

	v := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Koshime Profile"),
		"",
		m.DisplayProfile(),
		title,
		"",
		m.menu.View(),
		ui.HelpStyle.Render(m.help.View(m)),
	))

	return v
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

func (m MenuModel) DisplayProfile() string {
	p := m.profile

	tokenExpiration := utils.NewRelativeTimeUnits(p.TokenExpirationSec)
	expStyle := ui.ExpireStyle(tokenExpiration)

	props := []string{
		utils.ColorText(";dy;Name"),
		utils.ColorText(";dc;Completed Anime"),
		utils.ColorText(";dc;Time Watched"),
		utils.ColorText(";dc;Token Expiration"),
		utils.ColorText(";dc;Last Updated"),
	}

	values := []string{
		ui.Style.Foreground(ansi.BrightWhite).Render(p.Username),
		strconv.Itoa(p.CompletedSeries),
		utils.NewDurationUnits(time.Second * time.Duration(p.SecondsWatched)).
			ToShortString(),
		expStyle.Render(tokenExpiration.ToPrecisionString(utils.Days)),
		utils.NewRelativeTimeUnits(p.LastUpdateSec).String(),
	}

	// Add qBittorrent display
	if m.qbtState > None {
		props = append(props, utils.ColorText(";dc;qBittorrent"))
		strPort := strconv.Itoa(p.QbtPort)

		var qbtState string
		switch m.qbtState {
		case Pending:
			qbtState = ui.Style.Foreground(ansi.BrightMagenta).Render("Pending")
		case Offline:
			qbtState = ui.Style.Foreground(ansi.BrightRed).Render("Offline")
		case Online:
			qbtState = ui.Style.Foreground(ansi.BrightGreen).Render("Online")
		default:
		}

		values = append(
			values,
			ui.Style.Foreground(ansi.BrightBlue).
				Render(fmt.Sprintf(utils.ColorText(";dy;Port;x; ;c;%s;x;, ;dy;Status;x; %s"), strPort, qbtState)),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayPropValue(props, values),
	)
}

func (m *MenuModel) updateMenu() {
	items := make([]string, len(m.activeItems))
	menuDesc := make([]string, len(m.activeItems))
	for i, item := range m.activeItems {
		menuDesc[i] = item.Desc
		items[i] = item.Name
	}
	m.menu = ui.NewMenuModel(items, ui.WithMenuRotation(), ui.WithMenuDescriptions(menuDesc))
}

func (m MenuModel) initQbtState() (MenuModel, tea.Cmd) {
	cmd := func() tea.Msg {
		if m.profile.QbtPort <= 0 {
			return QbtStateMsg{None}
		}

		strPort := strconv.Itoa(m.profile.QbtPort)
		if err := qbittorrent.CheckConn(strPort); err != nil {
			return QbtStateMsg{Offline}
		}

		return QbtStateMsg{Online}
	}

	m.isQbtInit = true
	return m, cmd
}

func exitToMenu() tea.Msg {
	return ExitToMenuMsg{}
}

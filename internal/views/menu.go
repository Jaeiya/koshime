package views

import (
	"fmt"
	"strconv"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/qbittorrent"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
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
	db            *database.Database
	windowSize    tea.WindowSizeMsg
	menuItems     []MenuView
	activeItems   []MenuView
	selectedModel ViewModel
	help          help.Model
	menu          ui.MenuModel
	menuIndex     int
	qbtState      QbtState
	inSubMenu     bool
}

func NewMenuModel(views []MenuView, db *database.Database) MenuModel {
	m := MenuModel{}
	m.db = db

	m.help = help.New()
	m.help.Styles.ShortKey = ui.HelpKeyStyle
	m.help.Styles.FullKey = ui.HelpKeyStyle
	m.help.Styles.ShortDesc = ui.HelpDescStyle
	m.help.Styles.FullDesc = ui.HelpDescStyle
	m.qbtState = Pending

	m.menuItems = views
	m.activeItems = views
	m.menu = m.newMenu(m.activeItems)
	return m
}

func (m MenuModel) Init() tea.Cmd {
	return m.initQbtState()
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
				if m.selectedModel == nil || m.selectedModel.FullHelp() == nil {
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
				m.activeItems = m.menuItems
				m.menu = m.newMenu(m.menuItems)
				if err := m.menu.Select(m.menuIndex); err != nil {
					return m, func() tea.Msg { return err }
				}
				return m, nil
			}
			return m, exit
		}

	case ui.MenuItemSelMsg:
		chosen := m.activeItems[msg.Value]
		if chosen.SubViews != nil {
			m.inSubMenu = true
			m.menuIndex = msg.Value
			m.activeItems = chosen.SubViews
			m.menu = m.newMenu(chosen.SubViews)
		} else {
			m.selectedModel = chosen.ModelFunc()
			m.selectedModel, cmd = m.selectedModel.Update(m.windowSize)
			cmds = append(cmds, cmd, m.selectedModel.Init())
		}

	case QbtStateMsg:
		m.qbtState = msg.Value
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
	p := m.db.Profile()

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

func (m MenuModel) newMenu(views []MenuView) ui.MenuModel {
	items := make([]string, len(views))
	menuDesc := make([]string, len(views))
	for i, item := range views {
		menuDesc[i] = item.Desc
		items[i] = item.Name
	}
	return ui.NewMenuModel(items, ui.WithMenuRotation(), ui.WithMenuDescriptions(menuDesc))
}

func (m MenuModel) initQbtState() tea.Cmd {
	return func() tea.Msg {
		p := m.db.Profile()
		if p.QbtPort <= 0 {
			return QbtStateMsg{None}
		}
		strPort := strconv.Itoa(p.QbtPort)
		if err := qbittorrent.CheckConn(strPort); err != nil {
			return QbtStateMsg{Offline}
		}
		return QbtStateMsg{Online}
	}
}

func exitToMenu() tea.Msg {
	return ExitToMenuMsg{}
}

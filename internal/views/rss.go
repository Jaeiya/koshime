package views

import (
	"strconv"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/qbittorrent"
	"github.com/Jaeiya/koshime/internal/ui"
)

type RssView int

const (
	RssSelection = RssView(iota)
	RssSearch
	RssReview
	RssQbtSearch
	RssQbtReview
)

type RssMenuOption int

const (
	RssManualOpt = RssMenuOption(iota)
	RssAutoOpt
)

type RssRouteMsg struct {
	View RssView
}
type (
	QbtConnMsg        struct{ isOnline bool }
	QbtSavedMsg       struct{ err error }
	QbtRemovedFeedMsg bool
)

var menuOptions = [...]string{"Manual", "Automatic"}

type RssMainModel struct {
	db             *database.Database
	menu           ui.MenuModel
	consent        ui.ConsentModel
	view           RssView
	searchModel    RssSearchModel
	qbtSearchModel RssQbtSearchModel
	windowSize     tea.WindowSizeMsg
	err            error
	isQbtOnline    bool
}

func newRssMainModel(db *database.Database) RssMainModel {
	m := RssMainModel{db: db}
	m.menu = ui.NewMenuModel(menuOptions[:])
	m.searchModel = newRssSearchModel()
	m.qbtSearchModel = newRssQbtSearchModel(db)
	return m
}

func (m RssMainModel) Init() tea.Cmd {
	return m.testConn()
}

func (m RssMainModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg
		m.searchModel, _ = m.searchModel.Update(msg)
		m.qbtSearchModel, _ = m.qbtSearchModel.Update(msg)

	case RssRouteMsg:
		m.view = msg.View

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			if m.err != nil {
				m.view = RssSelection
				// Reset models so we don't have broken state after error
				m.searchModel = newRssSearchModel()
				m.qbtSearchModel = newRssQbtSearchModel(m.db)
				m.err = nil
				return m, nil
			}
			if m.view == RssSelection {
				return m, exitToMenu
			}

		case key.Matches(msg, ui.KeyMap.Select):
			if !m.isQbtOnline {
				if m.consent.Select() == ui.No {
					m.view = RssSearch
					return m, nil
				}
				return m, m.testConn()
			}
		}

	case QbtConnMsg:
		if msg.isOnline {
			m.isQbtOnline = true
			return m, nil
		}

	case ui.MenuIndexMsg:
		switch RssMenuOption(msg) {
		case RssManualOpt:
			m.view = RssSearch
		default:
			m.view = RssQbtSearch
		}

	case error:
		m.err = msg
	}

	// Do not execute updates on errors
	if m.err != nil {
		return m, nil
	}

	switch m.view {
	case RssSelection:
		m.menu, cmd = m.menu.Update(msg)
		if !m.isQbtOnline {
			m.consent = m.consent.Update(msg)
		}
		cmds = append(cmds, cmd)
	case RssSearch:
		m.searchModel, cmd = m.searchModel.Update(msg)
		cmds = append(cmds, cmd)
	case RssQbtSearch:
		m.qbtSearchModel, cmd = m.qbtSearchModel.Update(msg)
		cmds = append(cmds, cmd)
	default: // FIXME: remove this once refactor is complete
	}

	return m, tea.Batch(cmds...)
}

func (m RssMainModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(ui.DisplayError(m.err))
	}

	switch m.view {
	case RssSelection:
		return m.ViewSelection()
	case RssSearch:
		return m.searchModel.View()
	case RssQbtSearch:
		return m.qbtSearchModel.View()
	default: // FIXME: remove this once refactor is complete
	}
	return tea.NewView("")
}

func (m RssMainModel) ViewSelection() tea.View {
	if !m.isQbtOnline {
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle("RSS", "Offline"),
			ui.DisplayText([]string{
				`It appears that your qBittorrent client is ;r;Offline;x;,
which means you can't currently use the ;dc;automatic;x; rss option.`,
			}, 1, 1, 1),
			m.consent.View(ui.ConsentStyle.Render("Would you like to try again?")),
		))
	}

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS", "Lookup Method"),
		"",
		ui.DisplayText([]string{
			`Because you have ;dg;qBittorrent;x; setup, you have two options
for looking up RSS feeds.`,
			`;b;Manual:;x; provides you with an input box to manually search
for the desired fansub and anime name. It also provides you with a feed
link after each search.`,
			`;b;Automatic:;x; shows a list of your currently added anime
and allows you to bind your search to that anime. Once you find the fansub
you want, you can auto-add it to ;dg;qBittorrent;x; and it will begin
downloading immediately.`,
		}, 1, 0, 1),
		m.menu.View(),
	))
}

func (m RssMainModel) ShortHelp() []key.Binding {
	switch m.view {
	case RssSelection:
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
	default:
		return nil
	}
}

func (m RssMainModel) FullHelp() [][]key.Binding {
	switch m.view {
	case RssSelection:
		return nil
	default:
		return nil
	}
}

func (m RssMainModel) testConn() tea.Cmd {
	return func() tea.Msg {
		port := strconv.Itoa(m.db.Profile().QbtPort)
		err := qbittorrent.CheckConn(port)
		if err != nil {
			return QbtConnMsg{false}
		}
		return QbtConnMsg{true}
	}
}

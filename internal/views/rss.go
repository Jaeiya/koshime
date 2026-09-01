package views

import (
	"errors"
	"fmt"
	"strconv"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/qbittorrent"
	"github.com/Jaeiya/koshime/internal/ui"
)

type RssView int

const (
	RssSelection = RssView(iota)
	RssSearch
	RssQbtSearch
)

type RssMenuOption int

const (
	RssManualOpt = RssMenuOption(iota)
	RssAutoOpt
)

type (
	RssRouteMsg       struct{ View RssView }
	QbtConnMsg        struct{ isOnline bool }
	QbtSavedMsg       struct{ err error }
	QbtRemovedFeedMsg bool
	ParseRssMsg       struct{ Value app.RSSResult }
	ParsedRssResults  struct {
		List    list.Model
		Fansubs []app.FansubFileInfo
		Err     error
	}
	SearchFansubsMsg struct{ Query string }
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

	case ParseRssMsg:
		return m, m.parseRss(msg.Value)

	case SearchFansubsMsg:
		return m, m.search(msg.Query)

	case ui.MenuItemSelMsg:
		switch RssMenuOption(msg.Value) {
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
	case RssSearch:
		return m.searchModel.ShortHelp()
	case RssQbtSearch:
		return m.qbtSearchModel.ShortHelp()
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

func (m RssMainModel) search(query string) tea.Cmd {
	return func() tea.Msg {
		var rss app.RSS
		result, err := rss.FindAnimeFansub(app.Nyaa, query)
		if err != nil {
			return err
		}
		return result
	}
}

func (m RssMainModel) parseRss(r app.RSSResult) tea.Cmd {
	return func() tea.Msg {
		var parser app.FansubParser
		items := make([]list.Item, 0, len(r.Entries))
		rssFansubs := make([]app.FansubFileInfo, 0, len(r.Entries))

		count := 0
		for _, entry := range r.Entries {
			info, err := parser.Parse(entry.Title)
			if err != nil {
				if errors.Is(err, app.ErrBatchFile) {
					continue
				}
				return ParsedRssResults{List: list.Model{}, Fansubs: nil, Err: err}
			}

			// If a release doesn't have a readable fansub group name
			// then we consider it suspicious.
			if info.Fansub == "" {
				continue
			}

			// If a fansub file name does not contain "batch", but doesn't
			// include an episode #, then it's usually a batch release.
			if info.Episode == "" {
				continue
			}

			rssFansubs = append(rssFansubs, info)

			dateStr := entry.Date.Local().Format("Jan 2, 2006 at 3:04pm")
			seasonStr := ""
			if info.Season != "" {
				seasonStr = " | S" + info.Season
			}
			items = append(
				items, ui.NewListItem(
					fmt.Sprintf("[%s] %s - %s", info.Fansub, info.Title, info.Episode),
					fmt.Sprintf(
						"%s | %s | %s%s",
						dateStr,
						entry.Size,
						info.Encoding,
						seasonStr,
					),
					count,
				),
			)
			count++
		}

		return ParsedRssResults{
			List: ui.NewList(
				ui.ListOptions{
					Items:         items,
					ShortHelpKeys: []key.Binding{ui.KeyMap.Select, ui.KeyMap.EscBack},
					Width:         m.windowSize.Width - 3,
					MaxHeight:     int(float64(m.windowSize.Height) * 0.66),
					ItemsPerPage:  5,
				},
			),
			Fansubs: rssFansubs,
			Err:     nil,
		}
	}
}

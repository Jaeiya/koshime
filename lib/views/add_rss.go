package views

import (
	"errors"
	"fmt"

	"github.com/Jaeiya/koshime/lib"
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type Rss_View int

const (
	Rss_Search = Rss_View(iota)
	Rss_Review
)

type Rss_Model struct {
	windowSize tea.WindowSizeMsg
	ui         struct {
		loader  ui.LoaderModel
		list    list.Model
		input   textinput.Model
		consent ui.ConsentModel
	}
	config struct {
		minInputLen int
	}
	db    *database.Database
	state Rss_State
}

type Rss_State struct {
	err       error
	view      Rss_View
	rssResult lib.RSSResult
}

func newRssModel(db *database.Database) Rss_Model {
	m := Rss_Model{db: db}
	m.ui.list = ui.NewList(ui.ListOptions{})
	m.ui.input = ui.NewTextInput()
	m.ui.input.SetWidth(30)
	m.ui.loader = ui.NewLoader()
	m.config.minInputLen = 5
	return m
}

func (m Rss_Model) Init() tea.Cmd {
	return nil
}

func (m Rss_Model) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			if m.state.view == Rss_Search {
				return m, exitToMenu
			}
		}

	case DefaultErrorMsg:
		m.state.err = msg
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case Rss_Search:
		m, cmd = m.UpdateSearch(msg)
		cmds = append(cmds, cmd)
	case Rss_Review:
		m, cmd = m.UpdateReview(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Rss_Model) View() (string, *tea.Cursor) {
	if m.state.err != nil {
		return ui.DisplayError(m.state.err), nil
	}

	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	switch m.state.view {
	case Rss_Search:
		return m.ViewSearch()
	case Rss_Review:
		return m.ViewReview(), nil
	default:
		return "Unknown view", nil
	}
}

func (m Rss_Model) ShortHelp() []key.Binding {
	switch m.state.view {
	case Rss_Search:
		return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.MainMenu}
	case Rss_Review:
		return []key.Binding{ui.KeyMap.EscBack}
	}

	return []key.Binding{}
}

func (m Rss_Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m Rss_Model) UpdateSearch(msg tea.Msg) (Rss_Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Submit):
			if utils.RuneCount(m.ui.input.Value()) <= m.config.minInputLen {
				break
			}
			m.ui.loader, cmd = m.ui.loader.Start("Searching")
			return m, tea.Batch(cmd, m.search(m.ui.input.Value()))
		}

	case lib.RSSResult:
		var parser lib.FansubParser
		m.state.rssResult = msg
		items := []list.Item{}

		for i, entry := range m.state.rssResult.Entries {
			info, err := parser.Parse(entry.Title)
			if err != nil {
				if errors.Is(err, lib.ErrBatchFile) {
					continue
				}
				m.state.err = err
				return m, nil
			}
			dateStr := entry.Date.Local().Format("Jan 2, 2006 at 3:04pm")
			seasonStr := ""
			if info.Season != "" {
				seasonStr = " | S" + info.Season
			}
			items = append(
				items,
				ui.NewListItem(
					fmt.Sprintf("[%s] %s - %s", info.Fansub, info.Title, info.Episode),
					fmt.Sprintf("%s | %s | %s%s", dateStr, entry.Size, info.Encoding, seasonStr),
					i,
				),
			)
		}

		m.ui.list = ui.NewList(
			ui.ListOptions{
				Items:         items,
				ShortHelpKeys: []key.Binding{ui.KeyMap.Back},
				Width:         m.windowSize.Width - 3,
				MaxHeight:     int(float64(m.windowSize.Height) * 0.66),
				ItemsPerPage:  3,
			},
		)
		m.ui.loader.Stop()
		m.state.view = Rss_Review

	}

	m.ui.input, cmd = m.ui.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Rss_Model) ViewSearch() (string, *tea.Cursor) {
	c := m.ui.input.Cursor()
	c.Shape = tea.CursorBar

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("RSS Lookup"),
		"",
		ui.DisplayText([]string{
			`Here's an example of a typical search:`,
			`;dc;asw solo leveling 1080p`,
			`So if you were searching for the fansub group ;dc;asw;x;, the anime
;dc;solo leveling;x;, and the resolution ;dc;1080p;x;, then entering the above
line would give you those results.`,
		}, 1, 0, 0),
		ui.Style.MarginTop(1).Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.ui.input.View(),
			ui.DisplayCharLimit(m.config.minInputLen, m.ui.input.Value()),
		)),
	)
	c.Y = lipgloss.Height(view) - 1
	return view, c
}

func (m Rss_Model) UpdateReview(msg tea.Msg) (Rss_Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			m.state.view = Rss_Search
			return m, nil
		}
	}

	m.ui.consent = m.ui.consent.Update(msg)
	m.ui.list, cmd = m.ui.list.Update(msg)
	return m, cmd
}

func (m Rss_Model) ViewReview() string {
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS Lookup", "Selection"),
		"",
		ui.DisplayText([]string{";w;Feed URL:"}),
		ui.Style.MarginLeft(3).
			Render(utils.ColorText(fmt.Sprintf(";dg;%s", m.state.rssResult.FeedURL))),
		"",
		ui.DisplayText([]string{";w;Feed Results:"}),
		"",
		ui.Style.MarginLeft(3).Render(m.ui.list.View()),
		"",
	)
	return view
}

func (m Rss_Model) search(query string) tea.Cmd {
	return func() tea.Msg {
		var rss lib.RSS
		result, err := rss.FindAnimeFansub(lib.Nyaa, query)
		if err != nil {
			return DefaultErrorMsg{err}
		}

		return result
	}
}

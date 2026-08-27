package views

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
)

type RssSearchModel struct {
	loader       ui.LoaderModel
	input        textinput.Model
	list         list.Model
	windowSize   tea.WindowSizeMsg
	minInputLen  int
	searchResult app.RSSResult
}

func newRssSearchModel() RssSearchModel {
	m := RssSearchModel{}
	m.input = ui.NewTextInput()
	m.input.Placeholder = "<fansub anime search terms>"
	m.minInputLen = 5
	m.input.SetWidth(30)
	m.loader = ui.NewLoader()
	m.list = ui.NewList(ui.ListOptions{})
	return m
}

func (m RssSearchModel) Update(msg tea.Msg) (RssSearchModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			if m.hasResults() {
				m.searchResult = app.RSSResult{}
				return m, nil
			}
			return m, func() tea.Msg { return RssRouteMsg{RssSelection} }

		case key.Matches(msg, ui.KeyMap.Submit):
			// Do not submit if we already have results
			if m.hasResults() {
				return m, nil
			}
			if utils.RuneCount(m.input.Value()) < m.minInputLen {
				break
			}
			m.loader, cmd = m.loader.Start("Searching")
			return m, tea.Batch(cmd, func() tea.Msg { return SearchFansubsMsg{m.input.Value()} })
		}

	case app.RSSResult:
		m.searchResult = msg
		return m, func() tea.Msg { return ParseRssMsg{Value: msg} }

	case ParsedRssResults:
		m.loader.Stop()
		if msg.Err != nil {
			return m, func() tea.Msg { return msg.Err }
		}
		m.list = msg.List
	}

	if m.loader.IsLoading() {
		m.loader, cmd = m.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	if !m.hasResults() {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m RssSearchModel) View() tea.View {
	if m.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.loader.View()))
	}
	if !m.hasResults() {
		return m.ViewSearch()
	}
	return m.ViewReview()
}

func (m RssSearchModel) ShortHelp() []key.Binding {
	if m.loader.IsLoading() {
		return nil
	}
	if !m.hasResults() {
		return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.MainMenu}
	}
	return nil
}

func (m RssSearchModel) ViewSearch() tea.View {
	view := tea.NewView("")
	view.Cursor = m.input.Cursor()
	view.Cursor.Shape = tea.CursorBar

	view.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS", "Manual Lookup"),
		"",
		ui.DisplayText([]string{
			`Here's an example of a typical search:`,
			`;dc;asw solo leveling 1080p`,
			`So if you were searching for the fansub group ;dc;asw;x;, the anime
;dc;solo leveling;x;, and the resolution ;dc;1080p;x;, then entering the above
line would give you those results.`,
		}, 1, 0, 1),
		ui.Style.Render(lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.input.View(),
			ui.DisplayCharLimit(m.minInputLen, m.input.Value()),
		)),
	)

	view.Cursor.Y = lipgloss.Height(view.Content) - 1
	return view
}

func (m RssSearchModel) ViewReview() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS", "Selection"),
		"",
		ui.DisplayText([]string{";w;Feed URL:"}),
		ui.Style.MarginLeft(3).
			Render(utils.ColorText(fmt.Sprintf(";dg;%s", m.searchResult.FeedURL))),
		"",
		ui.DisplayText([]string{";w;Feed Results:"}),
		"",
		ui.Style.MarginLeft(3).Render(m.list.View()),
		"",
	))
}

func (m RssSearchModel) hasResults() bool {
	return m.searchResult.Host != ""
}

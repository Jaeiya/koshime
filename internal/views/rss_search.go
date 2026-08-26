package views

import (
	"errors"
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
	consent      ui.ConsentModel
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
			if m.searchResult.Host != "" {
				m.searchResult = app.RSSResult{}
				return m, nil
			}
			return m, func() tea.Msg { return RssRouteMsg{RssSelection} }

		case key.Matches(msg, ui.KeyMap.Submit):
			if utils.RuneCount(m.input.Value()) < m.minInputLen {
				break
			}
			m.loader, cmd = m.loader.Start("Searching")
			return m, tea.Batch(cmd, m.search(m.input.Value()))
		}

	case app.RSSResult:
		var err error
		m.list, _, err = m.parseRssResult(msg)
		m.loader.Stop()
		if err != nil {
			return m, func() tea.Msg { return err }
		}
		m.searchResult = msg
	}

	if m.loader.IsLoading() {
		m.loader, cmd = m.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.searchResult.Host {
	case "":
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	default:
		m.consent = m.consent.Update(msg)
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m RssSearchModel) View() tea.View {
	if m.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.loader.View()))
	}

	switch m.searchResult.Host {
	case "":
		return m.ViewSearch()
	default:
		return m.ViewReview()
	}
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

func (m RssSearchModel) search(query string) tea.Cmd {
	return func() tea.Msg {
		var rss app.RSS
		result, err := rss.FindAnimeFansub(app.Nyaa, query)
		if err != nil {
			return err
		}
		return result
	}
}

func (m RssSearchModel) parseRssResult(
	r app.RSSResult,
) (list.Model, []app.FansubFileInfo, error) {
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
			return list.Model{}, nil, err
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
				fmt.Sprintf("%s | %s | %s%s", dateStr, entry.Size, info.Encoding, seasonStr),
				count,
			),
		)
		count++
	}

	return ui.NewList(
		ui.ListOptions{
			Items:         items,
			ShortHelpKeys: []key.Binding{ui.KeyMap.Select, ui.KeyMap.Back},
			Width:         m.windowSize.Width - 3,
			MaxHeight:     int(float64(m.windowSize.Height) * 0.66),
			ItemsPerPage:  5,
		},
	), rssFansubs, nil
}

package views

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/qbittorrent"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
)

type RssQbtSearchModel struct {
	db             *database.Database
	loader         ui.LoaderModel
	input          textinput.Model
	consent        ui.ConsentModel
	animeList      list.Model
	resultRssList  list.Model
	refinedRssList list.Model
	windowSize     tea.WindowSizeMsg
	queriedFansubs []app.FansubFileInfo
	selAnime       kitsu.Anime
	selFeed        app.RSSResult
	selAnimeIdx    int
	minInputLen    int
	isSaved        bool
	lastInput      string
	noResults      bool
}

func newRssQbtSearchModel(db *database.Database) RssQbtSearchModel {
	m := RssQbtSearchModel{db: db}
	m.input = ui.NewTextInput()
	m.input.Placeholder = "<anime fansub search terms>"
	m.loader = ui.NewLoader()
	m.input.SetWidth(30)
	m.minInputLen = 3
	m.selAnimeIdx = -1
	return m
}

func (m RssQbtSearchModel) Update(msg tea.Msg) (RssQbtSearchModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.loader.IsLoading() {
		m.loader, cmd = m.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	if !m.hasRssResult() {
		m, cmd = m.UpdateSearch(msg)
		cmds = append(cmds, cmd)
	} else {
		m, cmd = m.UpdateBinding(msg)
		cmds = append(cmds, cmd)
	}

	if m.hasRssResult() && !m.hasRefinedResult() {
		m.resultRssList, cmd = m.resultRssList.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.hasRefinedResult() {
		m.refinedRssList, cmd = m.refinedRssList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m RssQbtSearchModel) UpdateSearch(msg tea.Msg) (RssQbtSearchModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg
		m.animeList = m.createAnimeList()

	case QbtRemovedFeedMsg:
		m.selAnime.QbtFeed = kitsu.QbtFeed{}

	case app.RSSResult:
		m.lastInput = m.input.Value()
		m.noResults = false
		return m, m.parseRss(msg)

	case ParsedRssResults:
		m.loader.Stop()
		if msg.Err != nil {
			return m, m.sendErr(msg.Err)
		}
		m.resultRssList = msg.List
		m.queriedFansubs = msg.Fansubs
		if !m.hasRssResult() {
			m.noResults = true
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			if m.animeList.FilterState() == list.Filtering {
				break
			}
			if m.hasFeedConflict() {
				m = m.reset()
				return m, nil
			}
			if m.hasSelectedAnime() {
				m.selAnimeIdx = -1
			} else {
				return m, func() tea.Msg { return RssRouteMsg{RssSelection} }
			}

		case key.Matches(msg, ui.KeyMap.Select):
			if m.animeList.FilterState() == list.Filtering {
				break
			}

			if m.hasFeedConflict() {
				if m.consent.Select() == ui.No {
					m = m.reset()
					return m, nil
				}
				return m, m.removeFeed()
			}

			if m.hasSelectedAnime() {
				if utils.RuneCount(m.input.Value()) < m.minInputLen {
					break
				}
				m.loader, cmd = m.loader.Start("Searching")
				return m, tea.Batch(cmd, m.searchFansubs(m.input.Value()))
			}

			// Save selected anime
			//nolint:errcheck // it will ALWAYS be a list item
			m.selAnimeIdx = m.animeList.SelectedItem().(ui.ListItem).Index()
			m.selAnime = m.db.Anime()[m.selAnimeIdx]
		}
	}

	// Get user consent for overriding existing feed
	if m.hasFeedConflict() {
		m.consent = m.consent.Update(msg)
	}

	if m.hasSelectedAnime() && !m.hasFeedConflict() {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	if !m.hasSelectedAnime() {
		m.animeList, cmd = m.animeList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m RssQbtSearchModel) UpdateBinding(msg tea.Msg) (RssQbtSearchModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			if m.hasRefinedResult() {
				m.refinedRssList = list.Model{}
				m.selFeed = app.RSSResult{}
				return m, nil
			}
			if m.hasRssResult() {
				m.resultRssList = list.Model{}
				m.input.Reset()
				return m, nil
			}

		case key.Matches(msg, ui.KeyMap.Select):
			if m.isSaved {
				m = m.reset()
				return m, nil
			}
			if m.hasSelectedFeed() {
				return m, m.saveQbtFeed(m.selFeed.FeedURL)
			}

			//nolint:errcheck // it will ALWAYS be a list item
			itemIdx := m.resultRssList.SelectedItem().(ui.ListItem).Index()
			fansubInfo := m.queriedFansubs[itemIdx]

			searchStr := strings.TrimSpace(fmt.Sprintf(
				"%s %s %s",
				fansubInfo.Fansub,
				fansubInfo.Title,
				m.findResolution(fansubInfo.Encoding),
			))

			if fansubInfo.Season != "" {
				ordinalSeason, err := utils.OrdinalString(fansubInfo.Season)
				if err != nil {
					return m, func() tea.Msg { return DefaultErrorMsg{err} }
				}
				searchStr = fmt.Sprintf(
					`%s ("s%s"|"season %s"|"%s season")`,
					searchStr,
					fansubInfo.Season,
					fansubInfo.Season,
					ordinalSeason,
				)
			}

			m.loader, cmd = m.loader.Start("Refining Feed")
			return m, tea.Batch(cmd, m.searchFansubs(searchStr))
		}

	case QbtSavedMsg:
		if msg.err != nil {
			return m, func() tea.Msg { return msg.err }
		}
		m.isSaved = true
		return m, nil

	case app.RSSResult:
		m.selFeed = msg
		return m, m.parseRss(msg)

	case ParsedRssResults:
		m.loader.Stop()
		if msg.Err != nil {
			return m, m.sendErr(msg.Err)
		}
		m.refinedRssList = msg.List
	}

	return m, cmd
}

func (m RssQbtSearchModel) View() tea.View {
	if m.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.loader.View()))
	}

	if !m.hasRssResult() {
		return m.ViewSearch()
	}

	return m.ViewBinding()
}

func (m RssQbtSearchModel) ShortHelp() []key.Binding {
	if m.loader.IsLoading() {
		return nil
	}
	if m.hasSelectedAnime() && !m.hasRssResult() {
		return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.EscBack}
	}
	return nil
}

func (m RssQbtSearchModel) ViewSearch() tea.View {
	view := tea.NewView("")

	if m.animeList.FilterState() == list.Filtering {
		view.Cursor = m.animeList.FilterInput.Cursor()
		view.Cursor.Color = lipgloss.Cyan
		view.Cursor.Y += 4
		view.Cursor.X += 5
	}

	switch {
	case !m.hasSelectedAnime():
		return m.ViewAnimeSelection(view)

	case m.hasFeedConflict():
		return m.ViewFeedConflict(view)

	case m.hasSelectedAnime():
		return m.ViewFeedBinding(view)
	}

	return tea.NewView("missing RSS QbtSearch view")
}

func (m RssQbtSearchModel) ViewBinding() tea.View {
	view := tea.NewView("")
	switch {
	case m.isSaved:
		return m.viewSavedBinding(view)
	case m.hasRefinedResult():
		return m.viewRefinedResults(view)
	default:
		return m.viewRssResults(view)
	}
}

func (m RssQbtSearchModel) ViewAnimeSelection(view tea.View) tea.View {
	view.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS", "Auto Lookup"),
		ui.DisplayText([]string{
			`;b;Select the anime you want to bind to an RSS feed:;x;`,
		}, 0, 1),
		ui.Style.MarginLeft(3).Render(m.animeList.View()),
	)
	return view
}

func (m RssQbtSearchModel) ViewFeedConflict(view tea.View) tea.View {
	view.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS", "Feed Conflict"),
		ui.DisplayText([]string{
			fmt.Sprintf(
				`The anime ;dg;%s;x; already has a feed attached to it.`,
				m.selAnime.ENG_Title,
			),
			`This means that it's ;w;already;x; attached to an existing
auto-download rule. If you continue, the feed will be removed.`,
		}, 1, 1, 1),
		m.consent.View(
			ui.ConsentStyle.Render("Do you want to override its existing feed?"),
		),
	)
	return view
}

func (m RssQbtSearchModel) ViewFeedBinding(view tea.View) tea.View {
	text := []string{
		`Here is a list of all titles associated with this anime. Most fansubs
use the official japanese name or closest alt title. If the title is very long,
they use either an alt or the ;w;first few words;x; of the japanese title.`,
	}

	text = append(text, fmt.Sprintf(";c;- ;db;%s;x;", m.selAnime.JPN_Title))
	text = append(text, fmt.Sprintf(";c;- ;db;%s;x;", m.selAnime.ENG_Title))

	for _, title := range m.selAnime.AltTitles {
		text = append(text, fmt.Sprintf(";c;- ;db;%s;x;", title))
	}
	text = append(text, []string{
		`Here's an example: ;dc;asw solo leveling 1080p s2;x;`,
		`As you can see, we used the fansub group ;dc;asw;x;, the title ;dc;solo
leveling;x;, the season ;dc;s2;x; and even the resolution ;dc;1080p;x;.`,
	}...)

	display := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS", "Fansub Lookup"),
		ui.DisplayText(text, 1, 1, 1),
	)

	if m.noResults {
		display += ui.DisplayText([]string{
			fmt.Sprintf(";y;Could not find results for: ;c;%s;x;", m.lastInput),
			";b;Try again:",
		}, 1, 1, 1)
	}

	view.Cursor = m.input.Cursor()
	view.Cursor.Shape = tea.CursorBar
	view.Cursor.Y += lipgloss.Height(display)

	view.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		display,
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.input.View(),
			ui.DisplayCharLimit(m.minInputLen, m.input.Value()),
		),
	)
	return view
}

func (m RssQbtSearchModel) viewRssResults(view tea.View) tea.View {
	view.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS", "Feed Refiner"),
		ui.DisplayText([]string{
			`;b;Select a fansub feed from the list below that most closely
matches the anime release you're looking for.`,
		}, 1, 1, 1),
		ui.DisplayText([]string{";w;Feed Results:"}, 1, 0, 1),
		ui.Style.MarginLeft(3).Render(m.resultRssList.View()),
	)
	return view
}

func (m RssQbtSearchModel) viewRefinedResults(view tea.View) tea.View {
	view.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS", "Save Feed"),
		ui.DisplayText([]string{
			`;b;Make sure the anime feed results below look like the
releases you're looking for.`,
		}, 1, 1, 1),
		ui.DisplayText([]string{";w;Feed Results:"}, 1, 0, 1),
		ui.Style.MarginLeft(3).Render(m.refinedRssList.View()),
	)
	return view
}

func (m RssQbtSearchModel) viewSavedBinding(view tea.View) tea.View {
	title := m.selAnime.ENG_Title
	if title == "" {
		title = m.selAnime.JPN_Title
	}
	view.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("RSS", "Feed Saved"),
		ui.DisplayText([]string{
			fmt.Sprintf(
				`;dc;%s;x; should now be downloading ;g;successfully;x; every week!`,
				title,
			),
		}, 1, 1, 1),
		ui.Style.MarginLeft(3).
			Foreground(lipgloss.BrightGreen).
			Background(lipgloss.Black).
			Render("> Continue "),
	)
	return view
}

func (m RssQbtSearchModel) reset() RssQbtSearchModel {
	m.resultRssList = list.Model{}
	m.refinedRssList = list.Model{}
	m.selFeed = app.RSSResult{}
	m.selAnimeIdx = -1
	return m
}

func (m RssQbtSearchModel) removeFeed() tea.Cmd {
	return func() tea.Msg {
		port := strconv.Itoa(m.db.Profile().QbtPort)
		qb, err := qbittorrent.NewLogin(port)
		if err != nil {
			return fmt.Errorf("removing feed failed: %w", err)
		}
		defer qb.Logout()

		err = qb.DeleteFeed(m.selAnime.QbtFeed.Name)
		if err != nil {
			return err
		}

		err = qb.DeleteRuleFeed(kitsu.RssRuleName, m.selAnime.QbtFeed.RuleURI)
		if err != nil {
			return fmt.Errorf("deleting rule failed: %w", err)
		}

		m.selAnime.QbtFeed = kitsu.QbtFeed{}

		err = m.db.UpdateAnime(m.selAnime)
		if err != nil {
			return err
		}

		return QbtRemovedFeedMsg(true)
	}
}

func (m RssQbtSearchModel) saveQbtFeed(feed string) tea.Cmd {
	return func() tea.Msg {
		port := strconv.Itoa(m.db.Profile().QbtPort)
		qb, err := qbittorrent.NewLogin(port)
		if err != nil {
			return QbtSavedMsg{err}
		}

		feedName := fmt.Sprintf("%s (%s)", m.selAnime.ENG_Title, m.selAnime.JPN_Title)
		err = qb.AddFeed(feedName, feed)
		if err != nil {
			return QbtSavedMsg{err}
		}

		err = qb.AddRuleFeed(kitsu.RssRuleName, feed)
		if err != nil {
			return QbtSavedMsg{err}
		}

		m.selAnime.QbtFeed = kitsu.QbtFeed{
			Name:    feedName,
			RuleURI: feed,
		}
		err = m.db.UpdateAnime(m.selAnime)
		if err != nil {
			return QbtSavedMsg{err}
		}

		return QbtSavedMsg{nil}
	}
}

func (m RssQbtSearchModel) createAnimeList() list.Model {
	anime := m.db.Anime()
	items := make([]list.Item, len(anime))
	for i, entry := range anime {
		items[i] = ui.NewListItem(entry.JPN_Title, entry.ENG_Title, i)
	}

	return ui.NewList(ui.ListOptions{
		Items:         items,
		Width:         m.windowSize.Width - 3,
		ShortHelpKeys: []key.Binding{ui.KeyMap.Select, ui.KeyMap.EscBack},
		MaxHeight:     int(float64(m.windowSize.Height) * 0.66),
		ItemsPerPage:  4,
		EnableFilter:  true,
	})
}

func (m RssQbtSearchModel) findResolution(encoding string) string {
	resolutions := [...]string{"480p", "720p", "1080p"}

	for _, res := range resolutions {
		if strings.Contains(encoding, res) {
			return res
		}
	}
	return ""
}

func (m RssQbtSearchModel) searchFansubs(query string) tea.Cmd {
	return func() tea.Msg {
		return SearchFansubsMsg{query}
	}
}

func (m RssQbtSearchModel) parseRss(r app.RSSResult) tea.Cmd {
	return func() tea.Msg {
		return ParseRssMsg{r}
	}
}

func (m RssQbtSearchModel) sendErr(err error) tea.Cmd {
	return func() tea.Msg {
		return err
	}
}

func (m RssQbtSearchModel) hasFeedConflict() bool {
	return m.selAnimeIdx > -1 && m.selAnime.QbtFeed.Name != ""
}

func (m RssQbtSearchModel) hasSelectedAnime() bool {
	return m.selAnimeIdx > -1
}

func (m RssQbtSearchModel) hasRssResult() bool {
	return len(m.resultRssList.VisibleItems()) > 0
}

func (m RssQbtSearchModel) hasRefinedResult() bool {
	return m.hasRssResult() && len(m.refinedRssList.VisibleItems()) > 0
}

func (m RssQbtSearchModel) hasSelectedFeed() bool {
	return len(m.selFeed.Entries) > 0
}

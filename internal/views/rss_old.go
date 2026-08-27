package views

// type RssModel struct {
// 	windowSize tea.WindowSizeMsg
// 	ui         struct {
// 		loader    ui.LoaderModel
// 		list      list.Model
// 		animeList list.Model
// 		selMenu   ui.MenuModel
// 		input     textinput.Model
// 		consent   ui.ConsentModel
// 	}
// 	config struct {
// 		minInputLen int
// 	}
// 	db    *database.Database
// 	state RssState
// }

// type RssState struct {
// 	err           error
// 	view          RssView
// 	rssResult     app.RSSResult
// 	refinedResult app.RSSResult
// 	parsedFansubs []app.FansubFileInfo
// 	selAnimeIdx   int
// 	isOffline     bool
// 	isRssRefined  bool
// 	saveStatus    struct {
// 		anime kitsu.Anime
// 		saved bool
// 		err   error
// 	}
// }

// func newRssModel(db *database.Database) RssModel {
// 	m := RssModel{db: db}
// 	m.ui.list = ui.NewList(ui.ListOptions{})
// 	m.ui.animeList = ui.NewList(ui.ListOptions{})
// 	m.ui.input = ui.NewTextInput()
// 	m.ui.input.Placeholder = "<fansub anime search terms>"
// 	m.ui.input.SetWidth(30)
// 	m.ui.loader = ui.NewLoader()
// 	m.ui.selMenu = ui.NewMenuModel(menuOptions[:])
// 	m.config.minInputLen = 5

// 	p := db.Profile()
// 	if p.QbtPort == 0 {
// 		m.state.view = RssSearch
// 	}
// 	m.state.selAnimeIdx = -1
// 	return m
// }

// func (m RssModel) Init() tea.Cmd {
// 	return nil
// }

// func (m RssModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
// 	var cmd tea.Cmd
// 	var cmds []tea.Cmd

// 	switch msg := msg.(type) {
// 	case tea.WindowSizeMsg:
// 		m.windowSize = msg

// 	case tea.KeyPressMsg:
// 		switch {
// 		case key.Matches(msg, ui.KeyMap.MainMenu):
// 			// Prevent state corruption
// 			if m.ui.loader.IsLoading() {
// 				return m, cmd
// 			}
// 			if m.state.view == RssSearch {
// 				return m, exitToMenu
// 			}
// 			// Reset on error
// 			if m.state.err != nil {
// 				m.state.err = nil
// 				m.resetFlowState()
// 				return m, nil
// 			}

// 		case key.Matches(msg, ui.KeyMap.Back):
// 			// Prevent state corruption
// 			if m.ui.loader.IsLoading() {
// 				return m, cmd
// 			}
// 			// Reset on error
// 			if m.state.err != nil {
// 				m.state.err = nil
// 				m.resetFlowState()
// 				return m, nil
// 			}
// 		}

// 	case DefaultErrorMsg:
// 		m.state.err = msg
// 	}

// 	// Prevent any actions when in error state
// 	if m.state.err != nil {
// 		return m, nil
// 	}

// 	if m.ui.loader.IsLoading() {
// 		m.ui.loader, cmd = m.ui.loader.Update(msg)
// 		cmds = append(cmds, cmd)
// 	}

// 	switch m.state.view {
// 	case RssSelection:
// 		m, cmd = m.UpdateSelection(msg)
// 		cmds = append(cmds, cmd)
// 	case RssSearch:
// 		m, cmd = m.UpdateSearch(msg)
// 		cmds = append(cmds, cmd)
// 	case RssReview:
// 		m, cmd = m.UpdateReview(msg)
// 		cmds = append(cmds, cmd)
// 	case RssQbtSearch:
// 		m, cmd = m.UpdateQbtSearch(msg)
// 		cmds = append(cmds, cmd)
// 	case RssQbtReview:
// 		m, cmd = m.UpdateQbtReview(msg)
// 		cmds = append(cmds, cmd)
// 	}

// 	return m, tea.Batch(cmds...)
// }

// func (m RssModel) View() tea.View {
// 	if m.state.err != nil {
// 		return tea.NewView(ui.DisplayError(m.state.err))
// 	}

// 	if m.ui.loader.IsLoading() {
// 		return tea.NewView(ui.Style.MarginTop(1).Render(m.ui.loader.View()))
// 	}

// 	switch m.state.view {
// 	case RssSelection:
// 		return m.ViewSelection()
// 	case RssSearch:
// 		return m.ViewSearch()
// 	case RssReview:
// 		return m.ViewReview()
// 	case RssQbtSearch:
// 		return m.ViewQbtSearch()
// 	case RssQbtReview:
// 		return m.ViewQbtReview()

// 	default:
// 		return tea.NewView("missing RSS model view")
// 	}
// }

// func (m RssModel) ShortHelp() []key.Binding {
// 	if m.state.err != nil {
// 		return []key.Binding{ui.KeyMap.EscBack}
// 	}

// 	switch m.state.view {
// 	case RssSelection:
// 		return []key.Binding{ui.KeyMap.Select, ui.KeyMap.MainMenu}
// 	case RssSearch:
// 		return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.MainMenu}
// 	case RssReview:
// 		return []key.Binding{ui.KeyMap.EscBack}
// 	case RssQbtSearch:
// 		if m.state.selAnimeIdx > -1 && m.state.saveStatus.anime.QbtFeed.Name != "" {
// 			return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
// 		}
// 		if m.state.selAnimeIdx > -1 {
// 			return []key.Binding{ui.KeyMap.Submit, ui.KeyMap.Abort}
// 		}
// 	case RssQbtReview:
// 		if m.state.err != nil || m.state.saveStatus.err != nil {
// 			return []key.Binding{ui.KeyMap.EscBack}
// 		}
// 		if m.state.saveStatus.saved {
// 			return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select}
// 		}
// 	}

// 	return []key.Binding{}
// }

// func (m RssModel) FullHelp() [][]key.Binding {
// 	return [][]key.Binding{}
// }

// func (m RssModel) UpdateSelection(msg tea.Msg) (RssModel, tea.Cmd) {
// 	switch msg := msg.(type) {
// 	case tea.KeyPressMsg:
// 		switch {
// 		case key.Matches(msg, ui.KeyMap.EscBack):
// 			return m, exitToMenu

// 		case key.Matches(msg, ui.KeyMap.Select):
// 			if m.state.isOffline {
// 				if m.ui.consent.Select() == ui.No {
// 					m.state.view = RssSearch
// 					return m, nil
// 				}
// 				return m, m.testConn()
// 			}

// 		}

// 	case QbtConnMsg:
// 		if !msg.isOnline {
// 			m.state.isOffline = true
// 			return m, nil
// 		}
// 		m.state.view = RssQbtSearch
// 		m.ui.animeList = m.createAnimeList()

// 	case ui.MenuIndexMsg:
// 		option := RssMenuOption(msg)
// 		switch option {
// 		case RssManualOpt:
// 			m.state.view = RssSearch
// 		case RssAutoOpt:
// 			return m, m.testConn()
// 		}
// 	}

// 	if m.state.isOffline {
// 		m.ui.consent = m.ui.consent.Update(msg)
// 		return m, nil
// 	}

// 	var cmd tea.Cmd
// 	m.ui.selMenu, cmd = m.ui.selMenu.Update(msg)

// 	return m, cmd
// }

// func (m RssModel) ViewSelection() tea.View {
// 	if m.state.isOffline {
// 		return tea.NewView(lipgloss.JoinVertical(
// 			lipgloss.Left,
// 			ui.DisplaySubTitle("RSS", "Offline"),
// 			ui.DisplayText([]string{
// 				`It appears that your qBittorrent client is ;r;Offline;x;,
// which means you can't currently use the ;dc;automatic;x; rss option.`,
// 			}, 1, 1, 1),
// 			m.ui.consent.View(ui.ConsentStyle.Render("Would you like to try again?")),
// 		))
// 	}

// 	return tea.NewView(lipgloss.JoinVertical(
// 		lipgloss.Left,
// 		ui.DisplaySubTitle("RSS", "Lookup Method"),
// 		"",
// 		ui.DisplayText([]string{
// 			`Because you have ;dg;qBittorrent;x; setup, you have two options
// for looking up RSS feeds.`,
// 			`;b;Manual:;x; provides you with an input box to manually search
// for the desired fansub and anime name. It also provides you with a feed
// link after each search.`,
// 			`;b;Automatic:;x; shows a list of your currently added anime
// and allows you to bind your search to that anime. Once you find the fansub
// you want, you can auto-add it to ;dg;qBittorrent;x; and it will begin
// downloading immediately.`,
// 		}, 1, 0, 1),
// 		m.ui.selMenu.View(),
// 	))
// }

// func (m RssModel) UpdateSearch(msg tea.Msg) (RssModel, tea.Cmd) {
// 	var cmd tea.Cmd
// 	var cmds []tea.Cmd
// 	switch msg := msg.(type) {
// 	case tea.KeyPressMsg:
// 		switch {
// 		case key.Matches(msg, ui.KeyMap.Submit):
// 			if utils.RuneCount(m.ui.input.Value()) < m.config.minInputLen {
// 				break
// 			}
// 			m.ui.loader, cmd = m.ui.loader.Start("Searching")
// 			return m, tea.Batch(cmd, m.search(m.ui.input.Value()))
// 		}

// 	case app.RSSResult:
// 		var err error
// 		m.state.rssResult = msg
// 		m.ui.list, m.state.parsedFansubs, err = m.parseRssResult(msg)
// 		if err != nil {
// 			return m, func() tea.Msg { return DefaultErrorMsg{err} }
// 		}
// 		m.ui.loader.Stop()
// 		m.state.view = RssReview

// 	}

// 	m.ui.input, cmd = m.ui.input.Update(msg)
// 	cmds = append(cmds, cmd)

// 	return m, tea.Batch(cmds...)
// }

// func (m RssModel) ViewSearch() tea.View {
// 	view := tea.NewView("")
// 	view.Cursor = m.ui.input.Cursor()
// 	view.Cursor.Shape = tea.CursorBar

// 	view.Content = lipgloss.JoinVertical(
// 		lipgloss.Left,
// 		ui.DisplaySubTitle("RSS", "Manual Lookup"),
// 		"",
// 		ui.DisplayText([]string{
// 			`Here's an example of a typical search:`,
// 			`;dc;asw solo leveling 1080p`,
// 			`So if you were searching for the fansub group ;dc;asw;x;, the anime
// ;dc;solo leveling;x;, and the resolution ;dc;1080p;x;, then entering the above
// line would give you those results.`,
// 		}, 1, 0, 1),
// 		ui.Style.Render(lipgloss.JoinHorizontal(
// 			lipgloss.Left,
// 			m.ui.input.View(),
// 			ui.DisplayCharLimit(m.config.minInputLen, m.ui.input.Value()),
// 		)),
// 	)

// 	view.Cursor.Y = lipgloss.Height(view.Content) - 1
// 	return view
// }

// func (m RssModel) UpdateReview(msg tea.Msg) (RssModel, tea.Cmd) {
// 	var cmd tea.Cmd
// 	switch msg := msg.(type) {
// 	case tea.KeyPressMsg:
// 		switch {
// 		case key.Matches(msg, ui.KeyMap.EscBack):
// 			m.state.view = RssSearch
// 			return m, nil
// 		}
// 	}

// 	m.ui.consent = m.ui.consent.Update(msg)
// 	m.ui.list, cmd = m.ui.list.Update(msg)
// 	return m, cmd
// }

// func (m RssModel) ViewReview() tea.View {
// 	return tea.NewView(lipgloss.JoinVertical(
// 		lipgloss.Left,
// 		ui.DisplaySubTitle("RSS", "Selection"),
// 		"",
// 		ui.DisplayText([]string{";w;Feed URL:"}),
// 		ui.Style.MarginLeft(3).
// 			Render(utils.ColorText(fmt.Sprintf(";dg;%s", m.state.rssResult.FeedURL))),
// 		"",
// 		ui.DisplayText([]string{";w;Feed Results:"}),
// 		"",
// 		ui.Style.MarginLeft(3).Render(m.ui.list.View()),
// 		"",
// 	))
// }

// func (m RssModel) UpdateQbtSearch(msg tea.Msg) (RssModel, tea.Cmd) {
// 	var cmd tea.Cmd
// 	var cmds []tea.Cmd

// 	hasSelectedAnime := m.state.selAnimeIdx > -1
// 	hasFeedConflict := hasSelectedAnime && m.state.saveStatus.anime.QbtFeed.Name != ""

// 	switch msg := msg.(type) {
// 	case QbtRemovedFeedMsg:
// 		m.state.saveStatus.anime.QbtFeed = struct {
// 			Name    string
// 			RuleURI string
// 		}{}

// 	case app.RSSResult:
// 		var err error
// 		m.state.rssResult = msg
// 		m.ui.list, m.state.parsedFansubs, err = m.parseRssResult(msg)
// 		if err != nil {
// 			return m, func() tea.Msg { return DefaultErrorMsg{err} }
// 		}
// 		m.ui.loader.Stop()
// 		m.state.view = RssQbtReview

// 	case tea.KeyPressMsg:
// 		switch {
// 		case key.Matches(msg, ui.KeyMap.EscBack):
// 			// Do not allow aborting consent
// 			if hasFeedConflict {
// 				return m, nil
// 			}

// 			if msg.String() == "esc" || msg.String() == "backspace" {
// 				if m.ui.animeList.FilterState() > list.Unfiltered {
// 					break
// 				}
// 			}
// 			if msg.String() == "backspace" && m.state.selAnimeIdx > -1 {
// 				break
// 			}

// 			if hasSelectedAnime {
// 				m.state.selAnimeIdx = -1
// 			} else {
// 				m.state.view = RssSelection
// 			}

// 		case key.Matches(msg, ui.KeyMap.Select):
// 			if m.ui.list.FilterState() == list.Filtering {
// 				break
// 			}

// 			if hasFeedConflict {
// 				if m.ui.consent.Select() == ui.No {
// 					m.resetFlowState()
// 					return m, nil
// 				}
// 				return m, m.removeFeed()
// 			}

// 			// Lookup library anime
// 			if hasSelectedAnime {
// 				if utils.RuneCount(m.ui.input.Value()) < m.config.minInputLen {
// 					break
// 				}
// 				m.ui.loader, cmd = m.ui.loader.Start("Searching")
// 				return m, tea.Batch(cmd, m.search(m.ui.input.Value()))
// 			}

// 			// Save selected anime
// 			//nolint:errcheck // it will ALWAYS be a list item
// 			m.state.selAnimeIdx = m.ui.animeList.SelectedItem().(ui.ListItem).Index()
// 			m.state.saveStatus.anime = m.db.Anime()[m.state.selAnimeIdx]
// 		}
// 	}

// 	// Get user consent for overriding existing feed
// 	if hasFeedConflict {
// 		m.ui.consent = m.ui.consent.Update(msg)
// 	}

// 	if hasSelectedAnime && !hasFeedConflict {
// 		m.ui.input, cmd = m.ui.input.Update(msg)
// 		cmds = append(cmds, cmd)
// 	}

// 	if !hasSelectedAnime {
// 		m.ui.animeList, cmd = m.ui.animeList.Update(msg)
// 		cmds = append(cmds, cmd)
// 	}

// 	return m, tea.Batch(cmds...)
// }

// func (m RssModel) ViewQbtSearch() tea.View {
// 	view := tea.NewView("")

// 	if m.ui.animeList.FilterState() == list.Filtering {
// 		view.Cursor = m.ui.animeList.FilterInput.Cursor()
// 		view.Cursor.Color = lipgloss.Cyan
// 		view.Cursor.Y += 4
// 		view.Cursor.X += 5
// 	}

// 	anime := m.state.saveStatus.anime

// 	switch {
// 	// Anime selection view
// 	case m.state.selAnimeIdx == -1:
// 		view.Content = lipgloss.JoinVertical(
// 			lipgloss.Left,
// 			ui.DisplaySubTitle("RSS", "Auto Lookup"),
// 			"",
// 			ui.DisplayText([]string{
// 				`;b;Select the anime you want to bind to an RSS feed:;x;`,
// 			}),
// 			ui.Style.MarginLeft(3).Render(m.ui.animeList.View()),
// 		)
// 		return view

// 	// Anime feed conflict view
// 	case m.state.selAnimeIdx > -1 && anime.QbtFeed.Name != "":
// 		view.Content = lipgloss.JoinVertical(
// 			lipgloss.Left,
// 			ui.DisplaySubTitle("RSS", "Feed Conflict"),
// 			ui.DisplayText([]string{
// 				fmt.Sprintf(
// 					`The anime ;dg;%s;x; already has a feed attached to it.`,
// 					anime.ENG_Title,
// 				),
// 				`This means that it's ;w;already;x; attached to an existing
// auto-download rule. If you continue, the feed will be removed.`,
// 			}, 1, 1, 1),
// 			m.ui.consent.View(
// 				ui.ConsentStyle.Render("Do you want to override its existing feed?"),
// 			),
// 		)
// 		return view

// 	// Anime lookup view
// 	case m.state.selAnimeIdx > -1:
// 		text := []string{
// 			`Here is a list of all titles associated with this anime. You can either use
// the full title or words within the titles, as search terms.`,
// 		}

// 		text = append(text, fmt.Sprintf(";c;- ;db;%s;x;", anime.ENG_Title))
// 		text = append(text, fmt.Sprintf(";c;- ;db;%s;x;", anime.JPN_Title))

// 		for _, title := range anime.AltTitles {
// 			text = append(text, fmt.Sprintf(";c;- ;db;%s;x;", title))
// 		}
// 		text = append(text, []string{
// 			`Here's an example: ;dc;asw solo leveling 1080p s2;x;`,
// 			`As you can see, we used the fansub group ;dc;asw;x;, the title ;dc;solo
// leveling;x;, the season ;dc;s2;x; and even the resolution ;dc;1080p;x;.`,
// 		}...)

// 		display := lipgloss.JoinVertical(
// 			lipgloss.Left,
// 			ui.DisplaySubTitle("RSS", "Fansub Lookup"),
// 			ui.DisplayText(text, 1, 1, 1),
// 		)

// 		view.Cursor = m.ui.input.Cursor()
// 		view.Cursor.Shape = tea.CursorBar
// 		view.Cursor.Y += lipgloss.Height(display)

// 		view.Content = lipgloss.JoinVertical(
// 			lipgloss.Left,
// 			display,
// 			lipgloss.JoinHorizontal(
// 				lipgloss.Left,
// 				m.ui.input.View(),
// 				ui.DisplayCharLimit(m.config.minInputLen, m.ui.input.Value()),
// 			),
// 		)
// 		return view
// 	}

// 	return tea.NewView("missing RSS QbtSearch view")
// }

// func (m RssModel) UpdateQbtReview(msg tea.Msg) (RssModel, tea.Cmd) {
// 	var cmd tea.Cmd

// 	switch msg := msg.(type) {
// 	case QbtSavedMsg:
// 		if msg.err != nil {
// 			m.state.saveStatus.err = msg.err
// 			return m, nil
// 		}
// 		m.state.saveStatus.saved = true
// 		return m, nil

// 	case app.RSSResult:
// 		var err error
// 		m.state.refinedResult = msg
// 		m.ui.list, m.state.parsedFansubs, err = m.parseRssResult(msg)
// 		if err != nil {
// 			return m, func() tea.Msg { return DefaultErrorMsg{err} }
// 		}
// 		m.state.isRssRefined = true
// 		m.ui.loader.Stop()

// 	case tea.KeyPressMsg:
// 		switch {
// 		case key.Matches(msg, ui.KeyMap.EscBack):
// 			if m.state.saveStatus.saved {
// 				return m, nil
// 			}
// 			// Go back to previous results instead of forcing
// 			// a new search.
// 			if m.state.isRssRefined {
// 				var err error
// 				m.state.isRssRefined = false
// 				m.state.saveStatus.err = nil
// 				m.ui.list, m.state.parsedFansubs, err = m.parseRssResult(m.state.rssResult)
// 				if err != nil {
// 					return m, func() tea.Msg { return DefaultErrorMsg{err} }
// 				}
// 				return m, nil
// 			}
// 			m.state.view = RssQbtSearch
// 			m.ui.input.Reset()

// 		case key.Matches(msg, ui.KeyMap.Select):
// 			// Does the user want to add another feed?
// 			if m.state.saveStatus.saved {
// 				if m.ui.consent.Select() == ui.No {
// 					return m, exitToMenu
// 				}
// 				m.resetFlowState()
// 				m.state.view = RssQbtSearch
// 				return m, nil
// 			}

// 			if m.state.isRssRefined {
// 				return m, m.saveQbtFeed(m.state.refinedResult.FeedURL)
// 			}

// 			//nolint:errcheck // it will ALWAYS be a list item
// 			itemIdx := m.ui.list.SelectedItem().(ui.ListItem).Index()
// 			fansubInfo := m.state.parsedFansubs[itemIdx]

// 			searchStr := strings.TrimSpace(fmt.Sprintf(
// 				"%s %s %s",
// 				fansubInfo.Fansub,
// 				fansubInfo.Title,
// 				m.findResolution(fansubInfo.Encoding),
// 			))

// 			if fansubInfo.Season != "" {
// 				ordinalSeason, err := utils.OrdinalString(fansubInfo.Season)
// 				if err != nil {
// 					return m, func() tea.Msg { return DefaultErrorMsg{err} }
// 				}
// 				searchStr = fmt.Sprintf(
// 					`%s ("s%s"|"season %s"|"%s season")`,
// 					searchStr,
// 					fansubInfo.Season,
// 					fansubInfo.Season,
// 					ordinalSeason,
// 				)
// 			}

// 			m.ui.loader, cmd = m.ui.loader.Start("Refining Feed")
// 			return m, tea.Batch(cmd, m.search(searchStr))

// 		}
// 	}

// 	if !m.ui.loader.IsLoading() {
// 		m.ui.list, cmd = m.ui.list.Update(msg)
// 	}

// 	if m.state.saveStatus.saved {
// 		m.ui.consent = m.ui.consent.Update(msg)
// 	}

// 	return m, cmd
// }

// func (m RssModel) ViewQbtReview() tea.View {
// 	view := tea.NewView("")

// 	switch {
// 	case m.state.saveStatus.err != nil:
// 		view.Content = lipgloss.JoinVertical(
// 			lipgloss.Left,
// 			ui.DisplaySubTitle("RSS", "Save Error"),
// 			ui.DisplayText([]string{
// 				fmt.Sprintf(";y;Error Occurred: ;r;%s", m.state.saveStatus.err),
// 			}, 1, 1, 1),
// 		)

// 	case m.state.saveStatus.saved:
// 		view.Content = lipgloss.JoinVertical(
// 			lipgloss.Left,
// 			ui.DisplaySubTitle("RSS", "Feed Saved"),
// 			ui.DisplayText([]string{
// 				fmt.Sprintf(
// 					`;dc;%s;x; should now be downloading ;g;successfully;x;
// every week!`,
// 					m.state.saveStatus.anime.ENG_Title,
// 				),
// 			}, 1, 1, 1),
// 			m.ui.consent.View(ui.ConsentStyle.Render("Would you like to add another feed?")),
// 		)

// 	case m.state.isRssRefined:
// 		view.Content = lipgloss.JoinVertical(
// 			lipgloss.Left,
// 			ui.DisplaySubTitle("RSS", "Save Feed"),
// 			"",
// 			ui.DisplayText([]string{
// 				`Make sure the anime feed results below look like the
// releases you're looking for.`,
// 			}, 1),
// 			ui.DisplayText([]string{";w;Feed Results:"}, 1),
// 			ui.Style.MarginLeft(3).Render(m.ui.list.View()),
// 			"",
// 			ui.Style.MarginLeft(3).
// 				Foreground(lipgloss.BrightGreen).
// 				Background(lipgloss.Black).
// 				Render("> Save "),
// 		)

// 	default:
// 		view.Content = lipgloss.JoinVertical(
// 			lipgloss.Left,
// 			ui.DisplaySubTitle("RSS", "Feed Finder"),
// 			"",
// 			ui.DisplayText([]string{
// 				`Select an anime feed from the list that ;w;most closely;x;
// matches the release you're looking for. The results will then be filtered
// based on that selection.`,
// 			}, 1),
// 			ui.DisplayText([]string{";w;Feed Results:"}, 1),
// 			ui.Style.MarginLeft(3).Render(m.ui.list.View()),
// 		)
// 	}

// 	return view
// }

// func (m RssModel) parseRssResult(r app.RSSResult) (list.Model, []app.FansubFileInfo, error) {
// 	var parser app.FansubParser
// 	items := make([]list.Item, 0, len(r.Entries))
// 	rssFansubs := make([]app.FansubFileInfo, 0, len(r.Entries))

// 	count := 0
// 	for _, entry := range r.Entries {
// 		info, err := parser.Parse(entry.Title)
// 		if err != nil {
// 			if errors.Is(err, app.ErrBatchFile) {
// 				continue
// 			}
// 			return list.Model{}, nil, err
// 		}

// 		// If a release doesn't have a readable fansub group name
// 		// then we consider it suspicious.
// 		if info.Fansub == "" {
// 			continue
// 		}

// 		// If a fansub file name does not contain "batch", but doesn't
// 		// include an episode #, then it's usually a batch release.
// 		if info.Episode == "" {
// 			continue
// 		}

// 		rssFansubs = append(rssFansubs, info)

// 		dateStr := entry.Date.Local().Format("Jan 2, 2006 at 3:04pm")
// 		seasonStr := ""
// 		if info.Season != "" {
// 			seasonStr = " | S" + info.Season
// 		}
// 		items = append(
// 			items, ui.NewListItem(
// 				fmt.Sprintf("[%s] %s - %s", info.Fansub, info.Title, info.Episode),
// 				fmt.Sprintf("%s | %s | %s%s", dateStr, entry.Size, info.Encoding, seasonStr),
// 				count,
// 			),
// 		)
// 		count++
// 	}

// 	return ui.NewList(
// 		ui.ListOptions{
// 			Items:         items,
// 			ShortHelpKeys: []key.Binding{ui.KeyMap.Select, ui.KeyMap.Back},
// 			Width:         m.windowSize.Width - 3,
// 			MaxHeight:     int(float64(m.windowSize.Height) * 0.66),
// 			ItemsPerPage:  3,
// 		},
// 	), rssFansubs, nil
// }

// func (m RssModel) search(query string) tea.Cmd {
// 	return func() tea.Msg {
// 		var rss app.RSS
// 		result, err := rss.FindAnimeFansub(app.Nyaa, query)
// 		if err != nil {
// 			return DefaultErrorMsg{err}
// 		}

// 		return result
// 	}
// }

// func (m RssModel) createAnimeList() list.Model {
// 	anime := m.db.Anime()
// 	items := make([]list.Item, len(anime))
// 	for i, entry := range anime {
// 		items[i] = ui.NewListItem(entry.JPN_Title, entry.ENG_Title, i)
// 	}

// 	return ui.NewList(ui.ListOptions{
// 		Items:        items,
// 		Width:        m.windowSize.Width - 3,
// 		MaxHeight:    int(float64(m.windowSize.Height) * 0.66),
// 		ItemsPerPage: 3,
// 		EnableFilter: true,
// 	})
// }

// func (m RssModel) findResolution(encoding string) string {
// 	resolutions := [...]string{"480p", "720p", "1080p"}

// 	for _, res := range resolutions {
// 		if strings.Contains(encoding, res) {
// 			return res
// 		}
// 	}
// 	return ""
// }

// func (m RssModel) saveQbtFeed(feed string) tea.Cmd {
// 	return func() tea.Msg {
// 		port := strconv.Itoa(m.db.Profile().QbtPort)
// 		qb, err := qbittorrent.NewLogin(port)
// 		if err != nil {
// 			return QbtSavedMsg{err}
// 		}

// 		feedName := m.createFeedName(m.state.saveStatus.anime)
// 		err = qb.AddFeed(feedName, feed)
// 		if err != nil {
// 			return QbtSavedMsg{err}
// 		}

// 		err = qb.AddRuleFeed(kitsu.RssRuleName, feed)
// 		if err != nil {
// 			return QbtSavedMsg{err}
// 		}

// 		m.state.saveStatus.anime.QbtFeed = struct {
// 			Name    string
// 			RuleURI string
// 		}{
// 			Name:    feedName,
// 			RuleURI: feed,
// 		}

// 		err = m.db.UpdateAnime(m.state.saveStatus.anime)
// 		if err != nil {
// 			return QbtSavedMsg{err}
// 		}

// 		return QbtSavedMsg{nil}
// 	}
// }

// func (m RssModel) removeFeed() tea.Cmd {
// 	return func() tea.Msg {
// 		port := strconv.Itoa(m.db.Profile().QbtPort)
// 		qb, err := qbittorrent.NewLogin(port)
// 		if err != nil {
// 			return DefaultErrorMsg{err}
// 		}
// 		defer func() { _ = qb.Logout() }()

// 		anime := &m.state.saveStatus.anime

// 		err = qb.DeleteFeed(anime.QbtFeed.Name)
// 		if err != nil {
// 			return DefaultErrorMsg{err}
// 		}

// 		err = qb.DeleteRuleFeed(kitsu.RssRuleName, anime.QbtFeed.RuleURI)
// 		if err != nil {
// 			return DefaultErrorMsg{err}
// 		}

// 		anime.QbtFeed = struct {
// 			Name    string
// 			RuleURI string
// 		}{}

// 		err = m.db.UpdateAnime(*anime)
// 		if err != nil {
// 			return DefaultErrorMsg{err}
// 		}

// 		return QbtRemovedFeedMsg(true)
// 	}
// }

// func (m RssModel) createFeedName(anime kitsu.Anime) string {
// 	return fmt.Sprintf(
// 		"%s (%s)",
// 		anime.ENG_Title,
// 		anime.JPN_Title,
// 	)
// }

// func (m *RssModel) resetFlowState() {
// 	m.state.selAnimeIdx = -1
// 	m.state.isRssRefined = false
// 	m.state.saveStatus = struct {
// 		anime kitsu.Anime
// 		saved bool
// 		err   error
// 	}{}
// }

// func (m RssModel) testConn() tea.Cmd {
// 	return func() tea.Msg {
// 		port := strconv.Itoa(m.db.Profile().QbtPort)
// 		err := qbittorrent.CheckConn(port)
// 		if err != nil {
// 			return QbtConnMsg{false}
// 		}
// 		return QbtConnMsg{true}
// 	}
// }

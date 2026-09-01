package views

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/charmbracelet/x/ansi"
)

type (
	WatchPlayMsg        struct{}
	WatchLoadedAnimeMsg struct{ Value []app.FilteredAnime }
)

type WatchView int

const (
	WatchLoading = WatchView(iota)
	WatchSelection
	WatchProgress
)

type WatchModel struct {
	windowSize tea.WindowSizeMsg
	ui         struct {
		list    list.Model
		consent ui.ConsentModel
		loader  ui.LoaderModel
	}
	keys struct {
		reload key.Binding
		enter  key.Binding
	}
	db    *database.Database
	state WatchState
}

type WatchState struct {
	view          WatchView
	err           error
	filteredAnime []app.FilteredAnime
	lastSelected  struct {
		title string
		index int
	}
	selection struct {
		anime     app.FilteredAnime
		fileState app.WatchState
	}
	progress struct {
		isUpdated   bool
		isCompleted bool
		last        int
		next        int
	}
}

func newWatchModel(db *database.Database) WatchModel {
	m := WatchModel{}
	m.ui.loader = ui.NewLoader()
	m.keys.reload = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload"))
	m.ui.list = list.Model{}
	m.db = db
	m.ui.loader, _ = m.ui.loader.Start("Discovering Anime")
	return m
}

func (m WatchModel) Init() tea.Cmd {
	return tea.Sequence(m.ui.loader.Init(), m.loadAnime)
}

func (m WatchModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if len(m.ui.list.Items()) > 0 {
			m.ui.list.SetWidth(msg.Width - 5)
		}
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.state.err != nil {
				m.state = WatchState{}
				m.ui.loader, cmd = m.ui.loader.Start("Reloading Anime")
				return m, tea.Sequence(cmd, m.loadAnime)
			}

		// Prevent accidental submissions when in error state
		case key.Matches(msg, ui.KeyMap.Select, m.keys.reload):
			if m.state.err != nil {
				return m, nil
			}

		}

	case WatchLoadedAnimeMsg:
		m.state.view = WatchSelection
		m.state.filteredAnime = msg.Value
		m.ui.list = m.createAnimeList()
		m.ui.loader.Stop()

	case error:
		if m.ui.loader.IsLoading() {
			m.ui.loader.Stop()
		}
		m.state.err = msg
	}

	switch m.state.view {
	case WatchLoading:
		// We do nothing while loading
	case WatchSelection:
		m, cmd = m.UpdateSelection(msg)
		cmds = append(cmds, cmd)
	case WatchProgress:
		m, cmd = m.UpdateProgress(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m WatchModel) View() tea.View {
	state := m.state

	if state.err != nil {
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle("Watch", "Error"),
			"",
			ui.DisplayError(state.err),
		))
	}

	if m.ui.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.ui.loader.View()))
	}

	switch state.view {
	case WatchLoading:
		// We do not need a view for initial loading
	case WatchSelection:
		return m.ViewSelection()
	case WatchProgress:
		return m.ViewProgress()
	}

	return tea.NewView("missing WatchAnime view")
}

func (m WatchModel) ShortHelp() []key.Binding {
	if m.state.err != nil {
		return []key.Binding{ui.KeyMap.EscBack}
	}

	switch m.state.view {
	case WatchLoading:
		return nil
	case WatchSelection:
		if len(m.state.filteredAnime) == 0 {
			return []key.Binding{ui.KeyMap.EscBack}
		}
	case WatchProgress:
		if m.state.progress.isUpdated {
			return []key.Binding{ui.KeyMap.Select}
		}
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select, ui.KeyMap.EscBack}
	}
	return []key.Binding{}
}

func (m WatchModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m WatchModel) UpdateSelection(msg tea.Msg) (WatchModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.ui.list.FilterState() == list.Unfiltered {
				return m, exitToMenu
			}

		case key.Matches(msg, m.keys.reload):
			if m.ui.list.FilterState() > list.Unfiltered {
				break
			}
			m.state = WatchState{}
			m.ui.loader, cmd = m.ui.loader.Start("Reloading Anime")
			return m, tea.Sequence(cmd, m.loadAnime)

		case key.Matches(msg, ui.KeyMap.Submit):
			// do not allow submission of nothing
			if len(m.state.filteredAnime) == 0 {
				return m, nil
			}

			if m.ui.list.FilterState() != list.Filtering {
				//nolint:errcheck // will ALWAYS be a list
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.lastSelected.index = item.Index()
				m.state.lastSelected.title = item.Title()
				m.state.selection.anime = m.state.filteredAnime[item.Index()]
				anime := m.state.selection.anime

				fileEp, err := strconv.Atoi(anime.FileInfo.Episode)
				if err != nil {
					m.state.err = fmt.Errorf("failed to parse fansub episode: %w", err)
					return m, nil
				}

				nextProgress := anime.Value.Progress + 1
				switch {
				case fileEp == 0:
					m.state.selection.fileState = app.Pilot

				case anime.Value.Episodes > 0 && fileEp > anime.Value.Episodes:
					m.state.selection.fileState = app.NonSeasonal

				case fileEp > nextProgress:
					m.state.selection.fileState = app.Mismatched

				case fileEp < nextProgress:
					m.state.selection.fileState = app.Watched
				}

				return m, m.playAnime
			}

		}

	case WatchPlayMsg:
		m.state.view = WatchProgress
	}

	m.ui.list, cmd = m.ui.list.Update(msg)
	return m, cmd
}

func (m WatchModel) ViewSelection() tea.View {
	if len(m.state.filteredAnime) == 0 {
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("Watch"),
			"",
			ui.DisplayText([]string{";y;No Anime Fansubs Detected"}),
		))
	}
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Watch"),
		"",
		m.ui.list.View(),
	))
}

func (m WatchModel) UpdateProgress(msg tea.Msg) (WatchModel, tea.Cmd) {
	var cmd tea.Cmd
	m.ui.consent = m.ui.consent.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			m.cancelProgress()
			return m, nil

		case key.Matches(msg, ui.KeyMap.Select):
			// Continue & Reload watch list
			if m.state.progress.isUpdated {
				ls := m.state.lastSelected
				m.state = WatchState{}
				m.state.lastSelected = ls
				m.ui.loader, cmd = m.ui.loader.Start("Reloading Anime")
				return m, tea.Sequence(cmd, m.loadAnime)
			}

			if m.ui.consent.Select() == ui.No {
				m.cancelProgress()
				return m, nil
			}

			m.ui.loader, cmd = m.ui.loader.Start("Updating Anime")
			return m, tea.Batch(cmd, m.saveProgress)
		}

	case app.Progress:
		p := &m.state.progress
		p.isUpdated = true
		p.isCompleted = msg.IsCompleted
		p.last = msg.LastEp
		p.next = msg.NextEp
		m.ui.loader.Stop()

	}
	return m, nil
}

func (m WatchModel) ViewProgress() tea.View {
	if m.state.progress.isUpdated {
		return tea.NewView(m.displayUpdatedProgress())
	}
	return tea.NewView(m.displayProgress())
}

func (m WatchModel) displayUpdatedProgress() string {
	progressStr := ui.DisplayText([]string{
		fmt.Sprintf(
			"Episode ;w;updated ;x;from ;y;%d ;x;to ;g;%d",
			m.state.progress.last,
			m.state.progress.next,
		),
	})

	if m.state.progress.isCompleted {
		progressStr = ui.DisplayText([]string{`Anime has been ;g;Completed;x;!`})
	}

	if m.state.selection.fileState == app.Watched || m.state.selection.fileState == app.Pilot {
		progressStr = ui.DisplayText([]string{
			utils.ColorText(
				fmt.Sprintf(`Progress remains at ;g;%d;x;, but the file ;dc;has been;x;
moved to the watched directory.`, m.state.progress.last),
			),
		})
	}

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Watch", "Progress"),
		"",
		progressStr,
		"",
		ui.TextStyle.MarginTop(1).Foreground(ansi.BrightGreen).Render("> Continue"),
	)

	return view
}

func (m WatchModel) displayProgress() string {
	header := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Watch", "Watching"),
		"",
		ui.DisplayText(
			[]string{
				`The following anime should now be ;c;playing ;x;in your default video player:`,
			},
			0,
		),
	)

	anime := m.state.selection.anime.Value
	title := anime.ENG_Title
	if title == "" {
		title = anime.JPN_Title
	}

	engTitle := ui.Style.MarginLeft(5).Render(lipgloss.JoinHorizontal(
		lipgloss.Left,
		utils.ColorText(";db;  Title: "),
		ui.Style.Width(40).Render(title),
	))

	fileName := ui.Style.MarginLeft(5).Render(lipgloss.JoinHorizontal(
		lipgloss.Left,
		utils.ColorText(";db;   File: "),
		ui.Style.Width(m.windowSize.Width-17).Render(strings.Replace(
			m.state.selection.anime.FileInfo.Filename,
			m.state.selection.anime.FileInfo.Episode,
			utils.ColorText(fmt.Sprintf(";g;%s;x;", m.state.selection.anime.FileInfo.Episode)),
			1,
		)),
	))

	fileEp, err := strconv.Atoi(m.state.selection.anime.FileInfo.Episode)
	if err != nil {
		return ui.DisplayError(fmt.Errorf("failed to parse fansub episode: %w", err))
	}

	warnText := ""
	switch m.state.selection.fileState {
	case app.Pilot:
		warnText = `;m;Pilot episode detected; '00' episodes do not count towards
your kitsu library progress.`

	case app.NonSeasonal:
		warnText = `;m;This fansub group is not following seasonal episode counts, which is
why the file episode does not match the actual episode number.`

	case app.Mismatched:
		warnText = `;m;File episode count mismatch; you're either watching an episode
ahead of your progress, or the fansub group is not following seasonal episode counts.`

	case app.Watched:
		warnText = `;m;You have already seen this episode according to your progress.`
	}

	progress := m.state.selection.anime.Value.Progress
	switch {
	case m.state.selection.fileState == app.Pilot:
		progress = 0

	case fileEp >= progress+1:
		progress++

	case fileEp <= progress:
		progress = fileEp
	}

	progressStr := utils.ColorText(fmt.Sprintf(
		";db; Progress: ;y;%d ;bk;(current)", m.state.selection.anime.Value.Progress,
	))

	episodeLine := utils.ColorText(fmt.Sprintf("  ;db;Episode: ;g;%d ;bk;(watching)", progress))

	statusText := []string{progressStr, episodeLine}
	if warnText != "" {
		statusText = append([]string{warnText, ""}, statusText...)
	}

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		header, "",
		engTitle, "",
		fileName, "",
		ui.DisplayText(statusText), "",
		ui.TextStyle.Render(
			m.ui.consent.View(utils.ColorText(";b;Have you finished watching?")),
		),
	)

	return view
}

func (m WatchModel) createAnimeList() list.Model {
	listItems := make([]list.Item, len(m.state.filteredAnime))
	for i, item := range m.state.filteredAnime {
		title := item.Value.ENG_Title
		if title == "" {
			title = item.Value.JPN_Title
			// Kitsu has a bad habit of making alt english titles and
			// not assigning one as the actual eng title.
			if len(item.Value.AltTitles) > 0 {
				title = item.Value.AltTitles[0]
			}
		}
		listItems[i] = ui.NewListItem(title, item.Value.JPN_Title, i)
	}
	list := ui.NewList(ui.ListOptions{
		Items:         listItems,
		ShortHelpKeys: []key.Binding{m.keys.reload},
		Width:         m.windowSize.Width - 5,
		MaxHeight:     m.windowSize.Height,
		ItemsPerPage:  5,
		EnableFilter:  true,
	})
	if m.state.lastSelected.title != "" {
		items := list.VisibleItems()
		if len(items) > 0 {
			for _, item := range items {
				listItem, _ := item.(ui.ListItem)
				if listItem.Title() == m.state.lastSelected.title {
					list.Select(m.state.lastSelected.index)
					break
				}
			}
		}
	}
	return list
}

func (m *WatchModel) cancelProgress() {
	anime := m.state.filteredAnime
	ls := m.state.lastSelected
	m.state = WatchState{}
	m.state.view = WatchSelection
	m.state.lastSelected = ls
	m.state.filteredAnime = anime
}

func (m WatchModel) loadAnime() tea.Msg {
	/*
		INFO:
		Without this, the loader can look/feel like its glitchy. This timing
		makes the loading feel fast and meaningful.
	*/
	time.Sleep(180 * time.Millisecond)
	items, err := app.ListWorkingAnime(m.db, 10)
	if err != nil {
		return err
	}
	return WatchLoadedAnimeMsg{items}
}

func (m WatchModel) playAnime() tea.Msg {
	if err := app.PlayAnime(m.state.selection.anime.FileInfo.Filename); err != nil {
		return err
	}
	return WatchPlayMsg{}
}

func (m *WatchModel) saveProgress() tea.Msg {
	progress, err := app.SaveAnimeProgress(
		m.db,
		m.state.selection.anime,
		m.state.selection.fileState,
	)
	if err != nil {
		return err
	}
	return progress
}

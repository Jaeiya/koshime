package views

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/charmbracelet/x/ansi"
)

type WatchFileState int

const (
	_ = WatchFileState(iota)
	Pilot
	WatchedAlready
	NonSeasonalCount
	Mismatched
)

type (
	WatchPlayMsg struct{}
	// Holds previous episode progress value
	WatchUpdateSuccessMsg struct {
		lastEpisode int
		nextEpisode int
		isCompleted bool
	}
	WatchLoadedAnimeMsg struct{ Value []app.FilteredAnime }
)

type WatchView int

const (
	WatchSelection = WatchView(iota)
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
		fileState WatchFileState
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
	m.ui.list = ui.NewList(ui.ListOptions{})
	m.db = db
	m.ui.loader, _ = m.ui.loader.Start("Discovering Anime")
	return m
}

func (m WatchModel) Init() tea.Cmd {
	return tea.Sequence(m.ui.loader.Init(), m.LoadAnime)
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
				return m, func() tea.Msg { return "forceUpdateToReload" }
			}

		// Prevent accidental submissions when in error state
		case key.Matches(msg, ui.KeyMap.Select, m.keys.reload):
			if m.state.err != nil {
				return m, nil
			}

		}

	case WatchLoadedAnimeMsg:
		m.state.filteredAnime = msg.Value
		m.PopulateAnimeList()
		m.ui.loader.Stop()

	case error:
		if m.ui.loader.IsLoading() {
			m.ui.loader.Stop()
		}
		m.state.err = msg
	}

	switch m.state.view {
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
			return m, func() tea.Msg { return "forceUpdateToReload" }

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

				nextProgress := anime.Anime.Progress + 1
				switch {
				case fileEp == 0:
					m.state.selection.fileState = Pilot

				case anime.Anime.Episodes > 0 && fileEp > anime.Anime.Episodes:
					m.state.selection.fileState = NonSeasonalCount

				case fileEp > nextProgress:
					m.state.selection.fileState = Mismatched

				case fileEp < nextProgress:
					m.state.selection.fileState = WatchedAlready
				}

				return m, m.PlayAnime
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
				return m, func() tea.Msg { return "forceUpdateToReload" }
			}

			if m.ui.consent.Select() == ui.No {
				m.cancelProgress()
				return m, nil
			}

			m.ui.loader, cmd = m.ui.loader.Start("Updating Anime")
			return m, tea.Batch(cmd, m.SaveProgress)
		}

	case WatchUpdateSuccessMsg:
		p := &m.state.progress
		p.isUpdated = true
		p.isCompleted = msg.isCompleted
		p.last = msg.lastEpisode
		p.next = msg.nextEpisode
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

	if m.state.selection.fileState == WatchedAlready || m.state.selection.fileState == Pilot {
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

	anime := m.state.selection.anime.Anime
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
	case Pilot:
		warnText = `;m;Pilot episode detected; '00' episodes do not count towards
your kitsu library progress.`

	case NonSeasonalCount:
		warnText = `;m;This fansub group is not following seasonal episode counts, which is
why the file episode does not match the actual episode number.`

	case Mismatched:
		warnText = `;m;File episode count mismatch; you're either watching an episode
ahead of your progress, or the fansub group is not following seasonal episode counts.`

	case WatchedAlready:
		warnText = `;m;You have already seen this episode according to your progress.`
	}

	progress := m.state.selection.anime.Anime.Progress
	switch {
	case m.state.selection.fileState == Pilot:
		progress = 0

	case fileEp >= progress+1:
		progress++

	case fileEp <= progress:
		progress = fileEp
	}

	progressStr := utils.ColorText(fmt.Sprintf(
		";db; Progress: ;y;%d ;bk;(current)", m.state.selection.anime.Anime.Progress,
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

func (m *WatchModel) PopulateAnimeList() {
	listItems := make([]list.Item, len(m.state.filteredAnime))
	for i, item := range m.state.filteredAnime {
		title := item.Anime.ENG_Title
		if title == "" {
			title = item.Anime.JPN_Title
			// Kitsu has a bad habit of making alt english titles and
			// not assigning one as the actual eng title.
			if len(item.Anime.AltTitles) > 0 {
				title = item.Anime.AltTitles[0]
			}
		}
		listItems[i] = ui.NewListItem(title, item.Anime.JPN_Title, i)
	}
	m.ui.list = ui.NewList(ui.ListOptions{
		Items:         listItems,
		ShortHelpKeys: []key.Binding{m.keys.reload},
		Width:         m.windowSize.Width - 5,
		MaxHeight:     m.windowSize.Height,
		ItemsPerPage:  5,
		EnableFilter:  true,
	})
	if m.state.lastSelected.title != "" {
		items := m.ui.list.VisibleItems()
		if len(items) > 0 {
			for _, item := range items {
				listItem, _ := item.(ui.ListItem)
				if listItem.Title() == m.state.lastSelected.title {
					m.ui.list.Select(m.state.lastSelected.index)
					break
				}
			}
		}
	}
}

func (m *WatchModel) cancelProgress() {
	anime := m.state.filteredAnime
	ls := m.state.lastSelected
	m.state = WatchState{}
	m.state.lastSelected = ls
	m.state.filteredAnime = anime
}

func (m WatchModel) LoadAnime() tea.Msg {
	stream, err := fileSys.NewFilenameStream(fileSys.GetWorkingDir())
	if err != nil {
		return fmt.Errorf("failed creating filename stream: %w", err)
	}

	ff := app.FansubFilter{}
	items, err := ff.FilterByAnime(stream, m.db.Anime(), 25)
	if err != nil {
		return fmt.Errorf("failed to filter fansubs: %w", err)
	}

	slices.SortFunc(items, func(a, b app.FilteredAnime) int {
		return app.CompareAnime(a.Anime, b.Anime)
	})

	return WatchLoadedAnimeMsg{items}
}

func (m WatchModel) PlayAnime() tea.Msg {
	wd := fileSys.GetWorkingDir()
	var cmd *exec.Cmd
	filePath := filepath.Join(wd, m.state.selection.anime.FileInfo.Filename)

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", "", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	default:
		cmd = exec.Command("xdg-open", filePath)
	}
	err := cmd.Run()
	if err != nil {
		return err
	}
	return WatchPlayMsg{}
}

func (m *WatchModel) SaveProgress() tea.Msg {
	libEntry := m.state.selection.anime.Anime
	fileInfo := m.state.selection.anime.FileInfo

	if !fileSys.FileExists(filepath.Join(fileSys.GetWorkingDir(), fileInfo.Filename)) {
		return fmt.Errorf("the fansub file has been moved or deleted")
	}

	lastProgress := libEntry.Progress

	// 🟡 This only works for fansubs that follow seasonal episode counts.
	// A seasonal count means that each new season's first episode,
	// starts at 1.
	if m.state.selection.fileState == WatchedAlready || m.state.selection.fileState == Pilot {
		if err := m.moveFansubFile(); err != nil {
			return err
		}
		return WatchUpdateSuccessMsg{
			lastEpisode: lastProgress,
		}
	}

	/*
	 * INFO: We assume the user is downloading anime in the order they want to
	 * watch it, therefore no matter what the file episode says, we update
	 * to next episode number. This allows support for non-seasonal episode
	 * counts.
	 */
	nextProgress := lastProgress + 1

	progResp, err := kitsu.UpdateAnimeProgress(
		libEntry.LibID,
		m.db.Profile().AccessToken,
		nextProgress,
	)
	if err != nil {
		return fmt.Errorf("failed to update Kitsu progress: %w", err)
	}

	// 🟢 Kitsu does not always know the correct total episodes for a series
	// until the series is about to end.
	libEntry.Episodes = progResp.Included[0].Attributes.EpisodeCount

	// 🟢 When an anime is completed (progress updated to match total episodes), the
	// anime status is automatically updated by Kitsu, unless the episode count
	// is unknown (0).
	isCompleted := false
	if libEntry.Episodes == nextProgress {
		if err = app.CompleteAnime(m.db, libEntry.LibID); err != nil {
			return err
		}
		isCompleted = true
	} else {
		libEntry.Progress = nextProgress
		err = m.db.UpdateAnime(libEntry)
		if err != nil {
			return fmt.Errorf("failed to update database: %w", err)
		}
	}

	if err = m.moveFansubFile(); err != nil {
		return err
	}

	return WatchUpdateSuccessMsg{
		lastEpisode: lastProgress,
		nextEpisode: nextProgress,
		isCompleted: isCompleted,
	}
}

func (m WatchModel) moveFansubFile() error {
	if err := app.MoveFansub(m.state.selection.anime); err != nil {
		return fmt.Errorf("failed to move fansub file: %w", err)
	}
	return nil
}

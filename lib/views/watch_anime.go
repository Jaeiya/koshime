package views

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Jaeiya/koshime/lib"
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type WatchState int

const (
	Unwatched = WatchState(iota)
	Watched
	NonSeasonalCount
	Mismatched
)

type (
	WatchAnimePlayMsg struct{}
	// Holds previous episode progress value
	WatchAnimeUpdateSuccessMsg struct {
		lastEpisode    int
		nextEpisode    int
		fileEpisode    int
		alreadyWatched bool
		isCompleted    bool
	}
	WatchAnimeLoadedAnimeMsg = []lib.FilteredAnime
)

type WatchAnime_View int

const (
	WatchAnime_Selection = WatchAnime_View(iota)
	WatchAnime_Progress
)

type WatchAnime_Model struct {
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
	state WatchAnime_State
}

type WatchAnime_State struct {
	view          WatchAnime_View
	err           error
	filteredAnime []lib.FilteredAnime
	selection     struct {
		data      lib.FilteredAnime
		fileState WatchState
	}
	progress struct {
		isUpdated   bool
		isCompleted bool
		hasWatched  bool
		last        int
		next        int
		file        int
	}
}

func newWatchAnimeModel(db *database.Database) WatchAnime_Model {
	m := WatchAnime_Model{}
	m.ui.loader = ui.NewLoader()

	m.keys.reload = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload"))

	m.ui.list = ui.NewList(ui.ListOptions{})
	m.db = db
	return m
}

func (m WatchAnime_Model) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
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
				m.state = WatchAnime_State{}
				return m, func() tea.Msg { return "forceUpdateToReload" }
			}

		// Prevent accidental submissions when in error state
		case key.Matches(msg, ui.KeyMap.Select, m.keys.reload):
			if m.state.err != nil {
				return m, nil
			}

		}

	case WatchAnimeLoadedAnimeMsg:
		m.state.filteredAnime = msg
		m.PopulateAnimeList()
		m.ui.loader.Stop()

	case error:
		if m.ui.loader.IsLoading() {
			m.ui.loader.Stop()
		}
		m.state.err = msg
	}

	// This should only execute once
	if m.state.filteredAnime == nil && !m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Start("Discovering Anime")
		cmds = append(cmds, cmd, m.LoadAnime)
	}

	switch m.state.view {
	case WatchAnime_Selection:
		m, cmd = m.UpdateSelection(msg)
		cmds = append(cmds, cmd)
	case WatchAnime_Progress:
		m, cmd = m.UpdateProgress(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m WatchAnime_Model) View() (string, *tea.Cursor) {
	state := m.state

	if state.err != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle("Watch Anime", "Error"),
			"",
			ui.DisplayError(state.err),
		), nil
	}

	if m.ui.loader.IsLoading() {
		return ui.Style.MarginTop(1).Render(m.ui.loader.View()), nil
	}

	switch state.view {
	case WatchAnime_Selection:
		return m.ViewSelection()
	case WatchAnime_Progress:
		return m.ViewProgress()
	}

	return "watch::missing view", nil
}

func (m WatchAnime_Model) ShortHelp() []key.Binding {
	if m.state.err != nil {
		return []key.Binding{ui.KeyMap.EscBack}
	}

	switch m.state.view {
	case WatchAnime_Progress:
		if m.state.progress.isUpdated {
			return []key.Binding{ui.KeyMap.Select}
		}
		return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select, ui.KeyMap.EscBack}
	}
	return []key.Binding{}
}

func (m WatchAnime_Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m WatchAnime_Model) UpdateSelection(msg tea.Msg) (WatchAnime_Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			if m.ui.list.FilterState() == list.Unfiltered {
				return m, exitToMenu
			}

		case key.Matches(msg, m.keys.reload):
			if m.ui.list.FilterState() == list.Filtering {
				return m, nil
			}
			m.state = WatchAnime_State{}
			return m, func() tea.Msg { return "forceUpdateToReload" }

		case key.Matches(msg, ui.KeyMap.Submit):
			if m.ui.list.FilterState() != list.Filtering {
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.selection.data = m.state.filteredAnime[item.Index()]
				data := m.state.selection.data

				fileEp, err := strconv.Atoi(data.Fansub.Episode)
				if err != nil {
					m.state.err = fmt.Errorf("failed to parse fansub episode: %w", err)
					return m, nil
				}

				nextProgress := data.Anime.Progress + 1
				switch {
				case fileEp > data.Anime.Episodes:
					m.state.selection.fileState = NonSeasonalCount

				case fileEp > nextProgress:
					m.state.selection.fileState = Mismatched

				case fileEp < nextProgress:
					m.state.selection.fileState = Watched
				}

				return m, m.PlayAnime
			}

		}

	case WatchAnimePlayMsg:
		m.state.view = WatchAnime_Progress
	}

	m.ui.list, cmd = m.ui.list.Update(msg)
	return m, cmd
}

func (m WatchAnime_Model) ViewSelection() (string, *tea.Cursor) {
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Watch Anime"),
		"",
		m.ui.list.View(),
	)
	return view, nil
}

func (m WatchAnime_Model) UpdateProgress(msg tea.Msg) (WatchAnime_Model, tea.Cmd) {
	var cmd tea.Cmd
	m.ui.consent = m.ui.consent.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.EscBack):
			m.cancelProgress()
			return m, nil

		case key.Matches(msg, ui.KeyMap.Select):
			// Reload watch list
			if m.state.progress.isUpdated {
				m.state = WatchAnime_State{}
				return m, func() tea.Msg { return "forceUpdateToReload" }
			}

			if m.ui.consent.Select() == ui.No {
				m.cancelProgress()
				return m, nil
			}

			m.ui.loader, cmd = m.ui.loader.Start("Updating Anime")
			return m, tea.Batch(cmd, m.SaveProgress)
		}

	case WatchAnimeUpdateSuccessMsg:
		p := &m.state.progress
		p.isUpdated = true
		p.hasWatched = msg.alreadyWatched
		p.isCompleted = msg.isCompleted
		p.last = msg.lastEpisode
		p.next = msg.nextEpisode
		m.ui.loader.Stop()

	}
	return m, nil
}

func (m WatchAnime_Model) ViewProgress() (string, *tea.Cursor) {
	if m.state.progress.isUpdated {
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

		if m.state.progress.hasWatched {
			progressStr = ui.DisplayText([]string{
				`You have already watched this Anime at an earlier time, therefore
;m;progress has not been updated.`,
			})
		}
		view := lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle("Watch Anime", "Progress"),
			"",
			progressStr,
			"",
			ui.TextStyle.MarginTop(1).Foreground(ansi.BrightGreen).Render("> Continue"),
		)
		return view, nil
	}

	engTitle := lipgloss.JoinHorizontal(
		lipgloss.Left,
		utils.ColorText(";db;  Title: "),
		ui.Style.Width(40).Render(m.state.selection.data.Anime.ENG_Title),
	)

	fileStr := strings.Replace(
		m.state.selection.data.Fansub.Filename,
		m.state.selection.data.Fansub.Episode,
		fmt.Sprintf(";g;%s;x;", m.state.selection.data.Fansub.Episode),
		1,
	)

	fileName := lipgloss.JoinHorizontal(
		lipgloss.Left,
		utils.ColorText(";db;   File: "),
		ui.Style.Width(m.windowSize.Width-17).Render(utils.ColorText(fileStr)),
	)

	fileEp, err := strconv.Atoi(m.state.selection.data.Fansub.Episode)
	if err != nil {
		return ui.DisplayError(fmt.Errorf("failed to parse fansub episode: %w", err)), nil
	}

	warnText := ""
	switch m.state.selection.fileState {
	case NonSeasonalCount:
		warnText = `;dm;This fansub group is not following seasonal episode counts, which is
why the file episode does not match the actual episode number.`

	case Mismatched:
		warnText = `;dm;File episode count mismatch; you're either watching an episode
ahead of your progress, or the fansub group is not following seasonal episode counts.`

	case Watched:
		warnText = `;dm;You have already seen this episode according to your progress.`
	}

	progress := m.state.selection.data.Anime.Progress
	if fileEp >= progress+1 {
		progress = progress + 1
	} else if fileEp <= progress {
		progress = fileEp
	}

	progressStr := utils.ColorText(
		fmt.Sprintf(
			";db; Progress: ;y;%d ;bk;(current)",
			m.state.selection.data.Anime.Progress,
		),
	)

	episodeText := []string{
		utils.ColorText(
			fmt.Sprintf("  ;db;Episode: ;g;%d ;bk;(watching)", progress),
		),
	}
	if warnText != "" {
		episodeText = append([]string{warnText, "", progressStr}, episodeText...)
	} else {
		episodeText = append([]string{progressStr}, episodeText...)
	}
	episodeInfo := ui.DisplayText(episodeText)

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Watch Anime", "Watching"),
		"",
		ui.DisplayText(
			[]string{
				`The following anime should now be ;c;playing ;x;in your default video player:`,
			},
			0,
		),
		"",
		ui.Style.MarginLeft(5).Render(engTitle),
		"",
		ui.Style.MarginLeft(5).Render(fileName),
		"",
		episodeInfo,
		"",
		ui.TextStyle.Render(
			m.ui.consent.View(utils.ColorText(";b;Have you finished watching?")),
		),
	)

	return view, nil
}

func (m *WatchAnime_Model) PopulateAnimeList() {
	listItems := make([]list.Item, len(m.state.filteredAnime))
	for i, item := range m.state.filteredAnime {
		listItems[i] = ui.NewListItem(item.Anime.ENG_Title, item.Anime.JPN_Title, i)
	}
	m.ui.list = ui.NewList(ui.ListOptions{
		Items:         listItems,
		ShortHelpKeys: []key.Binding{m.keys.reload},
		Width:         m.windowSize.Width - 5,
		MaxHeight:     m.windowSize.Height,
		ItemsPerPage:  5,
	})
}

func (m *WatchAnime_Model) cancelProgress() {
	anime := m.state.filteredAnime
	m.state = WatchAnime_State{}
	m.state.filteredAnime = anime
}

func (m WatchAnime_Model) LoadAnime() tea.Msg {
	stream, err := utils.NewFilenameStream(utils.GetWorkingDir())
	if err != nil {
		return fmt.Errorf("failed creating filename stream: %w", err)
	}

	ff := lib.FansubFilter{}
	items, err := ff.FilterByLibEntry(stream, m.db.GetAllAnime())
	if err != nil {
		return fmt.Errorf("failed to filter fansubs: %w", err)
	}

	return WatchAnimeLoadedAnimeMsg(items)
}

func (m WatchAnime_Model) PlayAnime() tea.Msg {
	wd := utils.GetWorkingDir()
	var cmd *exec.Cmd
	filePath := filepath.Join(wd, m.state.selection.data.Fansub.Filename)

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
	return WatchAnimePlayMsg{}
}

func (m *WatchAnime_Model) SaveProgress() tea.Msg {
	anime := m.state.selection.data
	if !utils.FileExists(filepath.Join(utils.GetWorkingDir(), anime.Fansub.Filename)) {
		return fmt.Errorf("the fansub file has been moved or deleted")
	}

	lastProgress := anime.Anime.Progress

	fileEpisode, err := strconv.Atoi(anime.Fansub.Episode)
	if err != nil {
		return fmt.Errorf("failed to parse file episode number: %w", err)
	}

	// 🟡 This only works for fansubs that follow seasonal episode counts.
	// A seasonal count means that each new season's first episode,
	// starts at 1.
	if m.state.selection.fileState == Watched {
		if err = m.moveFansubFile(); err != nil {
			return err
		}
		return WatchAnimeUpdateSuccessMsg{
			lastEpisode:    lastProgress,
			alreadyWatched: true,
		}
	}

	// 🔵 We assume the user is downloading anime in the order they want to
	// watch it, therefore no matter what the file episode says, we update
	// to next episode number. This allows support for non-seasonal episode
	// counts.
	nextProgress := lastProgress + 1

	progResp, err := kitsu.UpdateProgress(
		anime.Anime.LibID,
		m.db.GetProfile().AccessToken,
		nextProgress,
	)
	if err != nil {
		return fmt.Errorf("failed to update Kitsu progress: %w", err)
	}

	// 🟢 At the beginning of an Anime season, Kitsu does not always know
	// the total episodes for each series.
	if anime.Anime.Episodes == 0 {
		anime.Anime.Episodes = progResp.Included[0].Attributes.EpisodeCount
	}

	// 🟢 When an anime is completed (progress updated to match total episodes), the
	// anime status is automatically updated by Kitsu, unless the episode count
	// is unknown (0).
	isCompleted := false
	if anime.Anime.Episodes == nextProgress {
		err := m.db.DeleteAnimeById(anime.Anime.LibID)
		if err != nil {
			return fmt.Errorf("failed to delete completed anime: %w", err)
		}
		isCompleted = true
	} else {
		anime.Anime.Progress = nextProgress
		err = m.db.UpdateAnime(anime.Anime)
		if err != nil {
			return fmt.Errorf("failed to update database: %w", err)
		}
	}

	if err = m.moveFansubFile(); err != nil {
		return err
	}

	return WatchAnimeUpdateSuccessMsg{
		lastEpisode: lastProgress,
		nextEpisode: nextProgress,
		fileEpisode: fileEpisode,
		isCompleted: isCompleted,
	}
}

func (m WatchAnime_Model) moveFansubFile() error {
	anime := m.state.selection.data
	wd := utils.GetWorkingDir()
	err := utils.MoveFile(
		anime.Fansub.Filename,
		filepath.Join(wd, "(watched)", anime.Fansub.Filename),
	)
	if err != nil {
		return fmt.Errorf("failed to move fansub file: %w", err)
	}
	return nil
}

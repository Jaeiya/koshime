package views

import (
	"fmt"

	"github.com/Jaeiya/koshime/lib"
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type (
	WatchAnimeCommandRunMsg  struct{}
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
	}
	db    *database.Database
	state WatchAnime_State
}

type WatchAnime_State struct {
	view          WatchAnime_View
	err           error
	filteredAnime []lib.FilteredAnime
	selection     struct {
		anime lib.FilteredAnime
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

	case WatchAnimeLoadedAnimeMsg:
		m.state.filteredAnime = msg
		m.PopulateAnimeList()
		m.ui.loader.Stop()

	case WatchAnimeCommandRunMsg:
		m.state.view = WatchAnime_Progress

	case error:
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
			m.state.filteredAnime = nil
			return m, func() tea.Msg { return "execUpdateFunc" }

		case key.Matches(msg, ui.KeyMap.Submit):
			if m.ui.list.FilterState() != list.Filtering {
				item := m.ui.list.SelectedItem().(ui.ListItem)
				m.state.selection.anime = m.state.filteredAnime[item.Index()]
				return m, m.ExecAnime
			}

		}
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
	m.ui.consent = m.ui.consent.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			if m.ui.consent.Select() == ui.No {
				m.reset()
				return m, nil
			}
		}
	}
	return m, nil
}

func (m WatchAnime_Model) ViewProgress() (string, *tea.Cursor) {
	engTitle := lipgloss.JoinHorizontal(
		lipgloss.Left,
		utils.ColorText(";dc;Title:  "),
		ui.Style.Width(40).Render(m.state.selection.anime.Anime.ENG_Title),
	)

	jpnTitle := lipgloss.JoinHorizontal(
		lipgloss.Left,
		utils.ColorText(";dc;Cannon: "),
		ui.Style.Width(40).Render(m.state.selection.anime.Anime.JPN_Title),
	)

	episodeStr := lipgloss.JoinHorizontal(
		lipgloss.Left,
		utils.ColorText(";dc;Episode: "),
		ui.Style.Foreground(ansi.BrightYellow).Render(m.state.selection.anime.Fansub.Episode),
	)
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle("Watch Anime", "In-Progress"),
		"",
		ui.TextStyle.Render(engTitle),
		"",
		ui.TextStyle.Render(jpnTitle),
		"",
		ui.TextStyle.Render(episodeStr),
		"",
		"",
		ui.TextStyle.Render(
			m.ui.consent.View(
				utils.ColorText(
					";dy;Only update the episode progress once you've finished the episode.",
				),
				"",
				utils.ColorText(";b;Update now?"),
			),
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

func (m *WatchAnime_Model) reset() {
	m.state.view = WatchAnime_Selection
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

func (m WatchAnime_Model) ExecAnime() tea.Msg {
	// wd := utils.GetWorkingDir()
	// // var cmd *exec.Cmd
	// filePath := filepath.Join(wd, m.state.selection.anime.Fansub.Filename)

	// switch runtime.GOOS {
	// case "windows":
	// 	cmd = exec.Command("cmd", "/C", "start", "", filePath)
	// case "darwin":
	// 	cmd = exec.Command("open", filePath)
	// default:
	// 	cmd = exec.Command("xdg-open", filePath)
	// }
	// err := cmd.Run()
	// if err != nil {
	// 	return err
	// }
	return WatchAnimeCommandRunMsg{}
}

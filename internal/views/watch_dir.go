package views

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
)

type WatchDirView int

const (
	WatchDirMenu = WatchDirView(iota)
	WatchDirCleanRecent
	WatchDirCleanAll
)

type (
	WatchDirSuccessfulDeleteMsg struct {
		count int
		size  int64
	}
	WatchDirLoadFilesMsg struct{}
)

type WatchDirInfo struct {
	location    string
	size        int64
	fileCount   int
	avgFileSize int64
}

type WatchDirModel struct {
	windowSize tea.WindowSizeMsg
	loader     ui.LoaderModel
	menu       ui.MenuModel
	state      WatchDirState
}

type WatchDirState struct {
	err          error
	view         WatchDirView
	folderInfo   WatchDirInfo
	cleanResults struct {
		deleted int
		size    int64
	}
}

func newWatchDirModel() WatchDirModel {
	m := WatchDirModel{}
	m.menu = ui.NewMenuModel([]string{
		"Clean Recent",
		"Clean All",
	})
	m.loader = ui.NewLoader()
	return m
}

func (m WatchDirModel) Init() tea.Cmd {
	return m.loadWatchDir
}

func (m WatchDirModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case WatchDirLoadFilesMsg:
		m.loader, cmd = m.loader.Start("Loading Watched Files")
		return m, tea.Batch(cmd, m.loadFiles)

	case WatchDirInfo:
		m.state.folderInfo = msg
		m.loader.Stop()

	case error:
		m.loader.Stop()
		m.state.err = msg
	}

	if m.loader.IsLoading() {
		m.loader, cmd = m.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.view {
	case WatchDirMenu:
		m, cmd = m.UpdateMenu(msg)
		cmds = append(cmds, cmd)

	case WatchDirCleanRecent, WatchDirCleanAll:
		m, cmd = m.UpdateCleaned(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m WatchDirModel) View() tea.View {
	if m.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.loader.View()))
	}

	if m.state.err != nil {
		return tea.NewView(ui.DisplayError(m.state.err))
	}

	switch m.state.view {
	case WatchDirMenu:
		return m.ViewMenu()
	case WatchDirCleanRecent, WatchDirCleanAll:
		return m.ViewCleaned()
	default:
		return tea.NewView("missing ManageWatchDir view")
	}
}

func (m WatchDirModel) ShortHelp() []key.Binding {
	if m.loader.IsLoading() {
		return []key.Binding{}
	}

	if m.state.folderInfo.size == 0 {
		return []key.Binding{ui.KeyMap.MainMenu}
	}

	switch m.state.view {
	case WatchDirCleanRecent:
		return []key.Binding{ui.KeyMap.Submit}
	default:
	}

	return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select, ui.KeyMap.MainMenu}
}

func (m WatchDirModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m WatchDirModel) UpdateMenu(msg tea.Msg) (WatchDirModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			return m, exitToMenu
		}

	case ui.MenuItemSelMsg:
		if m.state.folderInfo.fileCount == 0 {
			return m, nil
		}
		m.loader, cmd = m.loader.Start("Cleaning Files")
		switch msg.Value {
		case 0:
			m.state.view = WatchDirCleanRecent
			return m, tea.Batch(cmd, m.cleanRecentFiles)
		case 1:
			m.state.view = WatchDirCleanAll
			return m, tea.Batch(cmd, m.cleanAll)
		}
	}

	if m.state.folderInfo.fileCount > 0 {
		m.menu, cmd = m.menu.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m WatchDirModel) ViewMenu() tea.View {
	if m.state.folderInfo.size == 0 {
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplayTitle("Watch Directory"),
			"",
			ui.DisplayText([]string{
				`;m;Watch directory is currently empty.`,
			}),
		))
	}

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplayTitle("Watch Directory"),
		"",
		ui.DisplayPropValue(
			[]string{
				";dc;Location",
				";dc;Size",
				";dc;File Count",
				";dc;Avg. File Size",
			},
			[]string{
				fmt.Sprintf(";dg;%s", m.state.folderInfo.location),
				fmt.Sprintf(";dy;%s", utils.FormatBytes(m.state.folderInfo.size)),
				fmt.Sprintf(";dy;%s", strconv.Itoa(m.state.folderInfo.fileCount)),
				fmt.Sprintf(";dy;%s", utils.FormatBytes(m.state.folderInfo.avgFileSize)),
			},
		),
		"",
		ui.DisplayText([]string{
			`;dgu;Clean Recent;x; removes all files except for the most recent, per
series. If you've watched ;dy;5;x; different series, this will leave ;dy;5;x; files.`,
			`;dgu;Clean All;x; removes all files within the watch directory. This is
typically a good idea after each season.`,
		}, 1, 0, 1),
		m.menu.View(),
	))
}

func (m WatchDirModel) UpdateCleaned(msg tea.Msg) (WatchDirModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			m.state = WatchDirState{}
			return m, m.loadWatchDir
		}

	case WatchDirSuccessfulDeleteMsg:
		m.state.cleanResults.deleted = msg.count
		m.state.cleanResults.size = msg.size
		m.loader.Stop()
	}
	return m, nil
}

func (m WatchDirModel) ViewCleaned() tea.View {
	viewLines := []string{
		ui.DisplaySubTitle("Manage Watch Directory", "Cleaned Recent"),
		"",
	}

	continueStr := ui.NewMenuModel([]string{"Continue"}).View()

	if m.state.cleanResults.deleted == 0 {
		typeStr := "Folder has no recent files to delete"
		if m.state.view == WatchDirCleanAll {
			typeStr = "Folder is already empty"
		}
		viewLines = append(
			viewLines,
			ui.DisplayText([]string{
				typeStr,
			}),
			"",
			continueStr,
		)
		return tea.NewView(lipgloss.JoinVertical(
			lipgloss.Left,
			viewLines...,
		))
	}

	viewLines = append(
		viewLines,
		ui.DisplayPropValue(
			[]string{
				";dc;Deleted",
				";dc;Freed",
			},
			[]string{
				fmt.Sprintf(";dy;%d Files", m.state.cleanResults.deleted),
				fmt.Sprintf(";dy;%s", utils.FormatBytes(m.state.cleanResults.size)),
			},
		),
		"",
		continueStr,
	)

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		viewLines...,
	))
}

func (m WatchDirModel) loadFiles() tea.Msg {
	dirPath := app.WatchedDir()
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("could not read watch dir: %w", err)
	}

	info := WatchDirInfo{
		location: dirPath,
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileInfo, err := os.Stat(filepath.Join(dirPath, entry.Name()))
		if err != nil {
			return fmt.Errorf("could not read file stats: %w", err)
		}
		info.size += fileInfo.Size()
		info.fileCount++
	}
	if info.size > 0 {
		info.avgFileSize = info.size / int64(info.fileCount)
	}
	return info
}

func (m WatchDirModel) cleanRecentFiles() tea.Msg {
	fileNames, err := fileSys.ReadDirFiles(app.WatchedDir())
	if err != nil {
		return fmt.Errorf("failed to read watch dir: %w", err)
	}

	type FileInfo struct {
		modTime  int64
		fileName string
		size     int64
	}

	infoMap := map[string][]FileInfo{}
	for _, name := range fileNames {
		fp := app.FansubParser{}
		fansub, err := fp.Parse(name)
		if err != nil {
			return fmt.Errorf("failed to parse fansub file: %w", err)
		}
		stats, err := os.Stat(filepath.Join(app.WatchedDir(), name))
		if err != nil {
			return fmt.Errorf("failed to get file stats: %w", err)
		}
		infoMap[fansub.Title] = append(infoMap[fansub.Title], FileInfo{
			modTime:  stats.ModTime().UnixMilli(),
			fileName: name,
			size:     stats.Size(),
		})
	}

	sortMostRecent := func(data []FileInfo) []FileInfo {
		slices.SortFunc(data, func(a FileInfo, b FileInfo) int {
			// Descending order
			if a.modTime < b.modTime {
				return 1
			}
			return -1
		})
		return data
	}

	var totalSize int64
	var count int

	for _, info := range infoMap {
		if len(info) < 2 {
			continue
		}

		sortMostRecent(info)
		for _, info := range info[1:] {
			err := os.Remove(filepath.Join(app.WatchedDir(), info.fileName))
			if err != nil {
				return fmt.Errorf("failed to remove fansub: %w", err)
			}
			totalSize += info.size
			count++
		}
	}

	return WatchDirSuccessfulDeleteMsg{count, totalSize}
}

func (m WatchDirModel) cleanAll() tea.Msg {
	fileNames, err := fileSys.ReadDirFiles(app.WatchedDir())
	if err != nil {
		return fmt.Errorf("failed to read watch dir: %w", err)
	}

	var size int64
	var count int

	for _, fn := range fileNames {
		path := filepath.Join(app.WatchedDir(), fn)
		stats, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to get file stats: %w", err)
		}

		if err = os.Remove(path); err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
		size += stats.Size()
		count++
	}

	return WatchDirSuccessfulDeleteMsg{count, size}
}

func (m WatchDirModel) loadWatchDir() tea.Msg {
	return WatchDirLoadFilesMsg{}
}

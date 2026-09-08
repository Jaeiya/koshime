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

type ManDirView int

const (
	ManDirMenu = ManDirView(iota)
	ManDirCleanRecent
	ManDirCleanAll
)

type (
	ManDirSuccessfulDeleteMsg struct {
		count int
		size  int64
	}
	ManDirLoadFilesMsg struct{}
)

type ManDirInfo struct {
	location    string
	size        int64
	fileCount   int
	avgFileSize int64
}

type ManDirResults struct {
	Deleted int
	Size    int64
}

type ManDirModel struct {
	windowSize tea.WindowSizeMsg
	loader     ui.LoaderModel
	menu       ui.MenuModel
	err        error
	view       ManDirView
	folderInfo ManDirInfo
	results    ManDirResults
}

func newManDirModel() ManDirModel {
	m := ManDirModel{}
	m.menu = ui.NewMenuModel([]string{
		"Clean Recent",
		"Clean All",
	})
	m.loader = ui.NewLoader()
	return m
}

func (m ManDirModel) Init() tea.Cmd {
	return m.loadWatchDir
}

func (m ManDirModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case ManDirLoadFilesMsg:
		m.loader, cmd = m.loader.Start("Loading Watched Files")
		return m, tea.Batch(cmd, m.loadFiles)

	case ManDirInfo:
		m.folderInfo = msg
		m.loader.Stop()

	case error:
		m.loader.Stop()
		m.err = msg
	}

	if m.loader.IsLoading() {
		m.loader, cmd = m.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.view {
	case ManDirMenu:
		m, cmd = m.UpdateMenu(msg)
		cmds = append(cmds, cmd)

	case ManDirCleanRecent, ManDirCleanAll:
		m, cmd = m.UpdateCleaned(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m ManDirModel) View() tea.View {
	if m.loader.IsLoading() {
		return tea.NewView(ui.Style.MarginTop(1).Render(m.loader.View()))
	}

	if m.err != nil {
		return tea.NewView(ui.DisplayError(m.err))
	}

	switch m.view {
	case ManDirMenu:
		return m.ViewMenu()
	case ManDirCleanRecent, ManDirCleanAll:
		return m.ViewCleaned()
	default:
		return tea.NewView("missing ManageWatchDir view")
	}
}

func (m ManDirModel) ShortHelp() []key.Binding {
	if m.loader.IsLoading() {
		return []key.Binding{}
	}

	if m.folderInfo.size == 0 {
		return []key.Binding{ui.KeyMap.MainMenu}
	}

	switch m.view {
	case ManDirCleanRecent:
		return []key.Binding{ui.KeyMap.Submit}
	default:
	}

	return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.Select, ui.KeyMap.MainMenu}
}

func (m ManDirModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m ManDirModel) UpdateMenu(msg tea.Msg) (ManDirModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			return m, exitToMenu
		}

	case ui.MenuItemSelMsg:
		if m.folderInfo.fileCount == 0 {
			return m, nil
		}
		m.loader, cmd = m.loader.Start("Cleaning Files")
		switch msg.Value {
		case 0:
			m.view = ManDirCleanRecent
			return m, tea.Batch(cmd, m.cleanRecentFiles)
		case 1:
			m.view = ManDirCleanAll
			return m, tea.Batch(cmd, m.cleanAll)
		}
	}

	if m.folderInfo.fileCount > 0 {
		m.menu, cmd = m.menu.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m ManDirModel) ViewMenu() tea.View {
	if m.folderInfo.size == 0 {
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
				fmt.Sprintf(";dg;%s", m.folderInfo.location),
				fmt.Sprintf(";dy;%s", utils.FormatBytes(m.folderInfo.size)),
				fmt.Sprintf(";dy;%s", strconv.Itoa(m.folderInfo.fileCount)),
				fmt.Sprintf(";dy;%s", utils.FormatBytes(m.folderInfo.avgFileSize)),
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

func (m ManDirModel) UpdateCleaned(msg tea.Msg) (ManDirModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.Select):
			// m.state = WatchDirState{}
			return m, m.loadWatchDir
		}

	case ManDirSuccessfulDeleteMsg:
		m.results.Deleted = msg.count
		m.results.Size = msg.size
		m.loader.Stop()
	}
	return m, nil
}

func (m ManDirModel) ViewCleaned() tea.View {
	viewLines := []string{
		ui.DisplaySubTitle("Manage Watch Directory", "Cleaned Recent"),
		"",
	}

	continueStr := ui.NewMenuModel([]string{"Continue"}).View()

	if m.results.Deleted == 0 {
		typeStr := "Folder has no recent files to delete"
		if m.view == ManDirCleanAll {
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
				fmt.Sprintf(";dy;%d Files", m.results.Deleted),
				fmt.Sprintf(";dy;%s", utils.FormatBytes(m.results.Size)),
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

func (m *ManDirModel) reset() {
	m.err = nil
	m.view = ManDirMenu
	m.folderInfo = ManDirInfo{}
	m.results = ManDirResults{}
}

func (m ManDirModel) loadFiles() tea.Msg {
	dirPath := app.WatchedDir()
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("could not read watch dir: %w", err)
	}

	info := ManDirInfo{
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

func (m ManDirModel) cleanRecentFiles() tea.Msg {
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

	return ManDirSuccessfulDeleteMsg{count, totalSize}
}

func (m ManDirModel) cleanAll() tea.Msg {
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

	return ManDirSuccessfulDeleteMsg{count, size}
}

func (m ManDirModel) loadWatchDir() tea.Msg {
	return ManDirLoadFilesMsg{}
}

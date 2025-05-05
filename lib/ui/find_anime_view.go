package ui

import tea "github.com/charmbracelet/bubbletea/v2"

func (m UIModel) UpdateFindAnime(msg tea.Msg) (UIModel, tea.Cmd) {
	return m, nil
}

func (m UIModel) ViewFindAnime() (string, *tea.Cursor) {
	return "view for Find Anime", nil
}

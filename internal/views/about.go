package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
)

type AboutModel struct {
	keys struct {
		openHistory  key.Binding
		closeHistory key.Binding
	}
	showHistory bool
}

func newAboutModel() AboutModel {
	m := AboutModel{}
	m.keys.openHistory = key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "open history"))
	m.keys.closeHistory = key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "close history"))
	return m
}

func (m AboutModel) Init() tea.Cmd {
	return nil
}

func (m AboutModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			return m, exitToMenu

		case key.Matches(msg, m.keys.openHistory):
			m.showHistory = !m.showHistory
			return m, nil
		}
	}

	return m, nil
}

func (m AboutModel) View() tea.View {
	header := []string{
		ui.DisplayTitle("About"),
		"",
	}

	if m.showHistory {
		return m.displayHistory(header)
	}

	return m.displayAbout(header)
}

func (m AboutModel) ShortHelp() []key.Binding {
	if m.showHistory {
		return []key.Binding{m.keys.closeHistory, ui.KeyMap.MainMenu}
	}
	return []key.Binding{m.keys.openHistory, ui.KeyMap.MainMenu}
}

func (m AboutModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m AboutModel) displayHistory(header []string) tea.View {
	header = append(
		header,
		ui.DisplayText([]string{`Koshime was created by me (;y;Jaeiya;x;) because I ❤️ Anime`}),
		ui.DisplayText([]string{
			`I didn't enjoy navigating ;dm;Kitsu's;x; convoluted interface just
to update the progress of anime I was watching. `,
			`That friction turned into an obsession. I was driven to make the
process of ;w;watching;x; and ;w;updating;x; anime as simple as possible.`,
			`;b;Wakitsu;x; was born. It was my first attempt at solving this
problem and it did a great job, but it wasn't good enough.`,
			`Eventually I learned a new programming language and used it as an
excuse to completely re-write ;b;Wakitsu;x;. ;g;Koshime;x; is technically
;b;Wakitsu;x; ;w;v2.0;x;.`,
			`Though as ;b;Wakitsu;x; was the flawed first iteration of my vision,
the name of the program was also a rough draft: `,
			`   ;dc;wa ;bk;(watch);x; ;dc;kitsu ;bk;(https://kitsu.app);x;`,
			`;g;Koshime;x; combines ;w;Koshin;x; and ;w;Anime;x;, where Koshin
is the Japanese word for update or renewal:`,
			`   ;dc;Kosh ;bk;(Kosh-in) ;dc;ime ;bk;(an-ime);x;`,
			`This new name completely captures the intended purpose of the
application...and the rest is history.`,
		}, 1, 1),
	)

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		header...,
	))
}

func (m AboutModel) displayAbout(header []string) tea.View {
	version := utils.ColorText(fmt.Sprintf(";c;%s", app.Version))
	if app.Version == "" || strings.Contains(app.Version, "build-") {
		version = utils.ColorText(fmt.Sprintf(";bk;0x;m;%s", app.CommitHash))
	}

	if app.CommitHash == "" {
		version = utils.ColorText(";r;Dev Build")
	}

	releaseDate := ""
	if app.BuildDate != "" {
		buildTime, err := utils.ToISO8601(app.BuildDate)
		if err != nil {
			return tea.NewView(ui.DisplayError(err))
		}
		releaseDate = buildTime.Local().Format("Jan 2, 2006 at 3:04pm")
	}

	if releaseDate == "" {
		releaseDate = utils.ColorText(";r;Now")
	}

	header = append(
		header,
		ui.DisplayText([]string{`Koshime was created by ;y;Jaeiya;x; with ❤️ for Anime.`}),
		"",
		ui.TextStyle.Render(utils.ColorText(fmt.Sprintf(";dc; Version:;x; %s", version))),
		ui.TextStyle.Render(utils.ColorText(fmt.Sprintf(";dc;Released:;x; %s", releaseDate))),
		"",
		ui.DisplayText([]string{`;dc;Source Code`}),
		ui.TextStyle.Render(
			utils.ColorText(" ;c;• ;x;https://github.com/Jaeiya/koshime"),
		),
		ui.TextStyle.Render(
			utils.ColorText(" ;c;• ;x;https://github.com/Jaeiya/wakitsu ;bk;(v1 of Koshime)"),
		),
	)

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		header...,
	))
}

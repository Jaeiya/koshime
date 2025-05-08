package views

import (
	"strings"

	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type userSetupText struct {
	welcome  string
	username struct {
		enter          string
		confirmHeader  string
		confirmConsent string
		failed         string
	}
	password struct {
		enter  string
		failed string
	}
	libAnime struct {
		failed string
	}
}

var userSetupMsgs = func() userSetupText {
	v := userSetupText{}

	v.welcome = newText([]string{
		`;b;Welcome to ;g;Koshime;b;!`,
		`;b;Before you continue, make sure you already have a Kitsu
account. If not, sign-up here: ;y;https://kitsu.app`,
		`;w;Would you like to setup ;g;Koshime ;w;in this directory?`,
	}, 1)

	// Username Msgs
	v.username.enter = newText([]string{`Enter your ;g;Kitsu;x; user name.`}, 1)
	v.username.confirmHeader = newText(
		[]string{`;b;This is the first profile to pop up for that user name:;x;`},
		1,
	)
	v.username.confirmConsent = newText(
		[]string{`;b;Does that look like your profile?;x;`},
		1,
	)
	v.username.failed = newText([]string{`;y;User name not found; ;g;try again?;x;`}, 1)

	// Password Msgs
	v.password.enter = newText(
		[]string{`Enter your ;g;Kitsu;x; password. [;m;It will not be saved;x;]`},
		1,
	)
	v.password.failed = newText(
		[]string{
			`;r;Authorization Failed. ;b;You must have entered your password incorrectly.`,
			`;w;Would you like to ;g;try again?;x;`,
		},
		1,
	)

	// Library anime msgs
	v.libAnime.failed = newText(
		[]string{
			`;r;Failed to fetch library anime. ;b;This is probably a temporary failure.`,
			`;w;Would you like to ;g;try again?;x;`,
		},
		1,
	)

	return v
}()

type findAnimeText struct {
	title string
	kitsu string
	local string
}

var findAnimeMsgs = func() findAnimeText {
	txt := findAnimeText{}

	txt.title = newText(
		[]string{
			";g;... ;w;Find Anime ;g;...",
			`;x;You can search for either a ;b;full ;x;or ;b;partial ;x;anime title. If you're
searching ;dgu;Kitsu;x;, it will also search descriptions.`,
			`The ;dgu;Local;x; source searches your ;b;Koshime ;x;database, which stores all the
anime you're currently watching.`,
		},
		0, 1, 1,
	)

	activeStyle := ui.Style.Foreground(ansi.Green).Underline(true)
	txt.kitsu = activeStyle.Render("Kitsu") + "🌐"
	txt.local = activeStyle.Render("Local") + "📁"

	return txt
}()

// newText creates a single string with multiple lines
// separated by specific margins.
//
//	1st margin sets size of bottom margin
//	2nd margin sets size of top margin
//	3rd margin sets size of text-block bottom margin
func newText(lines []string, margins ...int) string {
	marginLen := len(margins)
	for i, para := range lines {
		s := ui.TextStyle
		if marginLen > 0 && margins[0] > 0 {
			s = s.Margin().MarginBottom(margins[0])
		}

		if marginLen > 1 && margins[1] > 0 {
			s = s.MarginTop(margins[1])
		}

		para = strings.ReplaceAll(para, "\n", " ")
		lines[i] = s.Render(utils.ColorText(para))
	}

	text := lipgloss.JoinVertical(lipgloss.Left, lines...)
	if marginLen > 2 && margins[2] > 0 {
		text = ui.Style.MarginBottom(margins[2]).Render(text)
	}
	return text
}

func newList(props []string, values []string) string {
	var sb strings.Builder

	propStyle := ui.Style.Width(9).Foreground(lipgloss.BrightWhite)
	valStyle := ui.Style.Width(40)

	for i, prop := range props {
		sb.WriteString(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				propStyle.Align(lipgloss.Right).Render(prop)+": ",
				valStyle.Render(utils.ColorText(values[i])),
			) + "\n",
		)
	}

	return ui.Style.MarginLeft(5).Render(sb.String())
}

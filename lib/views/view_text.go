package views

import (
	"fmt"
	"strings"

	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type viewHeader func(view string) string

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
	header     string
	viewHeader viewHeader
	title      string
	kitsu      string
	local      string
	notFound   func(query, source string) string
}

var findAnimeMsgs = func() findAnimeText {
	txt := findAnimeText{}

	txt.header = newHeader("Find Anime")

	txt.viewHeader = newViewHeader("Find Anime")

	txt.title = newText(
		[]string{
			`;x;You can search for a ;b;full title;x;, ;b;phrase;x;, or just a ;b;single
word;x;. You can even search for ;b;part ;x;of a word. Your query will be applied to all
available titles, as well as the synopsis.`,
			`The ;dgu;Kitsu;x; source searches ;b;all ;x;of Kitsu (not just your Kitsu
library) for any matches.`,
			`The ;dgu;Local;x; source searches your ;b;Koshime ;x;database for any matches.
It only contains anime that you're currently watching.`,
		},
		0, 1, 1,
	)

	activeStyle := ui.Style.Foreground(ansi.Green).Underline(true)
	txt.kitsu = activeStyle.Render("Kitsu") + "🌐"
	txt.local = activeStyle.Render("Local") + "📁"

	txt.notFound = func(query, source string) string {
		return newText([]string{
			fmt.Sprintf(";x;No ;dgu;%s;x; results found for: ;y;%s", source, query),
		}, 0, 1)
	}

	return txt
}()

func newHeader(s string) string {
	return newText([]string{
		fmt.Sprintf(";g;... ;b;%s ;g;...", s),
	}, 0, 1)
}

func newViewHeader(s string) viewHeader {
	return func(view string) string {
		return newText([]string{
			fmt.Sprintf(";g;... ;b;%s;g; ⟶ ;w;%s ;g;...", s, view),
		}, 0, 1)
	}
}

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

// newPropValDisplay displays properties and values in a fixed
// width arrangement.
func newPropValDisplay(props []string, values []string) string {
	if len(props) != len(values) {
		panic("number of properties do not match number of values")
	}
	var sb strings.Builder

	var propWidth int
	for _, p := range props {
		if propWidth < lipgloss.Width(p) {
			propWidth = lipgloss.Width(p)
		}
	}

	propStyle := ui.Style.Width(propWidth + 1).
		Align(lipgloss.Right).
		Foreground(lipgloss.BrightWhite)
	valStyle := ui.Style.Width(60)

	for i, prop := range props {
		sb.WriteString(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				propStyle.Render(prop+":")+" ",
				valStyle.Render(utils.ColorText(values[i])),
			) + "\n",
		)
	}

	return ui.Style.MarginLeft(5).Render(strings.TrimRight(sb.String(), "\n"))
}

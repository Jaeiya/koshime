package ui

import (
	"strings"

	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/lipgloss/v2"
)

var userWelcomeTxt = newText([]string{
	`;b;Welcome to ;g;Koshime;b;!`,
	`;b;Before you continue, make sure you already have a Kitsu
account. If not, sign-up here: ;y;https://kitsu.app`,
	`;w;Would you like to setup ;g;Koshime ;w;in this directory?`,
}, true)

var userNameTxt = newText([]string{`Enter your ;g;Kitsu;x; user name.`}, true)

var confirmUsernamePreTxt = newText(
	[]string{`;b;This is the first profile to pop up for that user name:;x;`},
	true,
)

var confirmUsernameConsentTxt = newText(
	[]string{`;b;Does that look like your profile?;x;`},
	true,
)

var usernameFailedTxt = newText([]string{`;y;User name not found; ;g;try again?;x;`}, true)

var passwordTxt = newText(
	[]string{`Enter your ;g;Kitsu;x; password. [;m;It will not be saved;x;]`},
	true,
)

var passwordFailedTxt = newText([]string{
	`;r;Authorization Failed. ;b;You must have entered your password incorrectly.`,
	`;w;Would you like to ;g;try again?;x;`,
}, true)

var libAnimeFetchFailedTxt = newText([]string{
	`;r;Failed to fetch library anime. ;b;This is probably a temporary failure.`,
	`;w;Would you like to ;g;try again?;x;`,
}, true)

func newText(text []string, includeMargin bool) string {
	for i, para := range text {
		margin := 1
		if i == 0 || includeMargin == false {
			margin = 0
		}
		para = strings.ReplaceAll(para, "\n", " ")
		text[i] = textStyle.MarginTop(margin).Render(utils.ColorText(para))
	}

	return lipgloss.JoinVertical(lipgloss.Left, text...)
}

func createList(props []string, values []string) string {
	var sb strings.Builder

	propStyle := style.Width(9).Foreground(lipgloss.BrightWhite)
	valStyle := style.Width(40)

	for i, prop := range props {
		sb.WriteString(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				propStyle.Align(lipgloss.Right).Render(prop)+": ",
				valStyle.Render(utils.ColorText(values[i])),
			) + "\n",
		)
	}

	return style.MarginTop(1).MarginLeft(5).Render(sb.String())
}

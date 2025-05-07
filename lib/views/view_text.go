package views

import (
	"strings"

	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/lipgloss/v2"
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
	}, true)

	// Username Msgs
	v.username.enter = newText([]string{`Enter your ;g;Kitsu;x; user name.`}, true)
	v.username.confirmHeader = newText(
		[]string{`;b;This is the first profile to pop up for that user name:;x;`},
		true,
	)
	v.username.confirmConsent = newText(
		[]string{`;b;Does that look like your profile?;x;`},
		true,
	)
	v.username.failed = newText([]string{`;y;User name not found; ;g;try again?;x;`}, true)

	// Password Msgs
	v.password.enter = newText(
		[]string{`Enter your ;g;Kitsu;x; password. [;m;It will not be saved;x;]`},
		true,
	)
	v.password.failed = newText(
		[]string{
			`;r;Authorization Failed. ;b;You must have entered your password incorrectly.`,
			`;w;Would you like to ;g;try again?;x;`,
		},
		true,
	)

	// Library anime msgs
	v.libAnime.failed = newText(
		[]string{
			`;r;Failed to fetch library anime. ;b;This is probably a temporary failure.`,
			`;w;Would you like to ;g;try again?;x;`,
		},
		true,
	)

	return v
}()

func newText(text []string, includeMargin bool) string {
	for i, para := range text {
		margin := 1
		if i == 0 || includeMargin == false {
			margin = 0
		}
		para = strings.ReplaceAll(para, "\n", " ")
		text[i] = ui.TextStyle.MarginTop(margin).Render(utils.ColorText(para))
	}

	return lipgloss.JoinVertical(lipgloss.Left, text...)
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

	return ui.Style.MarginTop(1).MarginLeft(5).Render(sb.String())
}

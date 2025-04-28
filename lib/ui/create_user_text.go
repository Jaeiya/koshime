package ui

import (
	"strings"

	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/lipgloss/v2"
)

var defaultTextStyle = lipgloss.NewStyle().PaddingLeft(3)

var userWelcomeTxt = newText(defaultTextStyle.MarginTop(1).MarginBottom(1), `
;b;Welcome to ;g;Koshime;b;!

;b;Before you continue, make sure you already have
a Kitsu account. If you don't have one, go to
this link: ;y;https://kitsu.app;x;`,
)

var userConsentTxt = newText(defaultTextStyle, `
;w;Would you like to setup ;g;Koshime;x; ;w;in this directory?;x;`,
)

var userNameTxt = newText(defaultTextStyle.MarginTop(1).PaddingBottom(1), `
Enter your ;g;Kitsu;x; user name.`,
)

var confirmUsernamePreTxt = newText(defaultTextStyle.MarginTop(1), `
;b;This is the first profile to pop up for that user name:;x;`,
)

var confirmUsernameConsentTxt = newText(defaultTextStyle.MarginTop(1), `
;b;Does that look like your profile?;x;`,
)

var usernameFailedTxt = newText(defaultTextStyle.MarginTop(1), `
;y;User name not found; ;g;try again?;x;`,
)

var passwordTxt = newText(defaultTextStyle.MarginTop(1).PaddingBottom(1), `
Enter your ;g;Kitsu;x; password. [;m;It will not be saved;x;]`,
)

var passwordFailedTxt = newText(defaultTextStyle.MarginTop(1), `
;r;Authorization Failed. ;b;You must have entered your password
incorrectly.

;w;Would you like to ;g;try again?;x;`)

var libAnimeFetchFailedTxt = newText(defaultTextStyle.MarginTop(1), `
;r;Failed to fetch library anime. ;b;This is probably a
temporary failure.

;w;Would you like to ;g;try again?;x;`)

func newText(style lipgloss.Style, text string) string {
	return style.Render(utils.ColorText(strings.TrimSpace(text)))
}

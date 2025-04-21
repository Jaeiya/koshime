package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/jaeiya/koshime/lib/utils"
)

var defaultTextStyle = lipgloss.NewStyle().PaddingLeft(3)

var userWelcomeTxt = newText(defaultTextStyle.MarginTop(1).MarginBottom(1), `
;g;Welcome to Koshime!

;b;I will need to grab your profile from Kitsu, which requires a
user name and password. Your password will ;y;not;x; ;b;be saved
and will only ;y;ever;x; ;b;be used to get an access token.;x;`,
)

var userConsentTxt = newText(defaultTextStyle, `
;w;Would you like to setup ;m;Koshime;x; ;w;in this directory?;x;`,
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

func newText(style lipgloss.Style, text string) string {
	return style.Render(utils.ColorText(strings.TrimSpace(text)))
}

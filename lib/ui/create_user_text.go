package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/jaeiya/koshime/lib/utils"
)

var defaultTextStyle = lipgloss.NewStyle().PaddingLeft(3)

var UserWelcomeMsg = defaultTextStyle.
	MarginTop(1).
	MarginBottom(1).
	Render(strings.TrimSpace(utils.ColorText(`
;g;Welcome to Koshime!

;b;I will need to grab your profile from Kitsu, which requires a
user name and password. Your password will ;y;not;x; ;b;be saved
and will only ;y;ever;x; ;b;be used to get an access token.;x;

`)))

var UserConsentMsg = defaultTextStyle.Render(strings.TrimSpace(utils.ColorText(`
;w;Would you like to setup ;m;Koshime;x; ;w;in this directory?;x;
`)))

var UserNameMsg = defaultTextStyle.MarginTop(1).Render(strings.TrimSpace(
	utils.ColorText(`
Please enter your ;g;Kitsu;x; user name.
`),
))

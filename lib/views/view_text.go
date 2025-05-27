package views

import (
	"github.com/Jaeiya/koshime/lib/ui"
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

	v.welcome = ui.DisplayText([]string{
		`;b;Welcome to ;g;Koshime;b;!`,
		`;b;Before you continue, make sure you already have a Kitsu
account. If not, sign-up here: ;y;https://kitsu.app`,
		`;w;Would you like to setup ;g;Koshime ;w;in this directory?`,
	}, 1)

	// Username Msgs
	v.username.enter = ui.DisplayText([]string{`Enter your ;g;Kitsu;x; user name.`}, 1)
	v.username.confirmHeader = ui.DisplayText(
		[]string{`;b;This is the first profile to pop up for that user name:;x;`},
		1,
	)
	v.username.confirmConsent = ui.DisplayText(
		[]string{`;b;Does that look like your profile?;x;`},
		1,
	)
	v.username.failed = ui.DisplayText([]string{`;y;User name not found; ;g;try again?;x;`}, 1)

	// Password Msgs
	v.password.enter = ui.DisplayText(
		[]string{`Enter your ;g;Kitsu;x; password. [;m;It will not be saved;x;]`},
		1,
	)
	v.password.failed = ui.DisplayText(
		[]string{
			`;r;Authorization Failed. ;b;You must have entered your password incorrectly.`,
			`;w;Would you like to ;g;try again?;x;`,
		},
		1,
	)

	// Library anime msgs
	v.libAnime.failed = ui.DisplayText(
		[]string{
			`;r;Failed to fetch library anime. ;b;This is probably a temporary failure.`,
			`;w;Would you like to ;g;try again?;x;`,
		},
		1,
	)

	return v
}()

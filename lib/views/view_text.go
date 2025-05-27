package views

import (
	"fmt"

	"github.com/Jaeiya/koshime/lib/ui"
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

type findAnimeText struct {
	title    string
	kitsu    string
	local    string
	notFound func(query, source string) string
}

var findAnimeMsgs = func() findAnimeText {
	txt := findAnimeText{}

	txt.title = ui.DisplayText(
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
		return ui.DisplayText([]string{
			fmt.Sprintf(";x;No ;dgu;%s;x; results found for: ;y;%s", source, query),
		}, 0, 1)
	}

	return txt
}()

type addAnimeText struct {
	queryDesc string
}

var addAnimeMsgs = func() addAnimeText {
	txt := addAnimeText{}

	txt.queryDesc = ui.DisplayText([]string{
		`Lookup an anime by any ;b;word ;x;or ;b;phrase;x;. Try to use
words that might be in the ;dc;title ;x;or ;dc;description;x;, for
better results.`,
	}, 1)

	return txt
}()

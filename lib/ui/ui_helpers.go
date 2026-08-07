package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/x/ansi"
)

type AnimeInfo struct {
	ID        string
	LibID     string
	JpnTitle  string
	EngTitle  string
	AltTitles []string
	ShowType  string
	Synopsis  string
	Status    string
	Progress  int
	AvgRating string
	Episodes  int
	Slug      string
	QbtFeed   struct {
		Name    string
		RuleURI string
	}
}

// DisplayCharLimit returns a string that indicates how many
// more characters are required to match the minimum.
func DisplayCharLimit(min int, text string) string {
	actualLen := utils.RuneCount(text)
	var charLimit string

	switch {
	case actualLen < min && actualLen > 0:
		charLimit = utils.ColorText(
			fmt.Sprintf(";r;%d;x;/;g;%d", actualLen, min),
		)

	case actualLen >= min:
		charLimit = utils.ColorText(";g;✓")
	}

	return charLimit
}

// DisplayPropVal converts a slice of properties and values
// into a displayable string.
//
// 🔴 Panics with inconsistent value/property lengths
func DisplayPropValue(props []string, values []string) string {
	if len(props) != len(values) {
		panic("number of properties do not match number of values")
	}
	var sb strings.Builder

	var propWidth int
	for i, p := range props {
		props[i] = utils.ColorText(p)
		if propWidth < lipgloss.Width(props[i]) {
			propWidth = lipgloss.Width(props[i])
		}
	}

	propStyle := Style.Width(propWidth).
		Align(lipgloss.Right).
		Foreground(lipgloss.BrightWhite)
	valStyle := Style.Width(60)

	for i, prop := range props {
		sb.WriteString(propStyle.Render(prop))
		sb.WriteString(": ")
		sb.WriteString(valStyle.Render(utils.ColorText(values[i])))
		sb.WriteRune('\n')
	}

	return Style.MarginLeft(5).Render(strings.TrimRight(sb.String(), "\n"))
}

func DisplayError(err error) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		TextStyle.MarginTop(1).Foreground(ansi.BrightRed).Render("Error"),
		TextStyle.PaddingLeft(2).Foreground(ansi.BrightYellow).Render(err.Error()),
	)
}

func DisplayTitle(s string) string {
	return DisplayText([]string{
		fmt.Sprintf(";g;... ;b;%s ;g;...", s),
	}, 0, 1)
}

func DisplaySubTitle(title string, subtitle string) string {
	return DisplayText([]string{
		fmt.Sprintf(";g;... ;b;%s;g; ⟶ ;w;%s ;g;...", title, subtitle),
	}, 0, 1)
}

// DisplayText creates a string from multiple lines
// separated by specified margins.
//
//	1st margin sets size of bottom margin
//	2nd margin sets size of top margin
//	3rd margin sets size of text-block bottom margin
func DisplayText(lines []string, margins ...int) string {
	marginLen := len(margins)
	for i, para := range lines {
		s := TextStyle
		if marginLen > 0 && margins[0] > 0 && i+1 != len(lines) {
			s = s.MarginBottom(margins[0])
		}

		if marginLen > 1 && margins[1] > 0 && i == 0 {
			s = s.MarginTop(margins[1])
		}

		para = strings.ReplaceAll(para, "\n", " ")
		lines[i] = s.Render(utils.ColorText(para))
	}

	text := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// If bottom margin is set to 0
	if marginLen > 2 && margins[2] == 0 {
		text = strings.TrimRight(text, "\n ")
	}

	// Set bottom margin > 0
	if marginLen > 2 && margins[2] > 0 {
		text = Style.MarginBottom(margins[2]).Render(text)
	}
	return text
}

func ToAnimeInfo(entries []kitsu.LibraryEntry) []AnimeInfo {
	infoEntries := make([]AnimeInfo, len(entries))
	for i, entry := range entries {
		infoEntries[i] = AnimeInfo{
			ID:        entry.ID,
			LibID:     entry.LibID,
			JpnTitle:  entry.JPN_Title,
			EngTitle:  entry.ENG_Title,
			AltTitles: entry.AltTitles,
			ShowType:  entry.Type,
			Synopsis:  entry.Synopsis,
			Status:    entry.Status,
			Progress:  entry.Progress,
			AvgRating: utils.CalcRating(entry.AvgRating),
			Episodes:  entry.Episodes,
			Slug:      entry.Slug,
			QbtFeed:   entry.QbtFeed,
		}
	}
	return infoEntries
}

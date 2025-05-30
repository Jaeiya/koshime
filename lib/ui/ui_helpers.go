package ui

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/charmbracelet/lipgloss/v2"
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
	Episodes  int
	Slug      string
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

func DisplayAnimeInfo(info AnimeInfo, showSynopsis bool) string {
	headers := []string{
		utils.ColorText(";g;Title"),
		utils.ColorText(";dc;English"),
	}
	items := []string{
		info.JpnTitle,
		info.EngTitle,
	}

	for range len(info.AltTitles) {
		headers = append(headers, utils.ColorText(";db;AltTitle"))
	}
	items = append(items, info.AltTitles...)

	headers = append(headers, utils.ColorText(";dc;Status"))
	items = append(items, utils.ColorText(";b;"+info.Status))

	headers = append(headers, utils.ColorText(";y;Type"))
	items = append(items, utils.ColorText(";c;"+info.ShowType))

	totalEpsStr := "Unknown"
	if info.Episodes > 0 {
		totalEpsStr = strconv.Itoa(info.Episodes)
	}

	if info.Progress > -1 {
		headers = append(headers, utils.ColorText(";y;Progress"))
		items = append(items, utils.ColorText(
			fmt.Sprintf(";dg;%d ;y;/ ;m;%s", info.Progress, totalEpsStr),
		))
	} else {
		headers = append(headers, utils.ColorText(";dc;Episodes"))
		items = append(items, utils.ColorText(fmt.Sprintf(";m;%s", totalEpsStr)))
	}

	link, _ := url.JoinPath(kitsu.KitsuDomain, info.Slug)
	if showSynopsis {
		headers = append(headers, utils.ColorText(";dc;Synopsis"), utils.ColorText(";x;Link"))
		items = append(items, info.Synopsis, utils.ColorText(";bk;"+link))
	} else {
		headers = append(headers, utils.ColorText(";x;Link"))
		items = append(items, utils.ColorText(";bk;"+link))
	}

	return DisplayPropValue(headers, items)
}

// DisplayPropValue displays properties and values in a fixed
// width arrangement.
func DisplayPropValue(props []string, values []string) string {
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

	propStyle := Style.Width(propWidth + 1).
		Align(lipgloss.Right).
		Foreground(lipgloss.BrightWhite)
	valStyle := Style.Width(60)

	for i, prop := range props {
		sb.WriteString(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				propStyle.Render(prop+":")+" ",
				valStyle.Render(utils.ColorText(values[i])),
			) + "\n",
		)
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
		if marginLen > 0 && margins[0] > 0 {
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

// DisplayPropVal converts a slice of properties and values
// into a displayable string.
//
// 🔴 Panics with inconsistent value/property lengths
func DisplayPropVal(props []string, values []string) string {
	if len(props) != len(values) {
		panic("number of properties do not match number of values")
	}

	var propWidth int
	for _, p := range props {
		if propWidth < lipgloss.Width(p) {
			propWidth = lipgloss.Width(p)
		}
	}

	propStyle := Style.Width(propWidth + 1).
		Align(lipgloss.Right).
		Foreground(lipgloss.Cyan)

	valStyle := Style.Width(60)

	var sb strings.Builder
	for i, prop := range props {
		sb.WriteString(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				propStyle.Render(prop+":")+" ",
				valStyle.Render(utils.ColorText(values[i])),
			) + "\n",
		)
	}

	return Style.MarginLeft(5).Render(strings.TrimRight(sb.String(), "\n"))
}

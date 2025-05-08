package utils

import (
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var style = lipgloss.NewStyle()

var foregroundStyleMap = map[string]lipgloss.Style{
	";bk;":  style.Foreground(ansi.BrightBlack),
	";r;":   style.Foreground(ansi.BrightRed),
	";g;":   style.Foreground(ansi.BrightGreen),
	";gu;":  style.Underline(true).Foreground(ansi.BrightGreen),
	";dg;":  style.Foreground(ansi.Green),
	";dgu;": style.Underline(true).Foreground(ansi.Green),
	";y;":   style.Foreground(ansi.BrightYellow),
	";b;":   style.Foreground(ansi.BrightBlue),
	";db;":  style.Foreground(ansi.Blue),
	";m;":   style.Foreground(ansi.BrightMagenta),
	";dm;":  style.Foreground(ansi.Magenta),
	";c;":   style.Foreground(ansi.BrightCyan),
	";w;":   style.Foreground(ansi.BrightWhite),
	";x;":   style.Foreground(ansi.White), // Reset to default color
}

// ColorText replaces special strings (e.g., ";r;" for Bright Red) with
// their corresponding lipgloss foreground color and returns the
// modified string.
func ColorText(text string) string {
	return renderStyledTokens(text, foregroundStyleMap)
}

func renderStyledTokens(input string, tokenMap map[string]lipgloss.Style) string {
	var sb strings.Builder
	currentStyle := lipgloss.NewStyle().Foreground(ansi.White)
	start := 0

StringLoop:
	for i := 0; i < len(input); {
		// Try to match a token at the current position
		for t, style := range tokenMap {
			if strings.HasPrefix(input[i:], t) {
				sb.WriteString(currentStyle.Render(input[start:i]))
				currentStyle = style
				i += len(t)
				start = i
				continue StringLoop
			}
		}
		i++
	}

	sb.WriteString(currentStyle.Render(input[start:]))
	return sb.String()
}

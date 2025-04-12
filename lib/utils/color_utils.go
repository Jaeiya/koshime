package utils

import (
	"image/color"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

func ColorText(text string, defaultColor color.Color, wordMap map[string]color.Color) string {
	textSB := strings.Builder{}
	leftoverSB := strings.Builder{}
	s := lipgloss.NewStyle()

TextLoop:
	for i := 0; i < len(text); {
		for t, v := range wordMap {
			if strings.HasPrefix(text[i:], t) {
				if leftoverSB.Len() > 0 {
					textSB.WriteString(s.Foreground(defaultColor).Render(leftoverSB.String()))
					leftoverSB.Reset()
				}
				textSB.WriteString(s.Foreground(v).Render(t))
				i += len(t)
				continue TextLoop
			}
		}

		// Lipgloss acts weird when new lines and trailing spaces are
		// styled manually
		if text[i] == '\n' {
			textSB.WriteString(
				s.Foreground(defaultColor).Render(leftoverSB.String()) + "\n",
			)
			leftoverSB.Reset()
		} else {
			leftoverSB.WriteByte(text[i])
		}
		i++
	}

	return textSB.String()
}

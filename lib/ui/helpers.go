package ui

import (
	"fmt"
	"unicode/utf8"

	"github.com/Jaeiya/koshime/lib/utils"
)

// DisplayCharLimit returns a string that indicates how many
// more characters are required to match the minimum.
func DisplayCharLimit(min int, text string) string {
	actualLen := utf8.RuneCountInString(text)
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

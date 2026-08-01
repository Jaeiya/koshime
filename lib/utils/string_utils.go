package utils

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

var RuneCount = utf8.RuneCountInString

/*
ReplaceAll uses the token map to replace all keys with the
mapped value, within the given string.
*/
func ReplaceAll(s string, tokenMap map[string]string) string {
	sb := strings.Builder{}
StringLoop:
	for i := 0; i < len(s); {
		for t, v := range tokenMap {
			if strings.HasPrefix(s[i:], t) {
				sb.WriteString(v)
				i += len(t)
				continue StringLoop
			}
		}
		sb.WriteByte(s[i])
		i++
	}

	return sb.String()
}

func RemoveBrackets(s string) string {
	if len(s) == 0 {
		return s
	}

	switch s[0] {
	case '[', '(':
		s = s[1:]
	}

	if s == "" {
		return s
	}

	switch s[len(s)-1] {
	case ']', ')':
		s = s[:len(s)-1]
	}

	return s
}

func ReplaceCutset(s, cutset, replacement string) string {
	var sb strings.Builder
	for _, r := range s {
		if strings.ContainsRune(cutset, r) {
			sb.WriteString(replacement)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func OrdinalString(s string) (string, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return "", fmt.Errorf("ordinal: invalid integer string %q: %w", s, err)
	}

	absN := n
	if absN < 0 {
		absN = -absN
	}

	// Rule 1: Special cases 11th, 12th, 13th
	remainder100 := absN % 100
	if remainder100 >= 11 && remainder100 <= 13 {
		return s + "th", nil
	}

	// Rule 2: Last digit checks
	switch absN % 10 {
	case 1:
		return s + "st", nil
	case 2:
		return s + "nd", nil
	case 3:
		return s + "rd", nil
	default:
		return s + "th", nil
	}
}

package utils

import (
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

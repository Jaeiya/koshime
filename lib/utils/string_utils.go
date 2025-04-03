package utils

import "strings"

/*
ReplaceAll uses the token map to replace all keys with the
mapped value, within the given string.
*/
func ReplaceAll(s string, tokenMap map[string]string) string {
	sb := strings.Builder{}
	for i := 0; i < len(s); {
		var isSwapped bool
		for t, v := range tokenMap {
			if strings.HasPrefix(s[i:], t) {
				sb.WriteString(v)
				i += len(t)
				isSwapped = true
				break
			}
		}
		if !isSwapped {
			sb.WriteByte(s[i])
			i++
		}
	}

	return sb.String()
}

func RemoveBrackets(s string) string {
	if s[0] == '[' || s[0] == '(' {
		s = s[1:]
	}

	if s[len(s)-1] == ']' || s[len(s)-1] == ')' {
		s = s[:len(s)-1]
	}
	return s
}

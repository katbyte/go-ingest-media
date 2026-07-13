package content

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var prefixes = [...]string{"The ", "A ", "An "}

func GetLetter(folder string) string {
	s := folder

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)) {
			s = s[len(prefix):]
		}
	}

	// get first rune
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return "@"
	}

	if r >= '0' && r <= '9' {
		return "0"
	}

	lower := unicode.ToLower(r)
	if lower >= 'a' && lower <= 'z' {
		return string(lower)
	}

	return "@"
}

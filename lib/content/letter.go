package content

import (
	"strconv"
	"strings"
	"unicode"
)

var prefixes = [...]string{"The ", "A ", "An "}

func GetLetter(folder string) string {
	s := folder

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)) {
			s = s[len(prefix):]
		}
	}

	// get first letter
	letter := strings.ToLower(string(s[0]))
	c := letter[0]

	// if a number use 0
	if _, err := strconv.Atoi(letter); err == nil {
		letter = "0"
		// else if not a letter use @
	} else if !unicode.IsLetter(rune(c)) {
		letter = "@"
	}

	return letter
}

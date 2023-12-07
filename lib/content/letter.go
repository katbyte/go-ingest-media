package content

import (
	"strconv"
	"strings"
)

var prefixes = [...]string{"The ", "A ", "An "}

func GetLetter(folder string) string {
	s := folder

	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
		}
	}

	// get first letter
	letter := strings.ToLower(string(s[0]))
	c := letter[0]

	// if a number use 0
	if _, err := strconv.Atoi(letter); err == nil {
		letter = "0"
		// else if not a letter use @
	} else if !('a' <= c && c <= 'z') && !('A' <= c && c <= 'Z') {
		letter = "@"
	}

	return letter
}

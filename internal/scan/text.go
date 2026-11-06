package scan

import (
	"strings"

	"github.com/erdaboss99/gensin-scan/internal/fuzzy"
)

// firstTextLine returns the first line holding any letters or digits.
func firstTextLine(text string) string {
	for line := range strings.Lines(text) {
		if line = strings.TrimSpace(line); fuzzy.Normalize(line) != "" {
			return line
		}
	}
	return ""
}

func keepDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

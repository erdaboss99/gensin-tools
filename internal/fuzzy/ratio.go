package fuzzy

import (
	"strings"
	"unicode"
)

// accentFolder maps accented Latin letters to plain ones, because English OCR models
// read "Déjà Vu" as "Deja Vu".
var accentFolder = strings.NewReplacer(
	"à", "a", "á", "a", "â", "a", "ä", "a", "ã", "a", "å", "a",
	"è", "e", "é", "e", "ê", "e", "ë", "e",
	"ì", "i", "í", "i", "î", "i", "ï", "i",
	"ò", "o", "ó", "o", "ô", "o", "ö", "o", "õ", "o",
	"ù", "u", "ú", "u", "û", "u", "ü", "u",
	"ñ", "n", "ç", "c", "ß", "ss",
)

// Normalize lowercases s, drops punctuation and accents, and collapses whitespace,
// so comparisons ignore characters OCR commonly gets wrong without changing meaning.
func Normalize(s string) string {
	var b strings.Builder
	space := false
	for _, r := range accentFolder.Replace(strings.ToLower(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(unicode.ToLower(r))
		case unicode.IsSpace(r):
			space = true
		}
	}
	return b.String()
}

// Ratio scores the similarity of a and b from 0 to 100 using normalized
// Levenshtein distance. Two empty strings score 0, as they carry no evidence.
func Ratio(a, b string) int {
	ra, rb := []rune(Normalize(a)), []rune(Normalize(b))
	longest := max(len(ra), len(rb))
	if longest == 0 {
		return 0
	}
	return 100 - 100*levenshtein(ra, rb)/longest
}

func levenshtein(a, b []rune) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, ra := range a {
		curr[0] = i + 1
		for j, rb := range b {
			cost := 1
			if ra == rb {
				cost = 0
			}
			curr[j+1] = min(prev[j+1]+1, curr[j]+1, prev[j]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

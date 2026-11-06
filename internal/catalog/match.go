package catalog

import (
	"strings"

	"github.com/erdaboss99/gensin-scan/internal/fuzzy"
)

// Minimum match scores. Long names tolerate a few misread characters, while
// achievements get a stricter bar because many names differ by only a word.
const (
	minPieceScore       = 70
	minCharacterScore   = 75
	minAchievementScore = 85
)

// MatchPiece finds the artifact piece named by OCR text.
func (c *Catalog) MatchPiece(text string) (fuzzy.Match[Piece], bool) {
	return fuzzy.Best(text, c.matchablePieces, func(p Piece) string { return p.Name }, minPieceScore)
}

// MatchCharacter finds the character named by OCR text.
func (c *Catalog) MatchCharacter(text string) (fuzzy.Match[string], bool) {
	return fuzzy.Best(text, c.Characters, fuzzy.Identity, minCharacterScore)
}

// MatchAchievement finds the achievement named by OCR text.
func (c *Catalog) MatchAchievement(text string) (fuzzy.Match[Achievement], bool) {
	return fuzzy.Best(text, c.Achievements, func(a Achievement) string { return a.Name }, minAchievementScore)
}

// IsAmbiguous reports whether an in-game search for this achievement can also
// return a different one, so its result has to be identified by reading the title.
func (c *Catalog) IsAmbiguous(a Achievement) bool {
	return c.ambiguous()[fuzzy.Normalize(a.Name)]
}

func findAmbiguous(all []Achievement) nameSet {
	names := make([]string, len(all))
	for i, a := range all {
		names[i] = fuzzy.Normalize(a.Name)
	}
	out := make(nameSet)
	for i, a := range names {
		for j, b := range names {
			if i != j && strings.Contains(b, a) {
				out[a], out[b] = true, true
			}
		}
	}
	return out
}

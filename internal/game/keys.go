package game

import (
	"strings"
	"unicode"
)

// SetKey is a GOOD artifact set key, such as "GladiatorsFinale".
type SetKey string

// CharacterKey is a GOOD character key, such as "HuTao".
type CharacterKey string

// CharacterTraveler is the key for the player character, whatever their chosen name.
const CharacterTraveler CharacterKey = "Traveler"

// ToKey converts a display name to the PascalCase form GOOD uses for set and
// character keys: "Gladiator's Finale" becomes "GladiatorsFinale".
func ToKey(name string) string {
	var b strings.Builder
	for word := range strings.FieldsSeq(name) {
		first := true
		for _, r := range word {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				continue
			}
			if first {
				r = unicode.ToUpper(r)
				first = false
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

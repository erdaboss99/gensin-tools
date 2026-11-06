package scan

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/erdaboss99/gensin-scan/internal/fuzzy"
	"github.com/erdaboss99/gensin-scan/internal/game"
)

// minSubstatScore is the similarity at which text counts as a substat label.
const minSubstatScore = 70

var substatLabels = []string{"HP", "ATK", "DEF", "Elemental Mastery", "Energy Recharge", "CRIT Rate", "CRIT DMG"}

// statVariants names the flat and percent stats that share one label. A stat with
// only one form leaves the other empty.
type statVariants struct {
	flat, percent game.Stat
}

var substatVariants = map[string]statVariants{
	"HP":                {game.StatHP, game.StatHPPercent},
	"ATK":               {game.StatATK, game.StatATKPercent},
	"DEF":               {game.StatDEF, game.StatDEFPercent},
	"Elemental Mastery": {flat: game.StatElementalMastery},
	"Energy Recharge":   {percent: game.StatEnergyRecharge},
	"CRIT Rate":         {percent: game.StatCritRate},
	"CRIT DMG":          {percent: game.StatCritDMG},
}

var (
	numberPattern  = regexp.MustCompile(`\d+(?:\.\d+)?`)
	pendingPattern = regexp.MustCompile(`(?i)\(?\s*(pending|unactivated|activation|activated).*$`)
	ocrDigitFixes  = strings.NewReplacer("O", "0", "o", "0", "l", "1", "I", "1", "|", "1", "t", "1", "S", "5", "s", "5", "B", "8", ",", ".", "..", ".", " ", "")
)

// applySubstatLine parses a line such as "CRIT Rate+3.9%" into the record.
// Lines that are not substats, such as a set bonus header, are ignored.
func (record *ArtifactRecord) applySubstatLine(line string) {
	body := line
	pending := false
	if loc := pendingPattern.FindStringIndex(body); loc != nil {
		pending = true
		body = body[:loc[0]]
	}

	label, rawValue, ok := splitSubstat(body)
	if !ok {
		return
	}
	match, ok := fuzzy.Best(label, substatLabels, fuzzy.Identity, minSubstatScore)
	if !ok {
		return
	}
	numeric := ocrDigitFixes.Replace(rawValue)
	token := numberPattern.FindString(numeric)
	if token == "" {
		record.issue("substat %q has no readable value", line)
		return
	}

	reading, resolved := resolveSubstat(match.Item, token, strings.Contains(numeric, "%"), record.Artifact.Rarity)
	if !resolved {
		record.issue("substat %q has an implausible value", line)
		return
	}
	if reading.repaired {
		record.correction("substat %q read as %s=%v", line, reading.substat.Key, reading.substat.Value)
	}
	if pending {
		record.Artifact.Unactivated = append(record.Artifact.Unactivated, reading.substat)
	} else {
		record.Artifact.Substats = append(record.Artifact.Substats, reading.substat)
	}
}

// substatReading is a substat recovered from OCR text.
type substatReading struct {
	substat game.Substat
	// repaired means the value needed fixing, such as restoring a lost decimal point.
	repaired bool
}

// splitSubstat separates a line into its label and value. Normally the plus sign
// divides them, but OCR can read "+1" as letters, giving "Energy Rechargetl6.2%". In
// that case the label is recognized by its length, and whatever follows it is the value.
func splitSubstat(body string) (label, value string, ok bool) {
	if cut := strings.LastIndexByte(body, '+'); cut > 0 {
		return body[:cut], body[cut:], true
	}
	body = strings.TrimLeftFunc(body, func(r rune) bool { return !unicode.IsLetter(r) })
	best, bestScore := "", 0
	for _, candidate := range substatLabels {
		if len(body) > len(candidate) {
			if score := fuzzy.Ratio(body[:len(candidate)], candidate); score > bestScore {
				best, bestScore = candidate, score
			}
		}
	}
	if bestScore < minSubstatScore {
		return "", "", false
	}
	value = strings.TrimLeft(body[len(best):], " ")
	// The first character stands in for the plus sign unless it's a digit. When a
	// decimal point follows, it's the misread first digit instead: "+5.2%" as "s.2%".
	if len(value) > 1 && (value[0] < '0' || value[0] > '9') && value[1] != '.' {
		value = value[1:]
	}
	if value == "" || !strings.ContainsAny(value, "0123456789") {
		return "", "", false
	}
	return best, value, true
}

// resolveSubstat decides between flat and percent and repairs lost decimal points
// using the set of values that can actually roll. With an unknown rarity nothing
// can be cross-checked, so values are accepted as read.
func resolveSubstat(label, token string, hasPercent bool, rarity game.Rarity) (substatReading, bool) {
	variants := substatVariants[label]
	whole, fraction, hasDot := strings.Cut(token, ".")

	if variants.percent != "" && (hasDot || hasPercent) {
		text := token
		repaired := false
		if hasDot {
			// A misread "%" can add digits ("5.8%" as "5.896"). One decimal is
			// all the game ever shows.
			text = whole + "." + fraction[:1]
		} else if len(whole) >= 2 {
			text = withLastDigitAsDecimal(whole)
			repaired = true
		}
		return readSubstatValue(variants.percent, text, rarity, repaired)
	}

	digits := whole + fraction
	if variants.flat != "" {
		if reading, ok := readSubstatValue(variants.flat, digits, rarity, hasDot); ok {
			return reading, true
		}
	}
	if variants.percent != "" && len(digits) >= 2 {
		// The value lost both its decimal point and percent sign.
		return readSubstatValue(variants.percent, withLastDigitAsDecimal(digits), rarity, true)
	}
	return substatReading{}, false
}

// readSubstatValue parses text and accepts it if the stat can roll that value.
func readSubstatValue(stat game.Stat, text string, rarity game.Rarity, repaired bool) (substatReading, bool) {
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || (rarity.Valid() && !game.PlausibleSubstat(rarity, stat, value)) {
		return substatReading{}, false
	}
	return substatReading{substat: game.Substat{Key: stat, Value: value}, repaired: repaired}, true
}

// withLastDigitAsDecimal restores a lost decimal point, turning "39" into "3.9".
func withLastDigitAsDecimal(digits string) string {
	return digits[:len(digits)-1] + "." + digits[len(digits)-1:]
}

package scan

import (
	"context"
	"image"
	"image/color"
	"slices"
	"strconv"
	"strings"

	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/fuzzy"
	"github.com/erdaboss99/gensin-scan/internal/game"
	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
	"github.com/erdaboss99/gensin-scan/ocr"
)

const (
	minSlotScore     = 80
	minMainStatScore = 60
	minOverrideScore = 70
	minUnrelated     = 85

	digitsAllowed = "+0123456789"

	// elixirMinBlueExcess is how much bluer than green the band's lilac is.
	elixirMinBlueExcess = 30
	// iconOnMaxBrightness separates a toggle's dark "on" background from its light
	// "off" one.
	iconOnMaxBrightness = 180
)

// artifactStar is the colour of a filled rarity star.
var artifactStar = imaging.ColorRange{Target: color.RGBA{R: 255, G: 204, B: 50, A: 255}, Tolerance: 12}

// unrelatedItems are upgrade materials listed after the artifacts in the same
// inventory tab, which look like artifacts to the grid walker.
var unrelatedItems = []string{"Sanctifying Essence", "Sanctifying Unction", "Sanctifying Elixir"}

// ArtifactParser extracts artifacts from inventory frames.
type ArtifactParser struct {
	Layout  layout.Inventory
	OCR     ocr.Recognizer
	Catalog *catalog.Catalog
	// TravelerName is the player's chosen name, which the game shows instead of
	// "Traveler" on equipped artifacts.
	TravelerName string
}

// IsUnrelated reports whether the frame shows an upgrade material, not an artifact.
func (p ArtifactParser) IsUnrelated(ctx context.Context, frame *image.RGBA) (bool, error) {
	r, err := newReader(ctx, p.OCR, frame)
	if err != nil {
		return false, err
	}
	name, err := r.readName(p.Layout.Name)
	if err != nil {
		return false, err
	}
	return isUnrelatedName(name), nil
}

func isUnrelatedName(name string) bool {
	_, ok := fuzzy.Best(name, unrelatedItems, fuzzy.Identity, minUnrelated)
	return ok
}

// Parse reads one artifact. Unreadable fields become Issues rather than errors;
// an error means OCR could not run at all. skipped is true for upgrade materials.
func (p ArtifactParser) Parse(ctx context.Context, frame *image.RGBA) (record ArtifactRecord, skipped bool, err error) {
	r, err := newReader(ctx, p.OCR, frame)
	if err != nil {
		record.Issues = append(record.Issues, err.Error())
		return record, false, nil
	}

	if record.Raw.Name, err = r.readName(p.Layout.Name); err != nil {
		return record, false, err
	}
	if isUnrelatedName(record.Raw.Name) {
		return record, true, nil
	}

	a := &record.Artifact
	piece, pieceFound := p.Catalog.MatchPiece(record.Raw.Name)
	if pieceFound {
		a.Piece = piece.Item.Name
		a.SetName = piece.Item.Set
		a.SetKey = game.SetKey(game.ToKey(piece.Item.Set))
	} else {
		record.issue("unrecognized artifact name %q", record.Raw.Name)
	}

	if record.Raw.Slot, err = r.read(p.Layout.Slot, ocr.ModeLine, ""); err != nil {
		return record, false, err
	}
	p.resolveSlot(&record, piece.Item, pieceFound)

	switch a.Slot {
	case game.SlotFlower:
		a.MainStat = game.StatHP
	case game.SlotPlume:
		a.MainStat = game.StatATK
	default:
		if record.Raw.MainStat, err = r.read(p.Layout.MainStat, ocr.ModeLine, ""); err != nil {
			return record, false, err
		}
		resolveMainStat(&record)
	}

	a.Rarity = p.countRarity(r)
	if a.Rarity == 0 {
		record.issue("no rarity stars found")
	}

	// The elixir band pushes everything below it down the panel.
	shift := 0
	if a.ElixirCrafted = isElixirBand(imaging.MeanColor(frame, r.win.Rect(p.Layout.ElixirBand))); a.ElixirCrafted {
		shift = p.Layout.ElixirShift
	}
	a.Lock = iconIsOn(imaging.MeanColor(frame, r.win.Rect(p.Layout.LockIcon.Shifted(shift))))
	a.AstralMark = iconIsOn(imaging.MeanColor(frame, r.win.Rect(p.Layout.AstralIcon.Shifted(shift))))

	if record.Raw.Level, err = r.read(p.Layout.Level.Shifted(shift), ocr.ModeLine, digitsAllowed); err != nil {
		return record, false, err
	}
	digits := keepDigits(record.Raw.Level)
	if level, convErr := strconv.Atoi(digits); convErr == nil {
		a.Level = game.Level(level)
	} else {
		record.issue("unreadable level %q", record.Raw.Level)
	}

	substats, err := r.read(p.Layout.Substats.Shifted(shift), ocr.ModeBlock, "")
	if err != nil {
		return record, false, err
	}
	for line := range strings.Lines(substats) {
		if line = strings.TrimSpace(line); line != "" {
			record.Raw.Substats = append(record.Raw.Substats, line)
			record.applySubstatLine(line)
		}
	}

	if record.Raw.Equipped, err = r.read(p.Layout.Equipped, ocr.ModeLine, ""); err != nil {
		return record, false, err
	}
	a.Location = p.resolveLocation(record.Raw.Equipped)

	if verr := a.Validate(); verr != nil {
		for line := range strings.SplitSeq(verr.Error(), "\n") {
			if !slices.Contains(record.Issues, line) {
				record.issue("%s", line)
			}
		}
	}
	return record, false, nil
}

// isElixirBand tells the band's lilac apart from the panel's usual cream.
func isElixirBand(c color.RGBA) bool {
	return int(c.B)-int(c.G) > elixirMinBlueExcess
}

// iconIsOn reads a toggle icon's background: dark when on, near-white when off. The
// icons fade between states, and even frames caught mid-fade fall on the right side.
func iconIsOn(c color.RGBA) bool {
	return (int(c.R)+int(c.G)+int(c.B))/3 < iconOnMaxBrightness
}

// resolveSlot prefers the slot printed on the panel. The catalog is a fallback
// only, because the wiki's slot data has been wrong for several sets.
func (p ArtifactParser) resolveSlot(record *ArtifactRecord, piece catalog.Piece, pieceFound bool) {
	match, ok := fuzzy.Best(record.Raw.Slot, game.AllSlots(), game.Slot.Label, minSlotScore)
	switch {
	case ok:
		record.Artifact.Slot = match.Item
		if pieceFound && piece.Slot != "" && piece.Slot != match.Item {
			record.correction("panel says %s but catalog says %s for %q", match.Item, piece.Slot, piece.Name)
		}
	case pieceFound && piece.Slot != "":
		record.Artifact.Slot = piece.Slot
		record.correction("slot %q unreadable, used catalog value %s", record.Raw.Slot, piece.Slot)
	default:
		record.issue("unreadable slot %q", record.Raw.Slot)
	}
}

func resolveMainStat(record *ArtifactRecord) {
	candidates := game.MainStatsFor(record.Artifact.Slot)
	if len(candidates) == 0 {
		// Without a slot there is no safe way to choose between stats that share
		// a label, and an artifact without a slot cannot be exported anyway.
		record.issue("main stat %q skipped because the slot is unknown", record.Raw.MainStat)
		return
	}
	match, ok := fuzzy.Best(record.Raw.MainStat, candidates, game.Stat.Label, minMainStatScore)
	if !ok {
		record.issue("unreadable main stat %q", record.Raw.MainStat)
		return
	}
	record.Artifact.MainStat = match.Item
}

// countRarity counts star-shaped blobs of the star colour.
func (p ArtifactParser) countRarity(r reader) game.Rarity {
	minArea, maxArea := r.win.Area(p.Layout.StarMinArea), r.win.Area(p.Layout.StarMaxArea)
	stars := 0
	for _, blob := range imaging.FindBlobs(r.frame, r.win.Rect(p.Layout.Rarity), artifactStar) {
		if blob.Area >= minArea && blob.Area <= maxArea {
			stars++
		}
	}
	if !game.Rarity(stars).Valid() {
		return 0
	}
	return game.Rarity(stars)
}

func (p ArtifactParser) resolveLocation(text string) game.CharacterKey {
	text = strings.TrimSpace(text)
	if i := strings.IndexByte(text, ':'); i >= 0 {
		text = text[i+1:]
	}
	if fuzzy.Normalize(text) == "" {
		return ""
	}
	if p.TravelerName != "" && fuzzy.Ratio(text, p.TravelerName) >= minOverrideScore {
		return game.CharacterTraveler
	}
	if match, ok := p.Catalog.MatchCharacter(text); ok {
		return game.CharacterKey(game.ToKey(match.Item))
	}
	return ""
}

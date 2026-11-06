package game

import (
	"errors"
	"fmt"
	"slices"
)

// Substat is one rolled artifact substat. Percent values are stored as shown
// in game, so 3.9 means 3.9%.
type Substat struct {
	Key   Stat    `json:"key"`
	Value float64 `json:"value"`
}

// Artifact is a single scanned artifact. Fields the scanner could not read keep
// their zero value; Validate reports them.
type Artifact struct {
	SetKey   SetKey       `json:"setKey,omitempty"`
	SetName  string       `json:"setName,omitempty"`
	Piece    string       `json:"piece,omitempty"`
	Slot     Slot         `json:"slot,omitempty"`
	Rarity   Rarity       `json:"rarity,omitempty"`
	Level    Level        `json:"level"`
	MainStat Stat         `json:"mainStat,omitempty"`
	Location CharacterKey `json:"location,omitempty"`
	Lock     bool         `json:"lock,omitempty"`
	// AstralMark is the game's favourite marker, shown as a star beside the lock.
	AstralMark bool `json:"astralMark,omitempty"`
	// ElixirCrafted marks artifacts made with Sanctifying Elixir.
	ElixirCrafted bool      `json:"elixirCrafted,omitempty"`
	Substats      []Substat `json:"substats,omitempty"`
	// Unactivated holds substats the game lists as "pending activation".
	Unactivated []Substat `json:"unactivatedSubstats,omitempty"`
}

// Validate checks the artifact against the game's rules. It returns every
// problem found, joined, or nil when the artifact is safe to export.
func (a Artifact) Validate() error {
	var errs []error
	if a.SetKey == "" {
		errs = append(errs, errors.New("artifact set is unknown"))
	}
	if !a.Slot.Valid() {
		errs = append(errs, errors.New("slot is unknown"))
	}
	if !a.Rarity.Valid() {
		errs = append(errs, errors.New("rarity is unknown"))
	} else if a.Level < 0 || a.Level > a.Rarity.MaxLevel() {
		errs = append(errs, fmt.Errorf("level %d is impossible for a %d-star artifact", a.Level, a.Rarity))
	}
	if !a.MainStat.Valid() {
		errs = append(errs, errors.New("main stat is unknown"))
	} else if a.Slot.Valid() && !slices.Contains(mainStats[a.Slot], a.MainStat) {
		errs = append(errs, fmt.Errorf("main stat %s cannot appear on a %s", a.MainStat, a.Slot))
	}

	all := slices.Concat(a.Substats, a.Unactivated)
	if len(all) > 4 {
		errs = append(errs, fmt.Errorf("artifact has %d substats, at most 4 are possible", len(all)))
	}
	if want := a.Rarity.minSubstats(a.Level); len(all) < want {
		errs = append(errs, fmt.Errorf("%d substats found, a %d-star artifact at level %d has at least %d", len(all), a.Rarity, a.Level, want))
	}
	seen := make(map[Stat]bool, len(all))
	for _, sub := range all {
		switch {
		case !slices.Contains(SubstatTypes(), sub.Key):
			errs = append(errs, fmt.Errorf("%q is not a substat", sub.Key))
		case sub.Key == a.MainStat:
			errs = append(errs, fmt.Errorf("substat %s duplicates the main stat", sub.Key))
		case seen[sub.Key]:
			errs = append(errs, fmt.Errorf("substat %s appears twice", sub.Key))
		}
		seen[sub.Key] = true
	}
	return errors.Join(errs...)
}

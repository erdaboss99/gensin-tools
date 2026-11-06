package game

import (
	"encoding/json"
	"fmt"
)

// Rarity is an artifact's star count, from 1 to 5.
type Rarity int

// Valid reports whether r is between one and five stars.
func (r Rarity) Valid() bool { return r >= 1 && r <= 5 }

// MaxLevel returns the highest level an artifact of this rarity can reach.
func (r Rarity) MaxLevel() Level {
	switch r {
	case 5:
		return 20
	case 4:
		return 16
	case 3:
		return 12
	case 1, 2:
		return 4
	default:
		return 0
	}
}

// minSubstats returns the fewest substats an artifact can have at a level: its
// starting number plus one per four levels, up to four. Pending substats count.
func (r Rarity) minSubstats(level Level) int {
	var initial int
	switch r {
	case 5:
		initial = 3
	case 4:
		initial = 2
	case 3:
		initial = 1
	default:
		return 0
	}
	return min(initial+int(level)/4, 4)
}

// UnmarshalJSON rejects out-of-range rarities when reading saved scans.
func (r *Rarity) UnmarshalJSON(data []byte) error {
	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if !Rarity(v).Valid() {
		return fmt.Errorf("rarity %d is out of range", v)
	}
	*r = Rarity(v)
	return nil
}

// Level is an artifact's enhancement level.
type Level int

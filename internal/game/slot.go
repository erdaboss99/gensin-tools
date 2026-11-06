package game

import (
	"fmt"
	"strings"
)

// Slot is an artifact equipment slot.
type Slot string

// The five artifact slots.
const (
	SlotFlower  Slot = "flower"
	SlotPlume   Slot = "plume"
	SlotSands   Slot = "sands"
	SlotGoblet  Slot = "goblet"
	SlotCirclet Slot = "circlet"
)

var slotLabels = map[Slot]string{
	SlotFlower:  "Flower of Life",
	SlotPlume:   "Plume of Death",
	SlotSands:   "Sands of Eon",
	SlotGoblet:  "Goblet of Eonothem",
	SlotCirclet: "Circlet of Logos",
}

// AllSlots returns every slot in in-game order.
func AllSlots() []Slot {
	return []Slot{SlotFlower, SlotPlume, SlotSands, SlotGoblet, SlotCirclet}
}

// Label returns the English name the game displays for the slot.
func (s Slot) Label() string { return slotLabels[s] }

// Valid reports whether s is one of the five known slots.
func (s Slot) Valid() bool {
	_, ok := slotLabels[s]
	return ok
}

// SlotFromLabel converts a display name such as "Sands of Eon" to its Slot.
func SlotFromLabel(label string) (Slot, bool) {
	for slot, l := range slotLabels {
		if strings.EqualFold(l, strings.TrimSpace(label)) {
			return slot, true
		}
	}
	return "", false
}

// UnmarshalText rejects unknown slots when reading saved scans.
func (s *Slot) UnmarshalText(text []byte) error {
	v := Slot(text)
	if !v.Valid() {
		return fmt.Errorf("unknown artifact slot %q", text)
	}
	*s = v
	return nil
}

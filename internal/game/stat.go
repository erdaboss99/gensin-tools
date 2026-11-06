package game

import (
	"fmt"
	"slices"
)

// Stat is an artifact main stat or substat.
type Stat string

// All artifact stats, named with their GOOD keys.
const (
	StatHP               Stat = "hp"
	StatHPPercent        Stat = "hp_"
	StatATK              Stat = "atk"
	StatATKPercent       Stat = "atk_"
	StatDEF              Stat = "def"
	StatDEFPercent       Stat = "def_"
	StatElementalMastery Stat = "eleMas"
	StatEnergyRecharge   Stat = "enerRech_"
	StatHealingBonus     Stat = "heal_"
	StatCritRate         Stat = "critRate_"
	StatCritDMG          Stat = "critDMG_"
	StatPhysicalDMG      Stat = "physical_dmg_"
	StatAnemoDMG         Stat = "anemo_dmg_"
	StatGeoDMG           Stat = "geo_dmg_"
	StatElectroDMG       Stat = "electro_dmg_"
	StatHydroDMG         Stat = "hydro_dmg_"
	StatPyroDMG          Stat = "pyro_dmg_"
	StatCryoDMG          Stat = "cryo_dmg_"
	StatDendroDMG        Stat = "dendro_dmg_"
)

// statLabels holds the text the game prints for each stat. Flat and percent
// variants share a label; the value's format distinguishes them.
var statLabels = map[Stat]string{
	StatHP:               "HP",
	StatHPPercent:        "HP",
	StatATK:              "ATK",
	StatATKPercent:       "ATK",
	StatDEF:              "DEF",
	StatDEFPercent:       "DEF",
	StatElementalMastery: "Elemental Mastery",
	StatEnergyRecharge:   "Energy Recharge",
	StatHealingBonus:     "Healing Bonus",
	StatCritRate:         "CRIT Rate",
	StatCritDMG:          "CRIT DMG",
	StatPhysicalDMG:      "Physical DMG Bonus",
	StatAnemoDMG:         "Anemo DMG Bonus",
	StatGeoDMG:           "Geo DMG Bonus",
	StatElectroDMG:       "Electro DMG Bonus",
	StatHydroDMG:         "Hydro DMG Bonus",
	StatPyroDMG:          "Pyro DMG Bonus",
	StatCryoDMG:          "Cryo DMG Bonus",
	StatDendroDMG:        "Dendro DMG Bonus",
}

// Label returns the English name the game displays for the stat.
func (s Stat) Label() string { return statLabels[s] }

// Valid reports whether s is a known stat.
func (s Stat) Valid() bool {
	_, ok := statLabels[s]
	return ok
}

// IsFlat reports whether the stat is displayed as a whole number.
func (s Stat) IsFlat() bool {
	return s == StatHP || s == StatATK || s == StatDEF || s == StatElementalMastery
}

// UnmarshalText rejects unknown stats when reading saved scans.
func (s *Stat) UnmarshalText(text []byte) error {
	v := Stat(text)
	if !v.Valid() {
		return fmt.Errorf("unknown artifact stat %q", text)
	}
	*s = v
	return nil
}

var mainStats = map[Slot][]Stat{
	SlotFlower: {StatHP},
	SlotPlume:  {StatATK},
	SlotSands:  {StatHPPercent, StatATKPercent, StatDEFPercent, StatEnergyRecharge, StatElementalMastery},
	SlotGoblet: {
		StatHPPercent, StatATKPercent, StatDEFPercent, StatElementalMastery,
		StatPhysicalDMG, StatAnemoDMG, StatGeoDMG, StatElectroDMG, StatHydroDMG, StatPyroDMG, StatCryoDMG, StatDendroDMG,
	},
	SlotCirclet: {StatHPPercent, StatATKPercent, StatDEFPercent, StatElementalMastery, StatHealingBonus, StatCritRate, StatCritDMG},
}

// MainStatsFor returns the main stats that can roll on the given slot.
func MainStatsFor(slot Slot) []Stat { return slices.Clone(mainStats[slot]) }

// SubstatTypes returns every stat that can appear as a substat.
func SubstatTypes() []Stat {
	return []Stat{
		StatHP, StatHPPercent, StatATK, StatATKPercent, StatDEF, StatDEFPercent,
		StatElementalMastery, StatEnergyRecharge, StatCritRate, StatCritDMG,
	}
}

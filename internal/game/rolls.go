package game

import (
	"math"
	"slices"
	"sync"
)

// rollTable describes how substats roll at one rarity: the largest single roll
// per stat, the fraction of it each roll tier yields, and the most times one
// substat can roll.
type rollTable struct {
	maxRolls int
	tiers    []float64
	maxRoll  statRolls
}

// statRolls holds the largest single roll of each stat.
type statRolls map[Stat]float64

// reachableValues lists every value a substat can display, by rarity and stat.
type reachableValues map[Rarity]map[Stat][]float64

// rollTables holds the published substat roll values for each rarity.
var rollTables = map[Rarity]rollTable{
	5: {6, []float64{1, 0.9, 0.8, 0.7}, statRolls{
		StatHP: 298.75, StatHPPercent: 5.83, StatATK: 19.45, StatATKPercent: 5.83, StatDEF: 23.15, StatDEFPercent: 7.29,
		StatCritRate: 3.89, StatCritDMG: 7.77, StatEnergyRecharge: 6.48, StatElementalMastery: 23.31,
	}},
	4: {5, []float64{1, 0.9, 0.8, 0.7}, statRolls{
		StatHP: 239, StatHPPercent: 4.66, StatATK: 15.56, StatATKPercent: 4.66, StatDEF: 18.52, StatDEFPercent: 5.83,
		StatCritRate: 3.11, StatCritDMG: 6.22, StatEnergyRecharge: 5.18, StatElementalMastery: 18.65,
	}},
	3: {4, []float64{1, 0.9, 0.8, 0.7}, statRolls{
		StatHP: 143.4, StatHPPercent: 3.5, StatATK: 9.34, StatATKPercent: 3.5, StatDEF: 11.11, StatDEFPercent: 4.37,
		StatCritRate: 2.33, StatCritDMG: 4.66, StatEnergyRecharge: 3.89, StatElementalMastery: 13.99,
	}},
	2: {2, []float64{1, 0.85, 0.7}, statRolls{
		StatHP: 71.7, StatHPPercent: 2.33, StatATK: 4.67, StatATKPercent: 2.33, StatDEF: 5.56, StatDEFPercent: 2.91,
		StatCritRate: 1.55, StatCritDMG: 3.11, StatEnergyRecharge: 2.59, StatElementalMastery: 9.33,
	}},
	1: {1, []float64{1, 0.8}, statRolls{
		StatHP: 29.88, StatHPPercent: 1.46, StatATK: 1.95, StatATKPercent: 1.46, StatDEF: 2.31, StatDEFPercent: 1.82,
		StatCritRate: 0.97, StatCritDMG: 1.94, StatEnergyRecharge: 1.62, StatElementalMastery: 5.83,
	}},
}

var substatValues = sync.OnceValue(func() reachableValues {
	out := make(reachableValues, len(rollTables))
	for rarity, table := range rollTables {
		out[rarity] = make(map[Stat][]float64, len(table.maxRoll))
		for stat, top := range table.maxRoll {
			out[rarity][stat] = enumerate(stat, top, table)
		}
	}
	return out
})

// enumerate lists every displayable value reachable by summing rolls.
func enumerate(stat Stat, top float64, table rollTable) []float64 {
	rolls := make([]float64, len(table.tiers))
	for i, tier := range table.tiers {
		rolls[i] = math.Round(tier*top*100) / 100
	}
	seen := make(map[float64]bool)
	var walk func(tier, left int, sum float64)
	walk = func(tier, left int, sum float64) {
		if sum > 0 {
			seen[displayed(stat, sum)] = true
		}
		if tier == len(rolls) {
			return
		}
		for n := range left + 1 {
			walk(tier+1, left-n, sum+float64(n)*rolls[tier])
		}
	}
	walk(0, table.maxRolls, 0)

	values := make([]float64, 0, len(seen))
	for v := range seen {
		values = append(values, v)
	}
	slices.Sort(values)
	return values
}

func displayed(stat Stat, v float64) float64 {
	if stat.IsFlat() {
		return math.Round(v)
	}
	return math.Round(v*10) / 10
}

// PlausibleSubstat reports whether value could be displayed for this stat at
// this rarity. The game rounds from unrounded internal values, so a value within
// one display step of a computed sum counts as plausible.
func PlausibleSubstat(rarity Rarity, stat Stat, value float64) bool {
	values, ok := substatValues()[rarity][stat]
	if !ok {
		return false
	}
	tolerance := 0.11
	if stat.IsFlat() {
		tolerance = 1.01
	}
	i, _ := slices.BinarySearch(values, value)
	for _, j := range []int{i - 1, i} {
		if j >= 0 && j < len(values) && math.Abs(values[j]-value) <= tolerance {
			return true
		}
	}
	return false
}

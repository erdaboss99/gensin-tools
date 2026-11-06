package game

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToKey(t *testing.T) {
	tests := map[string]string{
		"Gladiator's Finale":                      "GladiatorsFinale",
		"Heart of Depth":                          "HeartOfDepth",
		"Nighttime Whispers in the Echoing Woods": "NighttimeWhispersInTheEchoingWoods",
		"Hu Tao":            "HuTao",
		"Kaedehara Kazuha":  "KaedeharaKazuha",
		"Long Night's Oath": "LongNightsOath",
	}
	for in, want := range tests {
		if got := ToKey(in); got != want {
			t.Errorf("ToKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlausibleSubstat(t *testing.T) {
	tests := []struct {
		rarity Rarity
		stat   Stat
		value  float64
		want   bool
	}{
		{5, StatCritRate, 3.9, true},
		{5, StatCritRate, 2.7, true},
		{5, StatCritRate, 7.4, true},
		{5, StatCritRate, 39, false},
		{5, StatCritRate, 1.2, false},
		{5, StatCritDMG, 46.6, true},
		{5, StatATK, 19, true},
		{5, StatATK, 58, true},
		{5, StatATK, 5, false},
		{5, StatHP, 299, true},
		{5, StatHP, 2990, false},
		{5, StatElementalMastery, 16, true},
		{4, StatCritRate, 3.1, true},
		{4, StatCritRate, 3.9, false},
		{5, StatPyroDMG, 3.9, false},
	}
	for _, tt := range tests {
		if got := PlausibleSubstat(tt.rarity, tt.stat, tt.value); got != tt.want {
			t.Errorf("PlausibleSubstat(%d, %s, %v) = %v, want %v", tt.rarity, tt.stat, tt.value, got, tt.want)
		}
	}
}

func TestArtifactValidate(t *testing.T) {
	valid := Artifact{
		SetKey:   "GladiatorsFinale",
		Slot:     SlotCirclet,
		Rarity:   5,
		Level:    20,
		MainStat: StatCritRate,
		Substats: []Substat{{StatCritDMG, 20.2}, {StatATKPercent, 9.9}, {StatHP, 299}, {StatDEF, 23}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid artifact rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Artifact)
		want   string
	}{
		{"missing set", func(a *Artifact) { a.SetKey = "" }, "set is unknown"},
		{"level too high", func(a *Artifact) { a.Rarity, a.Level = 4, 20 }, "impossible"},
		{"wrong main stat", func(a *Artifact) { a.MainStat = StatPyroDMG }, "cannot appear"},
		{"main duplicated", func(a *Artifact) { a.MainStat = StatCritDMG }, "duplicates"},
		{"duplicate substat", func(a *Artifact) { a.Substats[1].Key = StatCritDMG }, "twice"},
		{"too many", func(a *Artifact) { a.Unactivated = []Substat{{StatDEFPercent, 5.8}} }, "at most 4"},
		{"not a substat", func(a *Artifact) { a.Substats[0].Key = StatPyroDMG }, "not a substat"},
		{"lost a substat", func(a *Artifact) { a.Substats = a.Substats[:3] }, "at least 4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := valid
			a.Substats = append([]Substat(nil), valid.Substats...)
			tt.mutate(&a)
			err := a.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestMinimumSubstats(t *testing.T) {
	base := Artifact{SetKey: "X", Slot: SlotFlower, MainStat: StatHP}
	three := []Substat{{StatATK, 19}, {StatDEF, 23}, {StatCritRate, 3.9}}
	tests := []struct {
		rarity Rarity
		level  Level
		subs   []Substat
		ok     bool
	}{
		{5, 0, three, true},
		{5, 3, three, true},
		{5, 4, three, false},
		{4, 4, three, true},
		{4, 8, three, false},
		{4, 0, three[:2], true},
		{4, 0, three[:1], false},
		{3, 8, three, true},
		{2, 4, nil, true},
	}
	for _, tt := range tests {
		a := base
		a.Rarity, a.Level, a.Substats = tt.rarity, tt.level, tt.subs
		if err := a.Validate(); (err == nil) != tt.ok {
			t.Errorf("%d-star +%d with %d substats: err = %v, want ok=%v", tt.rarity, tt.level, len(tt.subs), err, tt.ok)
		}
	}
	pending := base
	pending.Rarity, pending.Level = 5, 4
	pending.Substats, pending.Unactivated = three, []Substat{{StatHPPercent, 5.8}}
	if err := pending.Validate(); err != nil {
		t.Errorf("pending substat not counted: %v", err)
	}
}

func TestSavedScansRejectUnknownEnums(t *testing.T) {
	var a Artifact
	for _, doc := range []string{`{"slot":"hat"}`, `{"mainStat":"luck"}`, `{"rarity":9}`} {
		if err := json.Unmarshal([]byte(doc), &a); err == nil {
			t.Errorf("Unmarshal(%s) succeeded, want error", doc)
		}
	}
	if err := json.Unmarshal([]byte(`{"slot":"sands","mainStat":"eleMas","rarity":5,"level":20}`), &a); err != nil {
		t.Errorf("valid document rejected: %v", err)
	}
}

func TestMainStatsForReturnsCopy(t *testing.T) {
	stats := MainStatsFor(SlotFlower)
	stats[0] = StatPyroDMG
	if MainStatsFor(SlotFlower)[0] != StatHP {
		t.Fatal("MainStatsFor exposed internal state")
	}
}

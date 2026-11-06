package scan

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/game"
	"github.com/erdaboss99/gensin-scan/internal/layout"
)

func TestApplySubstatLine(t *testing.T) {
	tests := []struct {
		line        string
		want        []game.Substat
		unactivated []game.Substat
		issue       bool
		corrected   bool
	}{
		{line: "CRIT Rate+3.9%", want: []game.Substat{{Key: game.StatCritRate, Value: 3.9}}},
		{line: "CRIT DMG+20.2%", want: []game.Substat{{Key: game.StatCritDMG, Value: 20.2}}},
		{line: "HP+299", want: []game.Substat{{Key: game.StatHP, Value: 299}}},
		{line: "HP+5.8%", want: []game.Substat{{Key: game.StatHPPercent, Value: 5.8}}},
		{line: "ATK+19", want: []game.Substat{{Key: game.StatATK, Value: 19}}},
		{line: "ATK+5.8%", want: []game.Substat{{Key: game.StatATKPercent, Value: 5.8}}},
		{line: "DEF+7.3%", want: []game.Substat{{Key: game.StatDEFPercent, Value: 7.3}}},
		{line: "Elemental Mastery+23", want: []game.Substat{{Key: game.StatElementalMastery, Value: 23}}},
		{line: "Energy Recharge+6.5%", want: []game.Substat{{Key: game.StatEnergyRecharge, Value: 6.5}}},

		{line: "• CRIT Rate+3.9%", want: []game.Substat{{Key: game.StatCritRate, Value: 3.9}}},
		{line: "CR1T Rate+3.9%", want: []game.Substat{{Key: game.StatCritRate, Value: 3.9}}},
		{line: "Elemental Masterv+23", want: []game.Substat{{Key: game.StatElementalMastery, Value: 23}}},
		{line: "ATK+5.896", want: []game.Substat{{Key: game.StatATKPercent, Value: 5.8}}},
		{line: "HP+2 99", want: []game.Substat{{Key: game.StatHP, Value: 299}}},
		{line: "CRIT Rate+3,9%", want: []game.Substat{{Key: game.StatCritRate, Value: 3.9}}},
		{line: "HP+1O.5%", want: []game.Substat{{Key: game.StatHPPercent, Value: 10.5}}},

		{line: "CRIT Rate+39%", want: []game.Substat{{Key: game.StatCritRate, Value: 3.9}}, corrected: true},
		{line: "ATK+58%", want: []game.Substat{{Key: game.StatATKPercent, Value: 5.8}}, corrected: true},
		{line: "CRIT DMG+78", want: []game.Substat{{Key: game.StatCritDMG, Value: 7.8}}, corrected: true},

		// Misreads taken from real game frames.
		{line: "Energy Rechargetl6.2%", want: []game.Substat{{Key: game.StatEnergyRecharge, Value: 16.2}}},
		{line: "Energy Rechargett1.0%", want: []game.Substat{{Key: game.StatEnergyRecharge, Value: 11}}},
		{line: "HP+4..2%", want: []game.Substat{{Key: game.StatHPPercent, Value: 4.2}}},
		{line: "Energy Rechargets.2%", want: []game.Substat{{Key: game.StatEnergyRecharge, Value: 5.2}}},
		{line: "Energy Recharget5.8%", want: []game.Substat{{Key: game.StatEnergyRecharge, Value: 5.8}}},
		{line: "Energy Recharget+4.5%", want: []game.Substat{{Key: game.StatEnergyRecharge, Value: 4.5}}},
		{line: "CRIT Rate+7.4.%", want: []game.Substat{{Key: game.StatCritRate, Value: 7.4}}},
		{line: "ATK+14.", want: []game.Substat{{Key: game.StatATK, Value: 14}}},
		{line: "erserker:"},
		{line: "my CJ us"},

		{line: "ATK+5.8% (unactivated)", unactivated: []game.Substat{{Key: game.StatATKPercent, Value: 5.8}}},
		{line: "DEF+16  (unactivated)", unactivated: []game.Substat{{Key: game.StatDEF, Value: 16}}},
		{line: "CRIT Rate+3.9% (Pending Activation)", unactivated: []game.Substat{{Key: game.StatCritRate, Value: 3.9}}},

		{line: "CRIT Rate+99.9%", issue: true},
		{line: "HP+7", issue: true},
		{line: "ATK+", issue: true},

		{line: "Gladiator's Finale:"},
		{line: "Night of the Sky's Unveiling:"},
		{line: "Berserker:"},
		{line: "2-Piece Set: ATK +18%"},
		{line: "+"},
		{line: ""},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			record := ArtifactRecord{Artifact: game.Artifact{Rarity: 5}}
			record.applySubstatLine(tt.line)
			if !slices.Equal(record.Artifact.Substats, tt.want) {
				t.Errorf("Substats = %+v, want %+v", record.Artifact.Substats, tt.want)
			}
			if !slices.Equal(record.Artifact.Unactivated, tt.unactivated) {
				t.Errorf("Unactivated = %+v, want %+v", record.Artifact.Unactivated, tt.unactivated)
			}
			if got := len(record.Issues) > 0; got != tt.issue {
				t.Errorf("Issues = %v, want issue=%v", record.Issues, tt.issue)
			}
			if got := len(record.Corrections) > 0; got != tt.corrected {
				t.Errorf("Corrections = %v, want corrected=%v", record.Corrections, tt.corrected)
			}
		})
	}
}

func TestSubstatPlausibilityDependsOnRarity(t *testing.T) {
	record := ArtifactRecord{Artifact: game.Artifact{Rarity: 3}}
	record.applySubstatLine("CRIT Rate+3.9%")
	if len(record.Issues) != 0 {
		// 3.9 is two low rolls at 3 stars, which is reachable.
		t.Errorf("reachable value rejected: %v", record.Issues)
	}
	record = ArtifactRecord{Artifact: game.Artifact{Rarity: 1}}
	record.applySubstatLine("CRIT Rate+3.9%")
	if len(record.Issues) == 0 {
		t.Error("1-star artifact accepted a value it cannot roll")
	}
	record = ArtifactRecord{}
	record.applySubstatLine("CRIT Rate+3.9%")
	if len(record.Artifact.Substats) != 1 {
		t.Error("unknown rarity should accept values as read")
	}
}

func TestResolveLocation(t *testing.T) {
	parser := ArtifactParser{
		Catalog:      catalog.New(catalog.Data{Characters: []string{"Hu Tao", "Traveler", "Kaedehara Kazuha"}}),
		TravelerName: "Lumine",
	}
	tests := map[string]game.CharacterKey{
		"Hu Tao":               "HuTao",
		"Equipped: Hu Tao":     "HuTao",
		"Kaedehara Kazuhe":     "KaedeharaKazuha",
		"Lumine":               "Traveler",
		"Equipped: Lumlne":     "Traveler",
		"":                     "",
		"  ":                   "",
		"~":                    "",
		"completely unrelated": "",
	}
	for text, want := range tests {
		if got := parser.resolveLocation(text); got != want {
			t.Errorf("resolveLocation(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestParseSlotPrecedence(t *testing.T) {
	cat := catalog.New(catalog.Data{Pieces: []catalog.Piece{
		{Name: "Wrong Slot Data", Set: "Some Set", Slot: game.SlotPlume},
		{Name: "Unknown Slot Data", Set: "Some Set"},
	}})
	parser := ArtifactParser{Layout: layout.Default().Inventory, Catalog: cat}

	t.Run("panel beats catalog", func(t *testing.T) {
		parser.OCR = &scriptedOCR{replies: []string{"Wrong Slot Data", "Sands of Eon", "ATK", "+0", "", ""}}
		record, _, err := parser.Parse(t.Context(), blankFrame())
		if err != nil {
			t.Fatal(err)
		}
		if record.Artifact.Slot != game.SlotSands {
			t.Errorf("Slot = %q, want sands", record.Artifact.Slot)
		}
		if record.Artifact.MainStat != game.StatATKPercent {
			t.Errorf("MainStat = %q, want the percent stat, as flat ATK cannot be on sands", record.Artifact.MainStat)
		}
		if len(record.Corrections) != 1 {
			t.Errorf("Corrections = %v", record.Corrections)
		}
	})
	t.Run("catalog fills in unreadable panel", func(t *testing.T) {
		parser.OCR = &scriptedOCR{replies: []string{"Wrong Slot Data", "###", "+0", "", ""}}
		record, _, err := parser.Parse(t.Context(), blankFrame())
		if err != nil {
			t.Fatal(err)
		}
		if record.Artifact.Slot != game.SlotPlume || record.Artifact.MainStat != game.StatATK {
			t.Errorf("Artifact = %+v", record.Artifact)
		}
	})
	t.Run("nothing to go on", func(t *testing.T) {
		parser.OCR = &scriptedOCR{replies: []string{"Unknown Slot Data", "###", "", "+0", "", ""}}
		record, _, err := parser.Parse(t.Context(), blankFrame())
		if err != nil {
			t.Fatal(err)
		}
		if record.Artifact.Slot != "" || record.Exportable() {
			t.Errorf("Artifact = %+v, want no slot and not exportable", record.Artifact)
		}
	})
}

func TestExportableRequiresNoIssues(t *testing.T) {
	record := ArtifactRecord{Artifact: game.Artifact{SetKey: "X", Slot: game.SlotFlower, Rarity: 2, MainStat: game.StatHP}}
	if !record.Exportable() {
		t.Fatalf("clean record rejected: %v", record.Artifact.Validate())
	}
	record.Issues = []string{"substat has an implausible value"}
	if record.Exportable() {
		t.Error("record with an issue counted as exportable")
	}
}

func TestPanelToggles(t *testing.T) {
	// Colours measured from real frames, including ones caught mid-fade.
	on := []color.RGBA{{73, 83, 102, 255}, {75, 85, 102, 255}, {109, 114, 129, 255}, {139, 143, 152, 255}}
	off := []color.RGBA{{243, 239, 233, 255}, {243, 239, 234, 255}, {219, 217, 215, 255}}
	for _, c := range on {
		if !iconIsOn(c) {
			t.Errorf("iconIsOn(%v) = false", c)
		}
	}
	for _, c := range off {
		if iconIsOn(c) {
			t.Errorf("iconIsOn(%v) = true", c)
		}
	}
	if !isElixirBand(color.RGBA{220, 192, 255, 255}) {
		t.Error("elixir band not recognized")
	}
	if isElixirBand(color.RGBA{236, 229, 216, 255}) {
		t.Error("cream panel recognized as an elixir band")
	}
}

func TestParseFlagsWrongWindowAsIssue(t *testing.T) {
	parser := ArtifactParser{Layout: layout.Default().Inventory, OCR: &scriptedOCR{}, Catalog: catalog.New(catalog.Data{})}
	record, _, err := parser.Parse(t.Context(), image.NewRGBA(image.Rect(0, 0, 640, 480)))
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Issues) == 0 || record.Exportable() {
		t.Errorf("record = %+v, want an issue", record)
	}
}

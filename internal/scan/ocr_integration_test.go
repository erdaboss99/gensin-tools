package scan

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/fuzzy"
	"github.com/erdaboss99/gensin-scan/internal/game"
	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
	"github.com/erdaboss99/gensin-scan/ocr"
)

// These tests run the real pipeline, including Tesseract, against the synthetic
// screenshots in testdata. They are skipped when Tesseract is not installed.

func realOCR(t *testing.T) ocr.Tesseract {
	t.Helper()
	path, err := ocr.Locate(os.Getenv("GENSIN_TESSERACT"))
	if err != nil {
		t.Skip("tesseract is not installed")
	}
	return ocr.Tesseract{Path: path, Lang: "eng"}
}

func loadCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	cat, err := catalog.Load("")
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func loadFixture(t *testing.T, name string) Frame {
	t.Helper()
	path := filepath.Join("testdata", name)
	img, err := imaging.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return Frame{Source: path, Image: img}
}

func TestRealOCRArtifacts(t *testing.T) {
	parser := ArtifactParser{Layout: layout.Default().Inventory, OCR: realOCR(t), Catalog: loadCatalog(t)}

	tests := []struct {
		fixture string
		want    game.Artifact
	}{
		{
			fixture: "artifact_circlet_1280.png",
			want: game.Artifact{
				SetKey: "NightOfTheSkysUnveiling", SetName: "Night of the Sky's Unveiling", Piece: "Crown of the Befallen",
				Slot: game.SlotCirclet, Rarity: 5, Level: 20, MainStat: game.StatCritRate, Location: "HuTao", Lock: true, AstralMark: true,
				Substats: []game.Substat{
					{Key: game.StatCritDMG, Value: 20.2}, {Key: game.StatATKPercent, Value: 9.9},
					{Key: game.StatHP, Value: 299}, {Key: game.StatDEF, Value: 23},
				},
			},
		},
		{
			fixture: "artifact_goblet_1920.png",
			want: game.Artifact{
				SetKey: "ScarletProof", SetName: "Scarlet Proof", Piece: "Chalice of Your Blood and Sorrow",
				Slot: game.SlotGoblet, Rarity: 5, Level: 0, MainStat: game.StatPyroDMG,
				Substats: []game.Substat{
					{Key: game.StatCritRate, Value: 3.9}, {Key: game.StatElementalMastery, Value: 23},
					{Key: game.StatEnergyRecharge, Value: 5.8},
				},
				Unactivated: []game.Substat{{Key: game.StatATKPercent, Value: 5.8}},
			},
		},
		{
			fixture: "artifact_flower_4star_1280.png",
			want: game.Artifact{
				SetKey: "GladiatorsFinale", SetName: "Gladiator's Finale", Piece: "Gladiator's Nostalgia",
				Slot: game.SlotFlower, Rarity: 4, Level: 16, MainStat: game.StatHP, Location: "KaedeharaKazuha", Lock: true,
				Substats: []game.Substat{
					{Key: game.StatATK, Value: 16}, {Key: game.StatCritRate, Value: 3.1}, {Key: game.StatDEFPercent, Value: 5.8}, {Key: game.StatHPPercent, Value: 4.2},
				},
			},
		},
		{
			// The band shifts the level and substats down, and the last substat
			// used to fall outside the box.
			fixture: "artifact_elixir_1920.png",
			want: game.Artifact{
				SetKey: "NightOfTheSkysUnveiling", SetName: "Night of the Sky's Unveiling", Piece: "Revelation's Toll",
				Slot: game.SlotSands, Rarity: 5, Level: 20, MainStat: game.StatATKPercent, Location: "Flins",
				Lock: true, AstralMark: true, ElixirCrafted: true,
				Substats: []game.Substat{
					{Key: game.StatCritRate, Value: 7.4}, {Key: game.StatCritDMG, Value: 17.9},
					{Key: game.StatDEFPercent, Value: 7.3}, {Key: game.StatElementalMastery, Value: 44},
				},
			},
		},
		{
			fixture: "artifact_wrapped_name_1280.png",
			want: game.Artifact{
				SetKey: "DisenchantmentInDeepShadow", SetName: "Disenchantment in Deep Shadow", Piece: "Sharpness That Ceased Upon Wondrous Creation",
				Slot: game.SlotPlume, Rarity: 5, Level: 4, MainStat: game.StatATK,
				Substats: []game.Substat{
					{Key: game.StatCritRate, Value: 3.1}, {Key: game.StatCritDMG, Value: 6.2},
					{Key: game.StatHPPercent, Value: 5.8}, {Key: game.StatDEF, Value: 23},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			record, skipped, err := parser.Parse(t.Context(), loadFixture(t, tt.fixture).Image)
			if err != nil {
				t.Fatal(err)
			}
			if skipped {
				t.Fatal("artifact treated as a material")
			}
			if len(record.Issues) != 0 {
				t.Errorf("issues: %v (raw %+v)", record.Issues, record.Raw)
			}
			got, want := record.Artifact, tt.want
			if !slices.Equal(got.Substats, want.Substats) {
				t.Errorf("Substats = %+v, want %+v (raw %q)", got.Substats, want.Substats, record.Raw.Substats)
			}
			if !slices.Equal(got.Unactivated, want.Unactivated) {
				t.Errorf("Unactivated = %+v, want %+v (raw %q)", got.Unactivated, want.Unactivated, record.Raw.Substats)
			}
			got.Substats, want.Substats = nil, nil
			got.Unactivated, want.Unactivated = nil, nil
			if !artifactEqual(got, want) {
				t.Errorf("Artifact = %+v\nwant       %+v\nraw %+v", got, want, record.Raw)
			}
			if !record.Exportable() {
				t.Errorf("not exportable: %v", record.Artifact.Validate())
			}
		})
	}
}

func artifactEqual(a, b game.Artifact) bool {
	return a.SetKey == b.SetKey && a.SetName == b.SetName && a.Piece == b.Piece && a.Slot == b.Slot &&
		a.Rarity == b.Rarity && a.Level == b.Level && a.MainStat == b.MainStat && a.Location == b.Location &&
		a.Lock == b.Lock && a.AstralMark == b.AstralMark && a.ElixirCrafted == b.ElixirCrafted
}

func TestRealOCRDetectsMaterials(t *testing.T) {
	parser := ArtifactParser{Layout: layout.Default().Inventory, OCR: realOCR(t), Catalog: loadCatalog(t)}

	_, skipped, err := parser.Parse(t.Context(), loadFixture(t, "material_1280.png").Image)
	if err != nil {
		t.Fatal(err)
	}
	if !skipped {
		t.Error("upgrade material was not skipped")
	}
	unrelated, err := parser.IsUnrelated(t.Context(), loadFixture(t, "artifact_circlet_1280.png").Image)
	if err != nil {
		t.Fatal(err)
	}
	if unrelated {
		t.Error("artifact reported as a material")
	}
}

func TestRealOCRItemCount(t *testing.T) {
	frame := loadFixture(t, "artifact_circlet_1280.png")
	r, err := newReader(t.Context(), realOCR(t), frame.Image)
	if err != nil {
		t.Fatal(err)
	}
	text, err := r.read(layout.Default().Inventory.ItemCount, ocr.ModeLine, "")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := parseItemCount(text); !ok || got != 123 {
		t.Errorf("parseItemCount(%q) = %d, %v; want 123", text, got, ok)
	}
}

func TestRealOCRAchievements(t *testing.T) {
	reader := AchievementReader{Layout: layout.Default().Achievements, OCR: realOCR(t)}
	cat := loadCatalog(t)
	entry := func(name string) catalog.Achievement {
		t.Helper()
		match, ok := cat.MatchAchievement(name)
		if !ok || match.Item.Name != name {
			t.Fatalf("catalog entry for %q = %+v, %v", name, match, ok)
		}
		return match.Item
	}

	tests := []struct {
		name    string
		fixture string
		entry   catalog.Achievement
		stars   int
	}{
		{"partial progress", "achievement_single_1280.png", entry("Zoo Tycoon"), 2},
		{"incomplete", "achievement_multiple_1280.png", entry("Tales of Monstrous Madness"), 0},
		{"completed stamp covers all stages", "achievement_multiple_1280.png", entry("Zoo Tycoon"), 3},
		{"near-identical name, first", "achievement_multiple_1280.png", entry("To Brave the Lightning's Glow"), 1},
		{"near-identical name, second", "achievement_multiple_1280.png", entry("Brave the Lightning's Glow..."), 1},
		{"three-line description pushes the title up", "achievement_tall_row_1280.png", entry("Urban Combat Specialist"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := reader.Classify(t.Context(), loadFixture(t, tt.fixture).Image, tt.entry, true)
			if err != nil {
				t.Fatal(err)
			}
			if !record.Verified || record.NeedsReview || record.Stars != tt.stars {
				t.Errorf("record = %+v, want verified with %d stars", record, tt.stars)
			}
			if record.Title == "" || fuzzy.Ratio(record.Title, tt.entry.Name) < 95 {
				t.Errorf("matched the wrong row: title %q for %q", record.Title, tt.entry.Name)
			}
		})
	}

	t.Run("game says no results", func(t *testing.T) {
		record, err := reader.Classify(t.Context(), loadFixture(t, "achievement_none_1280.png").Image, entry("Tales of Monstrous Madness"), false)
		if err != nil {
			t.Fatal(err)
		}
		if !record.NotFound || record.NeedsReview || record.Stars != 0 {
			t.Errorf("record = %+v, want not found", record)
		}
	})

	t.Run("offline screenshots", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"achievement_single_1280.png", "achievement_multiple_1280.png"} {
			if err := imaging.SavePNG(filepath.Join(dir, name), loadFixture(t, name).Image); err != nil {
				t.Fatal(err)
			}
		}
		records, err := ScanAchievementFrames(t.Context(), DirFrames(dir), reader, cat)
		if err != nil {
			t.Fatal(err)
		}
		byName := make(map[string]AchievementRecord)
		for _, r := range records {
			byName[r.Name] = r
		}
		if len(records) != 4 {
			t.Errorf("found %d achievements, want 4: %+v", len(records), records)
		}
		if byName["Zoo Tycoon"].Stars != 3 {
			t.Errorf("Zoo Tycoon = %+v, want the best of 2 and 3 stars", byName["Zoo Tycoon"])
		}
		if r, ok := byName["Tales of Monstrous Madness"]; !ok || r.Stars != 0 {
			t.Errorf("Tales of Monstrous Madness = %+v, %v", r, ok)
		}
	})
}

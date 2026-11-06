package export

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/erdaboss99/gensin-scan/internal/game"
)

func TestNewGOOD(t *testing.T) {
	good := game.Artifact{
		SetKey: "GladiatorsFinale", Slot: game.SlotSands, Rarity: 5, Level: 20,
		MainStat: game.StatATKPercent, Location: "HuTao", Lock: true, AstralMark: true, ElixirCrafted: true,
		Substats: []game.Substat{
			{Key: game.StatCritRate, Value: 3.9}, {Key: game.StatDEF, Value: 23}, {Key: game.StatEnergyRecharge, Value: 6.5},
		},
		Unactivated: []game.Substat{{Key: game.StatHP, Value: 299}},
	}
	bad := game.Artifact{Slot: game.SlotSands, Rarity: 5}

	doc, rejected := NewGOOD([]game.Artifact{good, bad})
	if len(rejected) != 1 || !strings.Contains(rejected[0].Error(), "artifact 1") {
		t.Errorf("rejected = %v", rejected)
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"format":"GOOD","version":2,"source":"gensin-scan","artifacts":[{"setKey":"GladiatorsFinale","slotKey":"sands","level":20,"rarity":5,"mainStatKey":"atk_","location":"HuTao","lock":true,"substats":[{"key":"critRate_","value":3.9},{"key":"def","value":23},{"key":"enerRech_","value":6.5}],"unactivatedSubstats":[{"key":"hp","value":299}],"astralMark":true,"elixirCrafted":true}]}`
	if string(raw) != want {
		t.Errorf("got  %s\nwant %s", raw, want)
	}
}

func TestNewGOODEmitsEmptyArrays(t *testing.T) {
	doc, _ := NewGOOD(nil)
	raw, _ := json.Marshal(doc)
	if !strings.Contains(string(raw), `"artifacts":[]`) {
		t.Errorf("nil artifacts must encode as []: %s", raw)
	}
	doc, _ = NewGOOD([]game.Artifact{{SetKey: "X", Slot: game.SlotFlower, Rarity: 2, MainStat: game.StatHP}})
	raw, _ = json.Marshal(doc)
	if !strings.Contains(string(raw), `"substats":[]`) {
		t.Errorf("nil substats must encode as []: %s", raw)
	}
}

func TestPaimonMoe(t *testing.T) {
	completed := []game.CompletedAchievement{{ID: 80091, Category: 0}, {ID: 80127, Category: 0}, {ID: 81000, Category: 4}}
	raw, err := PaimonMoe(completed, []byte(`{"achievement":{"9":{"1":true}},"wish-counter":{"pity":12}}`))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Achievement map[string]map[string]bool `json:"achievement"`
		Wish        map[string]int             `json:"wish-counter"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if !doc.Achievement["0"]["80127"] || !doc.Achievement["4"]["81000"] {
		t.Errorf("achievement = %v", doc.Achievement)
	}
	if doc.Achievement["9"] != nil {
		t.Error("stale achievements were kept")
	}
	if doc.Wish["pity"] != 12 {
		t.Error("unrelated data was lost")
	}
}

func TestSeelieMe(t *testing.T) {
	raw, err := SeelieMe([]game.CompletedAchievement{{ID: 5, Category: 1}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Achievements map[string]struct {
			Done bool `json:"done"`
		} `json:"achievements"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if !doc.Achievements["5"].Done {
		t.Errorf("achievements = %+v", doc.Achievements)
	}
}

func TestMergeRejectsCorruptExisting(t *testing.T) {
	if _, err := SeelieMe(nil, []byte(`[not json`)); err == nil {
		t.Error("corrupt merge input accepted")
	}
}

package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/erdaboss99/gensin-scan/internal/game"
)

func TestEmbeddedSnapshotIsUsable(t *testing.T) {
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Pieces) < 250 || len(c.Characters) < 100 || len(c.Achievements) < 1000 {
		t.Fatalf("snapshot looks truncated: %d pieces, %d characters, %d achievements",
			len(c.Pieces), len(c.Characters), len(c.Achievements))
	}

	sets := map[string]int{}
	for _, p := range c.Pieces {
		sets[p.Set]++
		if p.Slot != "" && !p.Slot.Valid() {
			t.Errorf("piece %q has invalid slot %q", p.Name, p.Slot)
		}
	}
	for set, n := range sets {
		if n != 5 && n != 1 {
			t.Errorf("set %q has %d pieces, want 5 (or 1 for Prayers sets)", set, n)
		}
	}
	for _, a := range c.Achievements {
		if a.Name == "" || len(a.IDs) == 0 {
			t.Errorf("malformed achievement %+v", a)
		}
	}
	if !slices.Contains(c.Characters, "Traveler") {
		t.Error("Traveler is missing from characters")
	}
}

// The wiki's object keys are scrambled for recent sets, while position is right.
func TestSnapshotUsesCorrectedWikiSlots(t *testing.T) {
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]game.Slot{
		"Pristine Plume of the Blessed":    game.SlotPlume,
		"Crown of the Befallen":            game.SlotCirclet,
		"Chalice of Your Blood and Sorrow": game.SlotGoblet,
		"Reckoning of the Xenogenic":       game.SlotFlower,
		"Myths of the Night Realm":         game.SlotSands,
		"Bloom Times":                      game.SlotFlower,
	}
	for name, slot := range want {
		match, ok := c.MatchPiece(name)
		if !ok || match.Item.Name != name {
			t.Errorf("piece %q not found (got %+v)", name, match)
			continue
		}
		if match.Item.Slot != slot {
			t.Errorf("piece %q slot = %q, want %q", name, match.Item.Slot, slot)
		}
	}
}

func TestMatchToleratesOCRNoise(t *testing.T) {
	c := New(Data{
		Pieces:     []Piece{{Name: "Gladiator's Nostalgia", Set: "Gladiator's Finale", Slot: game.SlotFlower}},
		Characters: []string{"Hu Tao", "Kaedehara Kazuha"},
		Achievements: []Achievement{
			{Name: "Zoo Tycoon", IDs: []game.AchievementID{1, 2, 3}},
		},
	})
	if m, ok := c.MatchPiece("Gladlator's Nostalgla"); !ok || m.Item.Set != "Gladiator's Finale" {
		t.Errorf("MatchPiece = %+v, %v", m, ok)
	}
	if m, ok := c.MatchCharacter("Kaedehara Kazuhe"); !ok || m.Item != "Kaedehara Kazuha" {
		t.Errorf("MatchCharacter = %+v, %v", m, ok)
	}
	if _, ok := c.MatchCharacter("Equipped by nobody"); ok {
		t.Error("MatchCharacter accepted unrelated text")
	}
	if _, ok := c.MatchAchievement("Zoo Keeper"); ok {
		t.Error("MatchAchievement accepted a different name")
	}
}

func TestIsAmbiguous(t *testing.T) {
	a := Achievement{Name: "Wind", IDs: []game.AchievementID{1}}
	b := Achievement{Name: "Wind Rider!", IDs: []game.AchievementID{2}}
	d := Achievement{Name: "Stone", IDs: []game.AchievementID{3}}
	c := New(Data{Achievements: []Achievement{a, b, d}})
	if !c.IsAmbiguous(a) || !c.IsAmbiguous(b) {
		t.Error("overlapping names not reported as ambiguous")
	}
	if c.IsAmbiguous(d) {
		t.Error("distinct name reported as ambiguous")
	}
}

func TestLoadPrefersDirAndFallsBack(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, charactersFile), []byte(`["OnlyMe"]`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Characters, []string{"OnlyMe"}) {
		t.Errorf("Characters = %v, want dir override", c.Characters)
	}
	if len(c.Pieces) == 0 {
		t.Error("missing file did not fall back to the snapshot")
	}

	if err := os.WriteFile(filepath.Join(dir, piecesFile), []byte(`{bad`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("corrupt file was silently accepted")
	}
}

func TestFetch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /list", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Rpc-Language") != "en-us" {
			t.Errorf("missing language header")
		}
		var body struct {
			Menu string `json:"menu_id"`
			Page int    `json:"page_num"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		list := []map[string]string{}
		total := "2"
		switch {
		case body.Menu == wikiMenuCharacters && body.Page == 1:
			list = append(list, map[string]string{"entry_page_id": "c1", "name": "Hu Tao"})
			total = "1"
		case body.Menu == wikiMenuArtifacts && body.Page == 1:
			list = append(list, map[string]string{"entry_page_id": "10", "name": "Obsidian Codex "})
		case body.Menu == wikiMenuArtifacts && body.Page == 2:
			list = append(list, map[string]string{"entry_page_id": "11", "name": "Scrambled"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"retcode": 0, "data": map[string]any{"list": list, "total": total}})
	})
	mux.HandleFunc("GET /entry", func(w http.ResponseWriter, r *http.Request) {
		var pieces string
		switch r.URL.Query().Get("entry_page_id") {
		case "10":
			pieces = `{"flower_of_life":{"title":"Myths of the Night Realm","position":"Flower of Life"},"sands_of_eon":{"title":"Myths of the Night Realm","position":"Sands of Eon"}}`
		case "11":
			pieces = `{"sands_of_eon":{"title":"A Feather","position":"Plume of Death"},"circlet_of_logos":{"title":"","position":"Circlet of Logos"}}`
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"page": map[string]any{"modules": []any{
			map[string]any{"components": []any{map[string]any{"component_id": "artifact_list", "data": pieces}}},
		}}}})
	})
	mux.HandleFunc("GET /achievements", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"5":{"name":"Second","order":2,"achievements":[{"id":50,"name":"Solo"}]},
			"0":{"name":"First","order":1,"achievements":[[{"id":1,"name":"Tiered"},{"id":2,"name":"Tiered"}]]}
		}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	data, err := Fetch(t.Context(), srv.Client(), Source{srv.URL + "/list", srv.URL + "/entry", srv.URL + "/achievements"})
	if err != nil {
		t.Fatal(err)
	}

	wantPieces := []Piece{
		{Name: "Myths of the Night Realm", Set: "Obsidian Codex", Slot: game.SlotSands},
		{Name: "Reckoning of the Xenogenic", Set: "Obsidian Codex", Slot: game.SlotFlower},
		{Name: "A Feather", Set: "Scrambled", Slot: game.SlotPlume},
	}
	if !slices.Equal(data.Pieces, wantPieces) {
		t.Errorf("Pieces = %+v, want %+v", data.Pieces, wantPieces)
	}
	if !slices.Equal(data.Characters, []string{"Hu Tao", "Traveler"}) {
		t.Errorf("Characters = %v", data.Characters)
	}
	if len(data.Achievements) != 2 || data.Achievements[0].Name != "Tiered" ||
		!slices.Equal(data.Achievements[0].IDs, []game.AchievementID{1, 2}) || data.Achievements[1].Category != 5 {
		t.Errorf("Achievements = %+v", data.Achievements)
	}

	dir := t.TempDir()
	if err := Save(dir, data); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(loaded.Pieces, wantPieces) {
		t.Errorf("Save/Load Pieces = %+v", loaded.Pieces)
	}
}

func TestFetchReportsServerErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer srv.Close()
	_, err := Fetch(t.Context(), srv.Client(), Source{srv.URL, srv.URL, srv.URL})
	if err == nil || !regexp.MustCompile(`502`).MatchString(err.Error()) {
		t.Errorf("err = %v, want a 502 error", err)
	}
}

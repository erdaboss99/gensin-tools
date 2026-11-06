package fuzzy

import "testing"

func TestRatio(t *testing.T) {
	tests := []struct {
		a, b     string
		min, max int
	}{
		{"Gladiator's Nostalgia", "gladiators nostalgia", 100, 100},
		{"Gladiator's Nostalgia", "Gladlator's Nostalgia", 90, 99},
		{"Sands of Eon", "Goblet of Eonothem", 0, 60},
		{"", "", 0, 0},
		{"abc", "", 0, 0},
		{"原神", "原神", 100, 100},
		{"Déjà Vu!", "Deja Vu!", 100, 100},
		{"DÉJÀ VU", "deja vu", 100, 100},
	}
	for _, tt := range tests {
		if got := Ratio(tt.a, tt.b); got < tt.min || got > tt.max {
			t.Errorf("Ratio(%q, %q) = %d, want %d..%d", tt.a, tt.b, got, tt.min, tt.max)
		}
	}
}

func TestBest(t *testing.T) {
	identity := func(s string) string { return s }
	candidates := []string{"CRIT Rate", "CRIT DMG", "Energy Recharge"}

	match, ok := Best("CR1T Rate", candidates, identity, 70)
	if !ok || match.Item != "CRIT Rate" {
		t.Errorf("Best = %+v, %v; want CRIT Rate", match, ok)
	}
	if _, ok := Best("completely different", candidates, identity, 70); ok {
		t.Error("Best accepted a match under the threshold")
	}
	if _, ok := Best("", candidates, identity, 0); ok {
		t.Error("Best accepted an empty query")
	}
	if _, ok := Best("x", nil, identity, 0); ok {
		t.Error("Best matched an empty candidate list")
	}
}

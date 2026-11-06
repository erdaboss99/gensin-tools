package export

import (
	"fmt"

	"github.com/erdaboss99/gensin-scan/internal/game"
)

const (
	goodFormat = "GOOD"
	goodSource = "gensin-scan"
	// goodVersion is the GOOD format version emitted. Version 2 has the widest tool
	// support, and tools ignore fields they don't recognize.
	goodVersion = 2
)

// GOOD is the Genshin Open Object Description document, restricted to the artifact
// data this project scans.
type GOOD struct {
	Format    string         `json:"format"`
	Version   int            `json:"version"`
	Source    string         `json:"source"`
	Artifacts []GOODArtifact `json:"artifacts"`
}

// GOODArtifact is one artifact in GOOD form.
type GOODArtifact struct {
	SetKey      game.SetKey       `json:"setKey"`
	SlotKey     game.Slot         `json:"slotKey"`
	Level       game.Level        `json:"level"`
	Rarity      game.Rarity       `json:"rarity"`
	MainStatKey game.Stat         `json:"mainStatKey"`
	Location    game.CharacterKey `json:"location"`
	Lock        bool              `json:"lock"`
	Substats    []game.Substat    `json:"substats"`
	Unactivated []game.Substat    `json:"unactivatedSubstats,omitempty"`
	AstralMark  bool              `json:"astralMark,omitempty"`
	Elixir      bool              `json:"elixirCrafted,omitempty"`
}

// NewGOOD builds a GOOD document. Artifacts failing validation are left out, since
// importers would silently replace bad fields with defaults.
func NewGOOD(artifacts []game.Artifact) (GOOD, []error) {
	doc := GOOD{Format: goodFormat, Version: goodVersion, Source: goodSource, Artifacts: []GOODArtifact{}}
	var rejected []error
	for i, a := range artifacts {
		if err := a.Validate(); err != nil {
			rejected = append(rejected, fmt.Errorf("artifact %d: %w", i, err))
			continue
		}
		doc.Artifacts = append(doc.Artifacts, newGOODArtifact(a))
	}
	return doc, rejected
}

func newGOODArtifact(a game.Artifact) GOODArtifact {
	substats := a.Substats
	if substats == nil {
		substats = []game.Substat{}
	}
	return GOODArtifact{
		SetKey:      a.SetKey,
		SlotKey:     a.Slot,
		Level:       a.Level,
		Rarity:      a.Rarity,
		MainStatKey: a.MainStat,
		Location:    a.Location,
		Lock:        a.Lock,
		Substats:    substats,
		Unactivated: a.Unactivated,
		AstralMark:  a.AstralMark,
		Elixir:      a.ElixirCrafted,
	}
}

package catalog

import (
	"slices"
	"sync"

	"github.com/erdaboss99/gensin-scan/internal/game"
)

// Piece is one artifact piece.
type Piece struct {
	Name string `json:"name"`
	Set  string `json:"set"`
	// Slot is empty when the data source is unreliable for this piece.
	Slot game.Slot `json:"slot,omitempty"`
}

// Achievement is one achievement, which has one ID per star.
type Achievement struct {
	Name         string               `json:"name"`
	Category     game.CategoryID      `json:"category"`
	CategoryName string               `json:"categoryName"`
	IDs          []game.AchievementID `json:"ids"`
}

// Data is the raw reference data.
type Data struct {
	Pieces       []Piece
	Characters   []string
	Achievements []Achievement
}

// pieceAliases are alternative names for pieces whose wiki name differs from the
// game's data files by too much for fuzzy matching to bridge.
var pieceAliases = []Piece{
	{Name: "Poetry of Days Past", Set: "Song of Days Past", Slot: game.SlotCirclet},
	{Name: "Mystic's Gold Dial", Set: "Scroll of the Hero of Cinder City", Slot: game.SlotSands},
}

// Catalog answers lookups over Data.
type Catalog struct {
	Data
	matchablePieces []Piece
	ambiguous       func() nameSet
}

// nameSet is a set of normalized names.
type nameSet map[string]bool

// New wraps data in a Catalog.
func New(data Data) *Catalog {
	c := &Catalog{Data: data, matchablePieces: slices.Concat(data.Pieces, pieceAliases)}
	c.ambiguous = sync.OnceValue(func() nameSet { return findAmbiguous(c.Achievements) })
	return c
}

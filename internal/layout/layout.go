package layout

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Layout is the complete description of the screens that get scanned.
type Layout struct {
	Inventory    Inventory    `json:"inventory"`
	Achievements Achievements `json:"achievements"`
}

// Default returns the built-in layout.
func Default() Layout {
	return Layout{
		Inventory: Inventory{
			Grid: Grid{
				Columns:           8,
				First:             Point{120, 131},
				DeltaX:            98,
				DeltaY:            117,
				VerticalPadding:   160,
				LastRowY:          -183,
				ScrollTicksPerRow: 9.734,
			},
			ItemCount:   Box{-250, 20, -75, 45},
			Panel:       Box{-408, 80, -81, -80},
			Name:        Box{-393, 81, -97, 116},
			Slot:        Box{-391, 120, -221, 144},
			MainStat:    Box{-391, 180, -221, 200},
			Rarity:      Box{-392, 236, -278, 260},
			Level:       Box{-387, 287, -353, 304},
			Substats:    Box{-378, 315, -82, 420},
			Equipped:    Box{-272, -110, -82, -85},
			LockIcon:    Box{-159, 283, -155, 287},
			AstralIcon:  Box{-121, 283, -117, 287},
			ElixirBand:  Box{-347, 273, -147, 277},
			ElixirShift: 25,
			StarMinArea: 85,
			StarMaxArea: 180,
		},
		Achievements: Achievements{
			ClearSearch:    Point{300, 90},
			CompletedCount: Box{-343, 20, -93, 44},
			Results:        Box{492, 100, 1212, -60},
			NoResults:      Box{490, 385, 1220, 470},
			Medals:         Box{501, 163, 567, -60},
			MedalMinHeight: 50,
			Title:          RowBox{Left: 575, Right: 1040, Top: -36, Bottom: -1},
			Stars:          RowBox{Left: 520, Right: 560, Top: -24, Bottom: 22},
			Status:         RowBox{Left: 1117, Right: 1210, Top: -12, Bottom: 12},
			MaxRows:        6,
			StarMinSize:    8,
			StarMaxSize:    16,
		},
	}
}

// Load reads a layout override from a JSON file. Fields the file omits keep their
// default value, and misspelled fields are an error rather than silently ignored.
func Load(path string) (Layout, error) {
	l := Default()
	f, err := os.Open(path)
	if err != nil {
		return l, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&l); err != nil {
		return l, fmt.Errorf("parse layout %s: %w", path, err)
	}
	if err := l.Validate(); err != nil {
		return l, fmt.Errorf("layout %s: %w", path, err)
	}
	return l, nil
}

// Validate rejects values that would make the scanners divide by zero or loop forever.
func (l Layout) Validate() error {
	var errs []error
	g := l.Inventory.Grid
	if g.Columns <= 0 || g.DeltaX <= 0 || g.DeltaY <= 0 || g.ScrollTicksPerRow <= 0 {
		errs = append(errs, errors.New("inventory.grid: columns, deltaX, deltaY and scrollTicksPerRow must be positive"))
	}
	if l.Inventory.StarMinArea > l.Inventory.StarMaxArea {
		errs = append(errs, errors.New("inventory: starMinArea exceeds starMaxArea"))
	}
	a := l.Achievements
	if a.MaxRows <= 0 || a.MedalMinHeight <= 0 {
		errs = append(errs, errors.New("achievements: maxRows and medalMinHeight must be positive"))
	}
	if a.StarMinSize > a.StarMaxSize {
		errs = append(errs, errors.New("achievements: starMinSize exceeds starMaxSize"))
	}
	return errors.Join(errs...)
}

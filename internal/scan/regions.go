package scan

import (
	"fmt"
	"image"

	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
)

// NamedRect is a labelled screen region, used for calibration output.
type NamedRect struct {
	Name string
	Rect image.Rectangle
}

// InventoryRegions lists the inventory regions for this frame, including the shift
// caused by a Sanctifying Elixir band.
func InventoryRegions(l layout.Inventory, win layout.Window, frame *image.RGBA) []NamedRect {
	shift := 0
	if isElixirBand(imaging.MeanColor(frame, win.Rect(l.ElixirBand))) {
		shift = l.ElixirShift
	}
	return []NamedRect{
		{"itemCount", win.Rect(l.ItemCount)},
		{"name", win.Rect(l.Name)},
		{"slot", win.Rect(l.Slot)},
		{"mainStat", win.Rect(l.MainStat)},
		{"rarity", win.Rect(l.Rarity)},
		{"level", win.Rect(l.Level.Shifted(shift))},
		{"substats", win.Rect(l.Substats.Shifted(shift))},
		{"equipped", win.Rect(l.Equipped)},
		{"lockIcon", win.Rect(l.LockIcon.Shifted(shift))},
		{"astralIcon", win.Rect(l.AstralIcon.Shifted(shift))},
	}
}

// AchievementRegions lists the regions of every result row found in this frame.
func AchievementRegions(l layout.Achievements, win layout.Window, frame *image.RGBA) []NamedRect {
	var regions []NamedRect
	for row, center := range findRows(l, win, frame) {
		regions = append(regions,
			NamedRect{fmt.Sprintf("title[%d]", row), win.RowRect(l.Title, center)},
			NamedRect{fmt.Sprintf("stars[%d]", row), win.RowRect(l.Stars, center)},
			NamedRect{fmt.Sprintf("status[%d]", row), win.RowRect(l.Status, center)},
		)
	}
	if len(regions) == 0 {
		regions = append(regions, NamedRect{"noResults", win.Rect(l.NoResults)})
	}
	return append(regions, NamedRect{"completed", win.Rect(l.CompletedCount)})
}

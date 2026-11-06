package scan

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"slices"
	"testing"

	"github.com/erdaboss99/gensin-scan/control"
	"github.com/erdaboss99/gensin-scan/internal/layout"
)

func TestParseItemCount(t *testing.T) {
	tests := []struct {
		text string
		want int
		ok   bool
	}{
		{"Artifacts 1234/2100", 1234, true},
		{"1234/2100", 1234, true},
		{"1234 / 2100", 1234, true},
		{"12O4/2100", 1204, true},
		{"12&4/2100", 1284, true},
		{"", 0, false},
		{"Artifacts", 0, false},
		{"0/2100", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseItemCount(tt.text)
		if got != tt.want || ok != tt.ok {
			t.Errorf("parseItemCount(%q) = %d, %v; want %d, %v", tt.text, got, ok, tt.want, tt.ok)
		}
	}
}

// simulatedInventory models the inventory grid well enough to check the traversal:
// the mouse wheel moves the list, clamped at its end, and a click selects whatever
// is under the cursor.
type simulatedInventory struct {
	width, height int
	grid          layout.Grid
	panel         layout.Box
	artifacts     int
	materials     int
	failAfter     int
	stuckClicks   bool
	// reuseFrame shares one image between captures, for tests that consume each
	// frame before capturing the next.
	reuseFrame bool
	frame      *image.RGBA

	ticks   int
	clicks  int
	current int
	visited []int
}

func (s *simulatedInventory) win() layout.Window {
	win, err := layout.NewWindow(s.width, s.height)
	if err != nil {
		panic(err)
	}
	return win
}

func (s *simulatedInventory) totalRows() int {
	return (s.artifacts+s.materials-1)/s.grid.Columns + 1
}

func (s *simulatedInventory) topRow() (row int, atBottom bool) {
	win := s.win()
	limit := max(s.totalRows()-s.grid.RowsPerPage(win), 0)
	row = int(float64(-s.ticks)/s.grid.ScrollTicksPerRow + 0.5)
	if row >= limit {
		return limit, true
	}
	return row, false
}

func (s *simulatedInventory) Size() (int, int) { return s.width, s.height }

func (s *simulatedInventory) MoveTo(int, int) error { return nil }

func (s *simulatedInventory) Scroll(ticks int) error {
	s.ticks += ticks
	if s.ticks > 0 {
		s.ticks = 0
	}
	return nil
}

func (s *simulatedInventory) Click(x, y int) error {
	s.clicks++
	if s.failAfter > 0 && s.clicks > s.failAfter {
		return control.ErrFocusLost
	}
	if s.stuckClicks {
		return nil
	}
	win := s.win()
	col := (x - win.X(s.grid.First.X) + win.Len(s.grid.DeltaX)/2) / win.Len(s.grid.DeltaX)
	var row int
	if top, atBottom := s.topRow(); atBottom && s.totalRows() > s.grid.RowsPerPage(win) {
		fromBottom := (win.Y(s.grid.LastRowY) - y + win.Len(s.grid.DeltaY)/2) / win.Len(s.grid.DeltaY)
		row = s.totalRows() - 1 - fromBottom
	} else {
		row = top + (y-win.Y(s.grid.First.Y)+win.Len(s.grid.DeltaY)/2)/win.Len(s.grid.DeltaY)
	}
	s.current = row*s.grid.Columns + col
	return nil
}

func (s *simulatedInventory) Capture() (*image.RGBA, error) {
	img := s.frame
	if img == nil || !s.reuseFrame {
		img = image.NewRGBA(image.Rect(0, 0, s.width, s.height))
		s.frame = img
	}
	id := s.current + 1
	item := color.RGBA{uint8(id >> 8), uint8(id), 7, 255}
	draw.Draw(img, s.win().Rect(s.panel), image.NewUniform(item), image.Point{}, draw.Src)
	return img, nil
}

func (s *simulatedInventory) Type(string) error { return nil }

func (s *simulatedInventory) PressEnter() error { return nil }

func (s *simulatedInventory) itemIn(frame *image.RGBA) int {
	at := s.win().Rect(s.panel).Min
	c := frame.RGBAAt(at.X+1, at.Y+1)
	return int(c.R)<<8 | int(c.G) - 1
}

func (s *simulatedInventory) walker() InventoryWalker {
	return InventoryWalker{
		Game:   s,
		Layout: layout.Inventory{Grid: s.grid, Panel: s.panel},
		Count:  s.artifacts,
		IsMaterial: func(_ context.Context, frame *image.RGBA) (bool, error) {
			return s.itemIn(frame) >= s.artifacts, nil
		},
	}
}

func TestInventoryWalkVisitsEveryArtifactExactlyOnce(t *testing.T) {
	type config struct {
		size                 [2]int
		artifacts, materials int
	}
	var configs []config
	for _, size := range [][2]int{{1280, 720}, {1280, 800}} {
		for _, materials := range []int{0, 3, 9} {
			for _, artifacts := range []int{1, 8, 9, 32, 33, 40, 41, 48, 49, 57, 65, 121} {
				configs = append(configs, config{size, artifacts, materials})
			}
		}
	}
	configs = append(configs, config{[2]int{1280, 720}, 1800, 3}, config{[2]int{1920, 1080}, 1801, 2})

	for _, c := range configs {
		sim := &simulatedInventory{
			width: c.size[0], height: c.size[1],
			grid:      layout.Default().Inventory.Grid,
			panel:     layout.Default().Inventory.Panel,
			artifacts: c.artifacts,
			materials: c.materials,

			reuseFrame: true,
		}
		seen := make(map[int]int)
		err := sim.walker().Walk(t.Context(), func(f Frame) error {
			seen[sim.itemIn(f.Image)]++
			if f.Duplicate {
				t.Errorf("%+v: frame %d flagged duplicate", c, f.Index)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%+v: %v", c, err)
		}
		if len(seen) != c.artifacts {
			t.Errorf("%+v: saw %d distinct items", c, len(seen))
		}
		for item, times := range seen {
			if item >= c.artifacts {
				t.Errorf("%+v: captured non-artifact %d", c, item)
			}
			if times != 1 {
				t.Errorf("%+v: item %d captured %d times", c, item, times)
			}
		}
	}
}

// The real inventory that exposed this had 384 artifacts and 364 materials, with the
// on-screen count of 748 including both.
func TestInventoryWalkEndsAtMaterialsCountedInTotal(t *testing.T) {
	for _, c := range []struct{ artifacts, materials int }{{384, 364}, {40, 32}, {8, 40}, {100, 100}} {
		sim := &simulatedInventory{
			width: 1280, height: 720,
			grid:      layout.Default().Inventory.Grid,
			panel:     layout.Default().Inventory.Panel,
			artifacts: c.artifacts,
			materials: c.materials,
		}
		w := sim.walker()
		w.Count = c.artifacts + c.materials
		seen := make(map[int]bool)
		err := w.Walk(t.Context(), func(f Frame) error {
			seen[sim.itemIn(f.Image)] = true
			return nil
		})
		if err != nil {
			t.Fatalf("%+v: %v", c, err)
		}
		for item := range c.artifacts {
			if !seen[item] {
				t.Errorf("%+v: artifact %d was missed", c, item)
			}
		}
	}
}

func TestInventoryWalkHonorsLimit(t *testing.T) {
	sim := &simulatedInventory{width: 1280, height: 720, grid: layout.Default().Inventory.Grid, panel: layout.Default().Inventory.Panel, artifacts: 500}
	w := sim.walker()
	w.Limit = 10
	frames := 0
	if err := w.Walk(t.Context(), func(Frame) error { frames++; return nil }); err != nil {
		t.Fatal(err)
	}
	if frames != 10 {
		t.Errorf("frames = %d, want 10", frames)
	}
}

func TestInventoryWalkFlagsStuckClicks(t *testing.T) {
	sim := &simulatedInventory{
		width: 1280, height: 720, grid: layout.Default().Inventory.Grid, panel: layout.Default().Inventory.Panel,
		artifacts: 3, stuckClicks: true,
	}
	var flags []bool
	if err := sim.walker().Walk(t.Context(), func(f Frame) error { flags = append(flags, f.Duplicate); return nil }); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(flags, []bool{false, true, true}) {
		t.Errorf("duplicate flags = %v", flags)
	}
}

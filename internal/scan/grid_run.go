package scan

import (
	"context"
	"errors"
	"fmt"
	"image"
	"math"
	"time"

	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
)

// gridRun is the state of one traversal.
type gridRun struct {
	walker   InventoryWalker
	ctx      context.Context
	win      layout.Window
	emit     EmitFrame
	total    int
	captured int
	scrolled int
	lastHash uint64
}

var errWalkDone = errors.New("all items captured")

const (
	// settleAttempts limits how many times captureSettled looks again.
	settleAttempts = 6

	cursorSettleDelay = 80 * time.Millisecond
	bottomScrollDelay = 200 * time.Millisecond
	scrollDelay       = 100 * time.Millisecond
	clickDelay        = 100 * time.Millisecond
	duplicateDelay    = 250 * time.Millisecond
	settlePollDelay   = 40 * time.Millisecond

	// bottomScrollExtraRows overshoots the list's end, which the game clamps.
	bottomScrollExtraRows = 2

	liveSourceFormat = "live:%d"
)

// walk visits items page by page. The grid only scrolls in whole-row steps while
// the list is long enough, and the last rows are only reachable near the bottom,
// so the traversal has three phases: full pages, a partial page to reach the
// second-to-last row, and the bottom row anchored to the end of the list.
func (g *gridRun) walk() error {
	if g.total <= 0 {
		return nil
	}
	grid := g.walker.Layout.Grid
	rowsPerPage := grid.RowsPerPage(g.win)
	rows := (g.total-1)/grid.Columns + 1
	pages := max(1, (rows-1)/rowsPerPage)

	if err := g.walker.Game.MoveTo(g.cell(0, 0).X, g.cell(0, 0).Y); err != nil {
		return err
	}
	if err := g.wait(cursorSettleDelay); err != nil {
		return err
	}

	err := g.visitPages(pages, rowsPerPage)
	if err == nil {
		lastPageRow := (pages - 1) * rowsPerPage
		err = g.visitPartialPage(rows, rowsPerPage, lastPageRow)
	}
	if err == nil {
		err = g.visitBottom()
	}
	if errors.Is(err, errWalkDone) {
		return nil
	}
	return err
}

func (g *gridRun) visitPages(pages, rowsPerPage int) error {
	columns := g.walker.Layout.Grid.Columns
	for page := range pages {
		if err := g.scrollToRow(page * rowsPerPage); err != nil {
			return err
		}
		for row := range rowsPerPage {
			for col := range columns {
				if err := g.visit(g.cell(row, col)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *gridRun) visitPartialPage(rows, rowsPerPage, currentRow int) error {
	target := rows - rowsPerPage - 1
	if target <= currentRow {
		return nil
	}
	if err := g.scrollToRow(target); err != nil {
		return err
	}
	for row := currentRow + rowsPerPage - target; row < rowsPerPage; row++ {
		for col := range g.walker.Layout.Grid.Columns {
			if err := g.visit(g.cell(row, col)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *gridRun) visitBottom() error {
	grid := g.walker.Layout.Grid
	if err := g.scrollToRow(g.total/grid.Columns + bottomScrollExtraRows); err != nil {
		return err
	}
	if err := g.wait(bottomScrollDelay); err != nil {
		return err
	}
	materialRows, found, err := g.countMaterialRows()
	if err != nil {
		return err
	}
	if !found {
		// The item count can include upgrade materials, and these sort after the
		// artifacts, so nothing is left to read.
		if g.walker.Log != nil {
			g.walker.Log.Info("reached the upgrade materials at the end of the inventory", "skipped", g.total-g.captured)
		}
		return nil
	}
	for col := range g.total - g.captured {
		if err := g.visit(g.bottomCell(materialRows, col)); err != nil {
			return err
		}
	}
	return nil
}

// countMaterialRows finds how many rows of upgrade materials sit below the
// artifacts. They sort after every artifact, so the walk checks upward from the
// bottom until it sees a real artifact. found is false when the whole screen holds
// materials.
func (g *gridRun) countMaterialRows() (rows int, found bool, err error) {
	maxRows := g.walker.Layout.Grid.RowsPerPage(g.win)
	for rows := range maxRows {
		at := g.bottomCell(rows, 0)
		if err := g.click(at); err != nil {
			return 0, false, err
		}
		frame, err := g.walker.Game.Capture()
		if err != nil {
			return 0, false, err
		}
		unrelated, err := g.walker.IsMaterial(g.ctx, frame)
		if err != nil {
			return 0, false, err
		}
		if !unrelated {
			return rows, true, nil
		}
	}
	return 0, false, nil
}

func (g *gridRun) cell(row, col int) image.Point {
	grid := g.walker.Layout.Grid
	first := g.win.Point(grid.First)
	return image.Pt(first.X+g.win.Len(grid.DeltaX)*col, first.Y+g.win.Len(grid.DeltaY)*row)
}

func (g *gridRun) bottomCell(rowsFromBottom, col int) image.Point {
	grid := g.walker.Layout.Grid
	x := g.win.Point(grid.First).X + g.win.Len(grid.DeltaX)*col
	y := g.win.Y(grid.LastRowY) - g.win.Len(grid.DeltaY)*rowsFromBottom
	return image.Pt(x, y)
}

func (g *gridRun) scrollToRow(row int) error {
	ticks := int(math.Round(float64(row) * g.walker.Layout.Grid.ScrollTicksPerRow))
	if err := g.walker.Game.Scroll(g.scrolled - ticks); err != nil {
		return err
	}
	g.scrolled = ticks
	return g.wait(scrollDelay)
}

func (g *gridRun) click(at image.Point) error {
	if err := g.walker.Game.Click(at.X, at.Y); err != nil {
		return err
	}
	return g.wait(clickDelay)
}

// visit selects an item and emits a frame of it. An unchanged panel gets one
// retry, in case the game was slow to redraw.
func (g *gridRun) visit(at image.Point) error {
	if err := g.click(at); err != nil {
		return err
	}
	frame, hash, err := g.captureSettled()
	if err != nil {
		return err
	}
	duplicate := g.captured > 0 && hash == g.lastHash
	if duplicate {
		if err := g.wait(duplicateDelay); err != nil {
			return err
		}
		if frame, hash, err = g.captureSettled(); err != nil {
			return err
		}
		duplicate = hash == g.lastHash
	}
	g.lastHash = hash

	err = g.emit(Frame{Index: g.captured, Source: fmt.Sprintf(liveSourceFormat, g.captured), Image: frame, Duplicate: duplicate})
	if err != nil {
		return err
	}
	g.captured++
	if g.captured >= g.total {
		return errWalkDone
	}
	return nil
}

// captureSettled waits for the panel to stop changing, as it fades between items
// for longer than a click's fixed delay. It gives up after a few tries and uses the
// latest frame, in case something on the panel animates continuously.
func (g *gridRun) captureSettled() (*image.RGBA, uint64, error) {
	panel := g.win.Rect(g.walker.Layout.Panel)
	frame, err := g.walker.Game.Capture()
	if err != nil {
		return nil, 0, err
	}
	hash := imaging.Fingerprint(frame, panel)
	for range settleAttempts {
		if err := g.wait(settlePollDelay); err != nil {
			return nil, 0, err
		}
		next, err := g.walker.Game.Capture()
		if err != nil {
			return nil, 0, err
		}
		nextHash := imaging.Fingerprint(next, panel)
		settled := nextHash == hash
		frame, hash = next, nextHash
		if settled {
			break
		}
	}
	return frame, hash, nil
}

func (g *gridRun) wait(d time.Duration) error {
	return g.walker.Pace.sleep(g.ctx, d)
}

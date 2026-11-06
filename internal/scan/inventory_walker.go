package scan

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/erdaboss99/gensin-scan/control"
	"github.com/erdaboss99/gensin-scan/internal/layout"
	"github.com/erdaboss99/gensin-scan/ocr"
)

// MaterialCheck reports whether a frame shows an upgrade material, not an artifact.
type MaterialCheck func(ctx context.Context, frame *image.RGBA) (bool, error)

// InventoryWalker clicks through the artifact grid and captures each item.
type InventoryWalker struct {
	Game   control.Game
	Layout layout.Inventory
	OCR    ocr.Recognizer
	// IsMaterial reports whether a frame shows an upgrade material.
	IsMaterial MaterialCheck
	Pace       Pace
	// Limit stops after this many items. Zero means all of them.
	Limit int
	// Count overrides the item count read from the screen. Zero reads it.
	Count int
	Log   *slog.Logger
}

// Walk implements FrameSource.
func (w InventoryWalker) Walk(ctx context.Context, emit EmitFrame) error {
	width, height := w.Game.Size()
	win, err := layout.NewWindow(width, height)
	if err != nil {
		return err
	}
	total := w.Count
	if total == 0 {
		if total, err = w.readItemCount(ctx); err != nil {
			return err
		}
	}
	if w.Limit > 0 {
		total = min(total, w.Limit)
	}
	if w.Log != nil {
		w.Log.Info("walking inventory", "items", total)
	}

	run := gridRun{walker: w, ctx: ctx, win: win, emit: emit, total: total}
	return run.walk()
}

func (w InventoryWalker) readItemCount(ctx context.Context) (int, error) {
	frame, err := w.Game.Capture()
	if err != nil {
		return 0, err
	}
	r, err := newReader(ctx, w.OCR, frame)
	if err != nil {
		return 0, err
	}
	text, err := r.read(w.Layout.ItemCount, ocr.ModeLine, "")
	if err != nil {
		return 0, err
	}
	count, ok := parseItemCount(text)
	if !ok {
		return 0, fmt.Errorf("cannot read the item count from %q: open the Artifacts tab of your inventory, run calibrate, or pass --count", text)
	}
	return count, nil
}

var (
	fractionPattern     = regexp.MustCompile(`(\d+)\s*/\s*\d+`)
	itemCountDigitFixes = strings.NewReplacer("O", "0", "o", "0", "A", "4", "&", "8")
)

// parseItemCount reads the first number of text such as "Artifacts 1234/2100".
func parseItemCount(text string) (int, bool) {
	text = itemCountDigitFixes.Replace(text)
	m := fractionPattern.FindStringSubmatch(text)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil && n > 0
}

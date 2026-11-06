package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"github.com/erdaboss99/gensin-scan/control"
	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
	"github.com/erdaboss99/gensin-scan/internal/scan"
	"github.com/erdaboss99/gensin-scan/ocr"
)

// screen names the game screen being calibrated.
type screen string

const (
	screenInventory    screen = "inventory"
	screenAchievements screen = "achievements"
)

// Set implements flag.Value.
func (s *screen) Set(value string) error {
	switch screen(value) {
	case screenInventory, screenAchievements:
		*s = screen(value)
		return nil
	}
	return fmt.Errorf("must be %q or %q", screenInventory, screenAchievements)
}

// String implements flag.Value.
func (s *screen) String() string { return string(*s) }

var _ flag.Value = (*screen)(nil)

// calibrateOptions holds the calibrate command's flags.
type calibrateOptions struct {
	engine      engineFlags
	which       screen
	image       string
	out         string
	printLayout bool
}

func (o *calibrateOptions) register(flags *flag.FlagSet) {
	o.engine.register(flags)
	o.which = screenInventory
	flags.Var(&o.which, "screen", `screen to check: "inventory" or "achievements"`)
	flags.StringVar(&o.image, "image", "", "use this screenshot instead of the live game")
	flags.StringVar(&o.out, "o", "calibration.png", "image to write with the regions drawn on it")
	flags.BoolVar(&o.printLayout, "print-layout", false, "print the active layout as JSON, as a starting point for --layout, and exit")
}

var (
	// regionPalette colours calibration boxes so neighbours stay distinguishable.
	regionPalette = []color.RGBA{
		{255, 0, 80, 255}, {0, 200, 255, 255}, {120, 255, 0, 255}, {255, 160, 0, 255},
		{200, 80, 255, 255}, {255, 255, 0, 255}, {0, 255, 170, 255}, {255, 120, 200, 255},
	}
	// clickMarkColor marks where the scanner will click.
	clickMarkColor = color.RGBA{255, 0, 80, 255}
)

func (a *app) runCalibrate(ctx context.Context, args []string) error {
	var opts calibrateOptions
	flags := a.newFlagSet("calibrate")
	opts.register(flags)
	if err := parseFlags(flags, args); err != nil {
		return err
	}

	lay, err := opts.engine.loadLayout()
	if err != nil {
		return err
	}
	if opts.printLayout {
		raw, err := json.MarshalIndent(lay, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(a.stdout, string(raw))
		return nil
	}

	t, err := opts.engine.newTesseract()
	if err != nil {
		return err
	}
	frame, err := a.frameFor(ctx, opts.image)
	if err != nil {
		return err
	}
	bounds := frame.Bounds()
	win, err := layout.NewWindow(bounds.Dx(), bounds.Dy())
	if err != nil {
		return err
	}

	var regions []scan.NamedRect
	if opts.which == screenAchievements {
		regions = scan.AchievementRegions(lay.Achievements, win, frame)
		if len(regions) == 0 {
			fmt.Fprintln(a.stdout, "No result rows found. If a search result is showing, check achievements.medals in the layout.")
		}
	} else {
		regions = scan.InventoryRegions(lay.Inventory, win, frame)
	}

	canvas := image.NewRGBA(bounds)
	draw.Draw(canvas, bounds, frame, bounds.Min, draw.Src)
	if err := a.drawRegions(ctx, t, frame, canvas, win, regions); err != nil {
		return err
	}
	if opts.which == screenInventory {
		drawGrid(canvas, win, lay.Inventory.Grid)
	}

	if err := imaging.SavePNG(opts.out, canvas); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "\nRegions drawn on %s. Each box should tightly contain the matching text.\n", opts.out)
	fmt.Fprintln(a.stdout, `Adjust boxes with "calibrate --print-layout > layout.json", then pass --layout layout.json.`)
	return nil
}

// drawRegions outlines each region on canvas and prints what OCR reads inside it.
func (a *app) drawRegions(ctx context.Context, rec ocr.Recognizer, frame, canvas *image.RGBA, win layout.Window, regions []scan.NamedRect) error {
	thickness := max(win.Len(1), 1)
	scale := imaging.OCRScale(win.Width)
	for i, region := range regions {
		imaging.Outline(canvas, region.Rect.Inset(-thickness), regionPalette[i%len(regionPalette)], thickness)
		text, err := rec.Recognize(ctx, ocr.Request{
			Image: imaging.PrepareForOCR(frame, region.Rect, scale),
			Mode:  regionOCRMode(region.Name),
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "%-12s %-24s %q\n", region.Name, region.Rect, strings.ReplaceAll(text, "\n", " | "))
	}
	return nil
}

func regionOCRMode(name string) ocr.Mode {
	if name == "substats" || name == "name" {
		return ocr.ModeBlock
	}
	return ocr.ModeLine
}

// drawGrid marks where the scanner will click on the first screen of items.
func drawGrid(canvas *image.RGBA, win layout.Window, grid layout.Grid) {
	first := win.Point(grid.First)
	size := max(win.Len(4), 2)
	for row := range grid.RowsPerPage(win) {
		for col := range grid.Columns {
			x := first.X + win.Len(grid.DeltaX)*col
			y := first.Y + win.Len(grid.DeltaY)*row
			mark := image.Rect(x-size, y-size, x+size, y+size)
			draw.Draw(canvas, mark, image.NewUniform(clickMarkColor), image.Point{}, draw.Src)
		}
	}
}

func (a *app) frameFor(ctx context.Context, path string) (*image.RGBA, error) {
	if path != "" {
		return imaging.Load(path)
	}
	game, err := control.Attach(ctx, control.AttachOptions{})
	if err != nil {
		return nil, err
	}
	return game.Capture()
}

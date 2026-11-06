package cli

import (
	"context"
	"flag"
	"fmt"
	"runtime"
	"slices"
	"time"

	"github.com/erdaboss99/gensin-scan/internal/scan"
)

// keepOnly filters artifact records in place by minimum rarity. Records with
// unknown rarity are kept, because discarding them would hide scan failures.
func keepOnly(records []scan.ArtifactRecord, minRarity int) []scan.ArtifactRecord {
	return slices.DeleteFunc(records, func(r scan.ArtifactRecord) bool {
		return r.Artifact.Rarity != 0 && int(r.Artifact.Rarity) < minRarity
	})
}

func defaultWorkers() int {
	return max(runtime.NumCPU()-1, 1)
}

const defaultCountdown = 5 * time.Second

// artifactOptions holds the artifacts command's flags.
type artifactOptions struct {
	engine     engineFlags
	out        string
	fromDir    string
	saveFrames string
	workers    int
	limit      int
	count      int
	minRarity  int
	traveler   string
	pace       float64
	countdown  time.Duration
}

func (o *artifactOptions) register(flags *flag.FlagSet) {
	o.engine.register(flags)
	flags.StringVar(&o.out, "o", "artifacts.json", "output report file")
	flags.StringVar(&o.fromDir, "from-dir", "", "scan the screenshots in this directory instead of the live game")
	flags.StringVar(&o.saveFrames, "save-frames", "", "also save each captured frame to this directory, for re-scanning with --from-dir")
	flags.IntVar(&o.workers, "workers", defaultWorkers(), "number of parallel OCR processes")
	flags.IntVar(&o.limit, "limit", 0, "stop after this many items (0 scans everything)")
	flags.IntVar(&o.count, "count", 0, "number of items in the inventory (0 reads it from the screen)")
	flags.IntVar(&o.minRarity, "min-rarity", 4, "leave artifacts below this rarity out of the report")
	flags.StringVar(&o.traveler, "traveler-name", "", "your in-game name, shown instead of \"Traveler\" on equipped artifacts")
	flags.Float64Var(&o.pace, "pace", 1, "multiplier for delays between actions; raise it if clicks get missed")
	flags.DurationVar(&o.countdown, "countdown", defaultCountdown, "delay before a live scan starts")
}

func (a *app) runArtifacts(ctx context.Context, args []string) error {
	var opts artifactOptions
	flags := a.newFlagSet("artifacts")
	opts.register(flags)
	if err := parseFlags(flags, args); err != nil {
		return err
	}

	engine, err := opts.engine.load(ctx)
	if err != nil {
		return err
	}
	parser := scan.ArtifactParser{
		Layout:       engine.layout.Inventory,
		OCR:          engine.ocr,
		Catalog:      engine.catalog,
		TravelerName: opts.traveler,
	}
	source, err := a.artifactSource(ctx, opts, engine, parser)
	if err != nil {
		return err
	}

	records, scanErr := scan.ScanArtifacts(ctx, source, parser, scan.ScanOptions{
		Workers:       opts.workers,
		SaveFramesDir: opts.saveFrames,
		Log:           a.log,
	})
	// A failure with nothing to show must not replace an earlier report.
	if scanErr != nil && len(records) == 0 {
		return scanErr
	}
	records = keepOnly(records, opts.minRarity)
	if err := scan.WriteJSON(opts.out, scan.NewReport(records, scanErr == nil)); err != nil {
		return err
	}
	a.printArtifactSummary(summarizeArtifacts(records), opts.out)
	if scanErr != nil {
		return fmt.Errorf("scan stopped early, partial results saved: %w", scanErr)
	}
	return nil
}

// artifactSource picks saved screenshots or the live game.
func (a *app) artifactSource(ctx context.Context, opts artifactOptions, engine engine, parser scan.ArtifactParser) (scan.FrameSource, error) {
	if opts.fromDir != "" {
		return scan.DirFrames(opts.fromDir), nil
	}
	game, err := a.attachForInput(ctx, opts.countdown)
	if err != nil {
		return nil, err
	}
	return scan.InventoryWalker{
		Game:       game,
		Layout:     engine.layout.Inventory,
		OCR:        engine.ocr,
		IsMaterial: parser.IsUnrelated,
		Pace:       scan.Pace(opts.pace),
		Limit:      opts.limit,
		Count:      opts.count,
		Log:        a.log,
	}.Walk, nil
}

// artifactSummary counts the outcomes of an artifact scan.
type artifactSummary struct {
	total      int
	exportable int
	withIssues int
	corrected  int
}

func summarizeArtifacts(records []scan.ArtifactRecord) artifactSummary {
	summary := artifactSummary{total: len(records)}
	for _, r := range records {
		if r.Exportable() {
			summary.exportable++
		}
		if len(r.Issues) > 0 {
			summary.withIssues++
		}
		if len(r.Corrections) > 0 {
			summary.corrected++
		}
	}
	return summary
}

func (a *app) printArtifactSummary(s artifactSummary, path string) {
	fmt.Fprintf(a.stdout, "Scanned %d artifacts: %d exportable, %d with issues, %d with repaired OCR errors.\n",
		s.total, s.exportable, s.withIssues, s.corrected)
	fmt.Fprintf(a.stdout, "Report written to %s\n", path)
	if s.withIssues > 0 {
		fmt.Fprintln(a.stdout, `Check the "issues" fields in the report. If many fail the same way, run "gensin-scan calibrate".`)
	}
}

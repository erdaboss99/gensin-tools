package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"time"

	"github.com/erdaboss99/gensin-scan/internal/scan"
)

// unknownCount marks a game total that could not be read.
const unknownCount = -1

// achievementOptions holds the achievements command's flags.
type achievementOptions struct {
	engine       engineFlags
	out          string
	fromDir      string
	resume       string
	limit        int
	saveFailures string
	pace         float64
	countdown    time.Duration
}

func (o *achievementOptions) register(flags *flag.FlagSet) {
	o.engine.register(flags)
	flags.StringVar(&o.out, "o", "achievements.json", "output report file")
	flags.StringVar(&o.fromDir, "from-dir", "", "scan the screenshots in this directory instead of the live game")
	flags.StringVar(&o.resume, "resume", "", "continue from an earlier report, rescanning only what it lacks")
	flags.IntVar(&o.limit, "limit", 0, "stop after this many searches (0 scans everything)")
	flags.StringVar(&o.saveFailures, "save-failures", "", "save a screenshot of each unconfirmed result to this directory")
	flags.Float64Var(&o.pace, "pace", 1, "multiplier for delays between actions; raise it if searches get missed")
	flags.DurationVar(&o.countdown, "countdown", defaultCountdown, "delay before a live scan starts")
}

// achievementScan is the outcome of scanning achievements either way.
type achievementScan struct {
	records []scan.AchievementRecord
	// gameCount is the game's own completed total, or unknownCount.
	gameCount int
	// err explains an early stop. The records collected so far remain usable.
	err error
}

func (a *app) runAchievements(ctx context.Context, args []string) error {
	var opts achievementOptions
	flags := a.newFlagSet("achievements")
	opts.register(flags)
	if err := parseFlags(flags, args); err != nil {
		return err
	}

	engine, err := opts.engine.load(ctx)
	if err != nil {
		return err
	}
	reader := scan.AchievementReader{Layout: engine.layout.Achievements, OCR: engine.ocr}

	var result achievementScan
	if opts.fromDir != "" {
		result.records, result.err = scan.ScanAchievementFrames(ctx, scan.DirFrames(opts.fromDir), reader, engine.catalog)
		result.gameCount = unknownCount
	} else if result, err = a.scanLiveAchievements(ctx, opts, engine, reader); err != nil {
		return err
	}
	// A failure with nothing to show must not replace an earlier report.
	if result.err != nil && len(result.records) == 0 {
		return result.err
	}

	if err := scan.WriteJSON(opts.out, scan.NewReport(result.records, result.err == nil)); err != nil {
		return err
	}
	a.printAchievementSummary(summarizeAchievements(result.records), opts.out, result.gameCount)
	if result.err != nil {
		return fmt.Errorf("scan stopped early, resume with --resume %s: %w", opts.out, result.err)
	}
	return nil
}

func (a *app) scanLiveAchievements(ctx context.Context, opts achievementOptions, engine engine, reader scan.AchievementReader) (achievementScan, error) {
	result := achievementScan{gameCount: unknownCount}
	done, err := loadResume(opts.resume)
	if err != nil {
		return result, err
	}
	game, err := a.attachForInput(ctx, opts.countdown)
	if err != nil {
		return result, err
	}
	walker := scan.AchievementWalker{
		Game:            game,
		Layout:          engine.layout.Achievements,
		Reader:          reader,
		Catalog:         engine.catalog,
		Pace:            scan.Pace(opts.pace),
		Limit:           opts.limit,
		Done:            done,
		SaveFailuresDir: opts.saveFailures,
		Log:             a.log,
	}
	count, ok, err := walker.ReadCompletedCount(ctx)
	if err != nil {
		return result, err
	}
	if ok {
		result.gameCount = count
	}
	result.records, result.err = walker.Run(ctx)
	return result, nil
}

func loadResume(path string) ([]scan.AchievementRecord, error) {
	if path == "" {
		return nil, nil
	}
	report, err := scan.ReadReport[scan.AchievementRecord](path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("--resume: %w", err)
	}
	if err != nil {
		return nil, err
	}
	return report.Items, nil
}

// achievementSummary counts the outcomes of an achievement scan.
type achievementSummary struct {
	total      int
	stages     int
	notFound   int
	review     int
	unverified int
}

func summarizeAchievements(records []scan.AchievementRecord) achievementSummary {
	summary := achievementSummary{total: len(records), stages: len(scan.CompletedAchievements(records))}
	for _, r := range records {
		switch {
		case r.NotFound:
			summary.notFound++
		case r.NeedsReview:
			summary.review++
		case !r.Verified:
			summary.unverified++
		}
	}
	return summary
}

func (a *app) printAchievementSummary(s achievementSummary, path string, gameCount int) {
	fmt.Fprintf(a.stdout, "Scanned %d achievements: %d stages complete, %d not listed by the game's search, %d need review, %d unverified.\n",
		s.total, s.stages, s.notFound, s.review, s.unverified)
	fmt.Fprintf(a.stdout, "Report written to %s\n", path)
	if gameCount != unknownCount {
		fmt.Fprintf(a.stdout, "The game reports %d completed; this scan found %d completed stages.\n", gameCount, s.stages)
	}
	if s.review > 0 {
		fmt.Fprintf(a.stdout, "To retry the %d that need review, run again with --resume %s\n", s.review, path)
	}
	if s.total > 0 && s.unverified > s.total/2 {
		fmt.Fprintln(a.stdout, `Most titles could not be verified, so the title region is probably off. Run "gensin-scan calibrate --screen achievements".`)
	}
}

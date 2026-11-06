package scan

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/erdaboss99/gensin-scan/control"
	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/game"
	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
)

const (
	// resultPolls and resultPollDelay bound the wait for search results to appear.
	resultPolls     = 25
	resultPollDelay = 100 * time.Millisecond
	// retryDelay is the wait before a second look at an unconfirmed result.
	retryDelay = 700 * time.Millisecond
	// typingDelay lets the search box catch up before Enter is pressed.
	typingDelay = 100 * time.Millisecond

	clearFocusDelay  = 100 * time.Millisecond
	clearButtonDelay = 250 * time.Millisecond

	failureFrameFormat = "achievement-%d.png"
)

// AchievementWalker searches for each achievement in the live game.
type AchievementWalker struct {
	Game    control.Game
	Layout  layout.Achievements
	Reader  AchievementReader
	Catalog *catalog.Catalog
	Pace    Pace
	Limit   int
	// Done holds earlier results to keep, which is how interrupted scans resume.
	Done []AchievementRecord
	// SaveFailuresDir, when set, stores the frame behind each unconfirmed result.
	SaveFailuresDir string
	Log             *slog.Logger
}

// Run scans every achievement, returning all records collected so far even when
// it stops early, so the caller can save them and resume.
func (w AchievementWalker) Run(ctx context.Context) ([]AchievementRecord, error) {
	width, height := w.Game.Size()
	win, err := layout.NewWindow(width, height)
	if err != nil {
		return w.Done, err
	}

	if w.SaveFailuresDir != "" {
		if err := os.MkdirAll(w.SaveFailuresDir, 0o755); err != nil {
			return w.Done, err
		}
	}

	known := make(map[game.AchievementID]bool, len(w.Done))
	records := make([]AchievementRecord, 0, len(w.Catalog.Achievements))
	for _, record := range w.Done {
		if keepResult(record) {
			known[record.IDs[0]] = true
			records = append(records, record)
		}
	}

	if err := w.clear(ctx, win); err != nil {
		return records, err
	}
	scanned := 0
	for _, entry := range w.Catalog.Achievements {
		if known[entry.IDs[0]] {
			continue
		}
		if w.Limit > 0 && scanned >= w.Limit {
			break
		}
		record, err := w.scanOne(ctx, win, entry)
		if err != nil {
			return records, fmt.Errorf("achievement %q: %w", entry.Name, err)
		}
		records = append(records, record)
		scanned++
		if w.Log != nil && scanned%progressInterval == 0 {
			w.Log.Info("progress", "scanned", scanned, "of", len(w.Catalog.Achievements)-len(known))
		}
	}
	return records, nil
}

func (w AchievementWalker) scanOne(ctx context.Context, win layout.Window, entry catalog.Achievement) (AchievementRecord, error) {
	var record AchievementRecord
	if err := w.Game.Type(entry.Name); err != nil {
		return record, err
	}
	if err := w.Pace.sleep(ctx, typingDelay); err != nil {
		return record, err
	}
	if err := w.Game.PressEnter(); err != nil {
		return record, err
	}
	frame, err := w.awaitResults(ctx, win)
	if err != nil {
		return record, err
	}
	ambiguous := w.Catalog.IsAmbiguous(entry)
	record, err = w.Reader.Classify(ctx, frame, entry, ambiguous)
	if err != nil {
		return record, err
	}
	if !keepResult(record) {
		// The list may have still been changing, so look once more before giving up.
		if err := w.Pace.sleep(ctx, retryDelay); err != nil {
			return record, err
		}
		if frame, err = w.Game.Capture(); err != nil {
			return record, err
		}
		if record, err = w.Reader.Classify(ctx, frame, entry, ambiguous); err != nil {
			return record, err
		}
	}
	if !keepResult(record) && w.SaveFailuresDir != "" {
		path := filepath.Join(w.SaveFailuresDir, fmt.Sprintf(failureFrameFormat, entry.IDs[0]))
		if err := imaging.SavePNG(path, frame); err != nil {
			return record, fmt.Errorf("save frame: %w", err)
		}
		record.Source = path
	}
	return record, w.clear(ctx, win)
}

// awaitResults waits until the list has stopped changing and shows either rows or
// the empty search message, since results take a variable time to draw. If neither
// appears, it returns the latest frame once the time limit passes.
func (w AchievementWalker) awaitResults(ctx context.Context, win layout.Window) (*image.RGBA, error) {
	area := win.Rect(w.Layout.Results)
	var (
		frame *image.RGBA
		last  uint64
	)
	for poll := range resultPolls {
		if err := w.Pace.sleep(ctx, resultPollDelay); err != nil {
			return nil, err
		}
		var err error
		if frame, err = w.Game.Capture(); err != nil {
			return nil, err
		}
		hash := imaging.Fingerprint(frame, area)
		if poll > 0 && hash == last {
			if len(findRows(w.Layout, win, frame)) > 0 {
				return frame, nil
			}
			r, err := newReader(ctx, w.Reader.OCR, frame)
			if err != nil {
				return nil, err
			}
			empty, err := w.Reader.saysNoResults(r)
			if err != nil {
				return nil, err
			}
			if empty {
				return frame, nil
			}
		}
		last = hash
	}
	return frame, nil
}

// ReadCompletedCount reads the game's total of completed achievements. The search
// box must be showing the normal screen, with the count visible.
func (w AchievementWalker) ReadCompletedCount(ctx context.Context) (int, bool, error) {
	frame, err := w.Game.Capture()
	if err != nil {
		return 0, false, err
	}
	return w.Reader.ReadCompletedCount(ctx, frame)
}

// clear empties the search box. The first click focuses it and the second presses
// the clear button, which needs both whether the box currently has text or not.
func (w AchievementWalker) clear(ctx context.Context, win layout.Window) error {
	at := win.Point(w.Layout.ClearSearch)
	for _, wait := range []time.Duration{clearFocusDelay, clearButtonDelay} {
		if err := w.Game.Click(at.X, at.Y); err != nil {
			return err
		}
		if err := w.Pace.sleep(ctx, wait); err != nil {
			return err
		}
	}
	return nil
}

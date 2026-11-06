package scan

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/erdaboss99/gensin-scan/internal/imaging"
)

const (
	// progressInterval is how many items pass between progress log lines.
	progressInterval = 25

	savedFrameFormat = "frame-%05d.png"
)

// ScanOptions configures ScanArtifacts.
type ScanOptions struct {
	Workers int
	// SaveFramesDir, when set, stores each frame for later offline scans.
	SaveFramesDir string
	Log           *slog.Logger
}

// ScanArtifacts analyzes frames from source using a worker pool, returning records
// in frame order. Capture is much faster than OCR, so the queue is bounded to keep
// memory flat. Records finished before a failure are still returned with the error.
func ScanArtifacts(ctx context.Context, source FrameSource, parser ArtifactParser, opts ScanOptions) ([]ArtifactRecord, error) {
	workers := max(opts.Workers, 1)
	if opts.SaveFramesDir != "" {
		if err := os.MkdirAll(opts.SaveFramesDir, 0o755); err != nil {
			return nil, err
		}
	}

	workCtx, cancelWork := context.WithCancelCause(ctx)
	defer cancelWork(nil)
	queue := make(chan Frame, workers)

	var (
		mu      sync.Mutex
		records []ArtifactRecord
		wg      sync.WaitGroup
	)
	for range workers {
		wg.Go(func() {
			for frame := range queue {
				record, skipped, err := analyzeFrame(workCtx, frame, parser, opts.SaveFramesDir)
				if err != nil {
					cancelWork(err)
					continue
				}
				if skipped {
					continue
				}
				mu.Lock()
				records = append(records, record)
				done := len(records)
				mu.Unlock()
				if opts.Log != nil && done%progressInterval == 0 {
					opts.Log.Info("progress", "artifacts", done)
				}
			}
		})
	}

	sourceErr := source(workCtx, func(frame Frame) error {
		select {
		case queue <- frame:
			return nil
		case <-workCtx.Done():
			return context.Cause(workCtx)
		}
	})
	close(queue)
	wg.Wait()

	slices.SortFunc(records, func(a, b ArtifactRecord) int { return a.Index - b.Index })
	// A worker's failure is the root cause, and the source's error is then only
	// its echo. Otherwise the source's error, such as lost focus, is the story.
	if cause := context.Cause(workCtx); cause != nil && ctx.Err() == nil {
		return records, cause
	}
	return records, sourceErr
}

func analyzeFrame(ctx context.Context, frame Frame, parser ArtifactParser, saveDir string) (ArtifactRecord, bool, error) {
	source := frame.Source
	if saveDir != "" {
		source = filepath.Join(saveDir, fmt.Sprintf(savedFrameFormat, frame.Index))
		if err := imaging.SavePNG(source, frame.Image); err != nil {
			return ArtifactRecord{}, false, fmt.Errorf("save frame: %w", err)
		}
	}
	record, skipped, err := parser.Parse(ctx, frame.Image)
	if err != nil {
		return ArtifactRecord{}, false, err
	}
	record.Index = frame.Index
	record.Source = source
	if frame.Duplicate {
		record.issue("identical to the previous frame: the click may not have registered")
	}
	return record, skipped, nil
}

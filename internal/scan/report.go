package scan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/erdaboss99/gensin-scan/internal/game"
)

// ReportVersion identifies the on-disk report format.
const ReportVersion = 1

// Report is the native output of a scan: every record plus the context needed to
// judge it. Export commands turn reports into formats other tools understand.
type Report[T any] struct {
	Version   int       `json:"version"`
	ScannedAt time.Time `json:"scannedAt"`
	// Complete is false when the scan stopped early.
	Complete bool `json:"complete"`
	Items    []T  `json:"items"`
}

// NewReport creates a report for the given records.
func NewReport[T any](items []T, complete bool) Report[T] {
	return Report[T]{Version: ReportVersion, ScannedAt: time.Now().UTC(), Complete: complete, Items: items}
}

// WriteJSON saves v as indented JSON, replacing path atomically so an interrupted
// write cannot destroy an earlier scan.
func WriteJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ReadReport loads a report written by WriteJSON.
func ReadReport[T any](path string) (Report[T], error) {
	var report Report[T]
	raw, err := os.ReadFile(path)
	if err != nil {
		return report, err
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return report, fmt.Errorf("parse %s: %w", path, err)
	}
	if report.Version != ReportVersion {
		return report, fmt.Errorf("%s: unsupported report version %d", path, report.Version)
	}
	return report, nil
}

// CompletedAchievements flattens records into completed stages.
func CompletedAchievements(records []AchievementRecord) []game.CompletedAchievement {
	var out []game.CompletedAchievement
	for _, record := range records {
		out = append(out, record.Completed()...)
	}
	return out
}

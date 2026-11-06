package scan

import (
	"context"
	"fmt"

	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/game"
)

// ScanAchievementFrames reads saved screenshots of achievement lists without any
// game automation. Each visible row is identified purely by OCR of its title, and
// an achievement seen twice keeps its highest star count.
func ScanAchievementFrames(ctx context.Context, source FrameSource, reader AchievementReader, cat *catalog.Catalog) ([]AchievementRecord, error) {
	found := make(map[game.AchievementID]*AchievementRecord)
	err := source(ctx, func(frame Frame) error {
		r, err := newReader(ctx, reader.OCR, frame.Image)
		if err != nil {
			return fmt.Errorf("%s: %w", frame.Source, err)
		}
		for _, center := range findRows(reader.Layout, r.win, frame.Image) {
			row, err := reader.readRow(r, center)
			if err != nil {
				return err
			}
			match, ok := cat.MatchAchievement(row.title)
			if !ok {
				continue
			}
			if row.status, err = reader.readStatus(r, center); err != nil {
				return err
			}
			entry := match.Item
			stars := row.stagesDone(len(entry.IDs))
			if prev := found[entry.IDs[0]]; prev == nil || stars > prev.Stars {
				found[entry.IDs[0]] = &AchievementRecord{
					Name: entry.Name, Category: entry.Category, IDs: entry.IDs,
					Stars: stars, Title: row.title, Status: row.status, Verified: true, Source: frame.Source,
				}
			}
		}
		return nil
	})

	records := make([]AchievementRecord, 0, len(found))
	for _, entry := range cat.Achievements {
		if record, ok := found[entry.IDs[0]]; ok {
			records = append(records, *record)
		}
	}
	return records, err
}

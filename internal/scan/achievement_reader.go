package scan

import (
	"context"
	"image"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/fuzzy"
	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
	"github.com/erdaboss99/gensin-scan/ocr"
)

const (
	// minTitleScore is the similarity at which a result row's title counts as the
	// achievement that was searched for.
	minTitleScore = 85
	// minStatusScore is the similarity at which a row's status reads "Completed".
	minStatusScore = 70
	completedLabel = "Completed"
	// truncatedTitleLen is the length from which a title may have wrapped, leaving
	// only the start of the name on the line that gets read.
	truncatedTitleLen = 40
	// perfectScore ends the search for the best row early.
	perfectScore = 100
)

// achievementStar is the colour of an earned achievement star.
var achievementStar = imaging.ColorRange{Target: color.RGBA{R: 254, G: 242, B: 212, A: 255}, Tolerance: 10}

var (
	progressPattern   = regexp.MustCompile(`\d\s*/\s*\d`)
	lastNumberPattern = regexp.MustCompile(`(\d+)\D*$`)
)

// isMedalGold matches the gold laurels on every achievement medal, finished or not.
func isMedalGold(r, g, b uint8) bool {
	return r > 190 && g > 140 && g < 225 && b < 150 && int(r)-int(b) > 70
}

// AchievementReader reads achievement rows from frames.
type AchievementReader struct {
	Layout layout.Achievements
	OCR    ocr.Recognizer
}

// resultRow is one row read from a frame.
type resultRow struct {
	title  string
	status string
	stars  int
}

// rowMatch is a row's similarity to the achievement being looked for.
type rowMatch struct {
	row    resultRow
	center int
	score  int
}

// stagesDone combines the star count with the status text, which is the stronger
// signal. "Completed" means every stage is finished, while visible progress means
// at least one is not.
func (row resultRow) stagesDone(stages int) int {
	switch {
	case fuzzy.Ratio(row.status, completedLabel) >= minStatusScore:
		return stages
	case progressPattern.MatchString(row.status):
		return min(row.stars, stages-1)
	default:
		return min(row.stars, stages)
	}
}

// findRows returns the vertical centre of each result row, found from its medal.
func findRows(l layout.Achievements, win layout.Window, frame *image.RGBA) []int {
	spans := imaging.VerticalSpans(frame, win.Rect(l.Medals), isMedalGold, max(win.Len(3), 2), win.Len(l.MedalMinHeight))
	centers := make([]int, 0, len(spans))
	for _, span := range spans {
		if len(centers) == l.MaxRows {
			break
		}
		centers = append(centers, span.Center())
	}
	return centers
}

func (a AchievementReader) readRow(r reader, center int) (resultRow, error) {
	text, err := r.readRect(r.win.RowRect(a.Layout.Title, center), ocr.ModeBlock, "")
	if err != nil {
		return resultRow{}, err
	}
	// The box can also take in the description below, so only the first line counts.
	return resultRow{title: firstTextLine(text), stars: a.countStars(r, center)}, nil
}

// titleScore compares a row's title with an achievement's name. A long name wraps
// onto a second line, so a long title also counts if it matches the start of the name.
func titleScore(title, name string) int {
	score := fuzzy.Ratio(title, name)
	normTitle, normName := []rune(fuzzy.Normalize(title)), []rune(fuzzy.Normalize(name))
	if len(normTitle) >= truncatedTitleLen && len(normTitle) < len(normName) {
		score = max(score, fuzzy.Ratio(string(normTitle), string(normName[:len(normTitle)])))
	}
	return score
}

// ReadCompletedCount reads the game's own total of completed achievements.
func (a AchievementReader) ReadCompletedCount(ctx context.Context, frame *image.RGBA) (int, bool, error) {
	r, err := newReader(ctx, a.OCR, frame)
	if err != nil {
		return 0, false, err
	}
	text, err := r.read(a.Layout.CompletedCount, ocr.ModeLine, "")
	if err != nil {
		return 0, false, err
	}
	match := lastNumberPattern.FindStringSubmatch(text)
	if match == nil {
		return 0, false, nil
	}
	count, err := strconv.Atoi(match[1])
	return count, err == nil, nil
}

// saysNoResults reports whether the game is showing its empty search message.
func (a AchievementReader) saysNoResults(r reader) (bool, error) {
	text, err := r.read(a.Layout.NoResults, ocr.ModeBlock, "")
	if err != nil {
		return false, err
	}
	normalized := fuzzy.Normalize(text)
	return strings.Contains(normalized, "no achievements") || strings.Contains(normalized, "achievements related"), nil
}

// readStatus is separate because only the row that gets used needs this extra OCR.
func (a AchievementReader) readStatus(r reader, center int) (string, error) {
	return r.readRect(r.win.RowRect(a.Layout.Status, center), ocr.ModeLine, "")
}

func (a AchievementReader) countStars(r reader, center int) int {
	lo, hi := r.win.Len(a.Layout.StarMinSize), r.win.Len(a.Layout.StarMaxSize)
	stars := 0
	for _, blob := range imaging.FindBlobs(r.frame, r.win.RowRect(a.Layout.Stars, center), achievementStar) {
		w, h := blob.Bounds.Dx(), blob.Bounds.Dy()
		if w >= lo && w <= hi && h >= lo && h <= hi {
			stars++
		}
	}
	return stars
}

// Classify reads a search result frame for the achievement that was searched for.
//
// Searches match substrings, so other achievements can appear too, and some names
// nearly contain each other ("To Brave the Lightning's Glow" and "Brave the
// Lightning's Glow..."). The row with the most similar title therefore wins, not
// the first one that's close enough. A single result for an unambiguous name can't
// be anything else, so it counts even with an unreadable title, just unverified.
// Anything else that can't be confirmed is left for review rather than guessed.
func (a AchievementReader) Classify(ctx context.Context, frame *image.RGBA, entry catalog.Achievement, ambiguous bool) (AchievementRecord, error) {
	record := AchievementRecord{Name: entry.Name, Category: entry.Category, IDs: entry.IDs}
	r, err := newReader(ctx, a.OCR, frame)
	if err != nil {
		return record, err
	}

	centers := findRows(a.Layout, r.win, frame)
	if len(centers) == 0 {
		found, err := a.saysNoResults(r)
		if err != nil {
			return record, err
		}
		record.NotFound = found
		record.NeedsReview = !found
		return record, nil
	}

	best, err := a.bestRow(r, centers, entry.Name)
	if err != nil {
		return record, err
	}
	record.Title = best.row.title
	record.Verified = best.score >= minTitleScore
	if !record.Verified && (ambiguous || len(centers) > 1) {
		record.NeedsReview = true
		return record, nil
	}
	if best.row.status, err = a.readStatus(r, best.center); err != nil {
		return record, err
	}
	record.Status = best.row.status
	record.Stars = best.row.stagesDone(len(entry.IDs))
	return record, nil
}

// bestRow reads rows until one matches name exactly, and returns the closest one.
func (a AchievementReader) bestRow(r reader, centers []int, name string) (rowMatch, error) {
	best := rowMatch{score: -1}
	for _, center := range centers {
		row, err := a.readRow(r, center)
		if err != nil {
			return best, err
		}
		if score := titleScore(row.title, name); score > best.score {
			best = rowMatch{row: row, center: center, score: score}
		}
		if best.score == perfectScore {
			break
		}
	}
	return best, nil
}

package scan

import "github.com/erdaboss99/gensin-scan/internal/game"

// AchievementRecord is the scan result for one achievement.
type AchievementRecord struct {
	Name     string               `json:"name"`
	Category game.CategoryID      `json:"category"`
	IDs      []game.AchievementID `json:"ids"`
	// Stars is the number of completed stages.
	Stars int `json:"stars"`
	// Title is what OCR read from the matching row.
	Title string `json:"title,omitempty"`
	// Status is what OCR read beside it: "Completed", or progress such as "1/30".
	Status string `json:"status,omitempty"`
	// Verified means OCR confirmed the row was this achievement.
	Verified bool `json:"verified"`
	// NotFound means the game's search said it has no such achievement, so it was
	// not counted as complete. The game keeps some incomplete achievements out of
	// its search. Compare the game's completed total to confirm none were missed.
	NotFound bool `json:"notFound,omitempty"`
	// NeedsReview marks a result that could not be attributed to this achievement,
	// so it was not counted as complete.
	NeedsReview bool   `json:"needsReview,omitempty"`
	Source      string `json:"source,omitempty"`
}

// Completed lists the stages this record marks as complete.
func (r AchievementRecord) Completed() []game.CompletedAchievement {
	out := make([]game.CompletedAchievement, 0, r.Stars)
	for _, id := range r.IDs[:min(r.Stars, len(r.IDs))] {
		out = append(out, game.CompletedAchievement{ID: id, Category: r.Category})
	}
	return out
}

// keepResult reports whether a result is settled, so it needs no second look now and
// can be reused when resuming. Anything uncertain is searched again.
func keepResult(record AchievementRecord) bool {
	return record.NotFound || (record.Verified && !record.NeedsReview)
}

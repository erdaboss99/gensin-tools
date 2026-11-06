package catalog

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/erdaboss99/gensin-scan/internal/game"
)

// paimonStage is one stage of an achievement.
type paimonStage struct {
	ID   game.AchievementID `json:"id"`
	Name string             `json:"name"`
}

// paimonEntry is either a single stage object or an array of stages.
type paimonEntry []paimonStage

// UnmarshalJSON accepts both shapes.
func (e *paimonEntry) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
		return json.Unmarshal(data, (*[]paimonStage)(e))
	}
	var stage paimonStage
	if err := json.Unmarshal(data, &stage); err != nil {
		return err
	}
	*e = paimonEntry{stage}
	return nil
}

// paimonCategory is one achievement category.
type paimonCategory struct {
	Name         string        `json:"name"`
	Order        int           `json:"order"`
	Achievements []paimonEntry `json:"achievements"`
}

// paimonDocument is the whole data file, keyed by category ID.
type paimonDocument map[string]paimonCategory

// orderedCategory is a converted category awaiting sorting into the game's order.
type orderedCategory struct {
	order   int
	id      int
	entries []Achievement
}

func fetchAchievements(ctx context.Context, client *http.Client, src Source) ([]Achievement, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.AchievementsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	var doc paimonDocument
	if err := doJSON(client, req, &doc); err != nil {
		return nil, err
	}

	categories := make([]orderedCategory, 0, len(doc))
	for key, category := range doc {
		id, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("category id %q: %w", key, err)
		}
		categories = append(categories, newOrderedCategory(id, category))
	}
	slices.SortFunc(categories, func(a, b orderedCategory) int {
		return cmp.Or(cmp.Compare(a.order, b.order), cmp.Compare(a.id, b.id))
	})

	var out []Achievement
	for _, category := range categories {
		out = append(out, category.entries...)
	}
	return out, nil
}

func newOrderedCategory(id int, category paimonCategory) orderedCategory {
	ordered := orderedCategory{order: category.Order, id: id}
	for _, entry := range category.Achievements {
		if len(entry) == 0 {
			continue
		}
		achievement := Achievement{
			Name:         strings.TrimSpace(entry[0].Name),
			Category:     game.CategoryID(id),
			CategoryName: category.Name,
		}
		for _, stage := range entry {
			achievement.IDs = append(achievement.IDs, stage.ID)
		}
		ordered.entries = append(ordered.entries, achievement)
	}
	return ordered
}

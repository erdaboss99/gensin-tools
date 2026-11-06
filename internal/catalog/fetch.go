package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	userAgent   = "gensin-scan (+https://github.com/erdaboss99/gensin-scan)"
	httpTimeout = 45 * time.Second
)

// Source says where update-data downloads from.
type Source struct {
	WikiListURL     string
	WikiEntryURL    string
	AchievementsURL string
}

// DefaultSource returns the public HoYoWiki and Paimon.moe endpoints.
func DefaultSource() Source {
	return Source{
		WikiListURL:     "https://sg-wiki-api.hoyolab.com/hoyowiki/genshin/wapi/get_entry_page_list",
		WikiEntryURL:    "https://sg-wiki-api-static.hoyolab.com/hoyowiki/genshin/wapi/entry_page",
		AchievementsURL: "https://raw.githubusercontent.com/MadeBaruna/paimon-moe/main/src/data/achievement/en.json",
	}
}

// NewHTTPClient returns the client update-data uses.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: httpTimeout}
}

// Fetch downloads fresh reference data.
func Fetch(ctx context.Context, client *http.Client, src Source) (Data, error) {
	pieces, err := fetchPieces(ctx, client, src)
	if err != nil {
		return Data{}, fmt.Errorf("artifacts: %w", err)
	}
	characters, err := fetchCharacters(ctx, client, src)
	if err != nil {
		return Data{}, fmt.Errorf("characters: %w", err)
	}
	achievements, err := fetchAchievements(ctx, client, src)
	if err != nil {
		return Data{}, fmt.Errorf("achievements: %w", err)
	}
	return Data{Pieces: pieces, Characters: characters, Achievements: achievements}, nil
}

func doJSON[T any](client *http.Client, req *http.Request, out *T) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected status %s", req.URL, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s: %w", req.URL, err)
	}
	return nil
}

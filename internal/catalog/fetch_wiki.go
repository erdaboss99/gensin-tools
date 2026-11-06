package catalog

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/erdaboss99/gensin-scan/internal/game"
)

const (
	wikiMenuCharacters = "2"
	wikiMenuArtifacts  = "5"
	wikiPageSize       = 50
	wikiArtifactList   = "artifact_list"
	wikiReferer        = "https://wiki.hoyolab.com/"
	wikiLanguage       = "en-us"
	travelerName       = "Traveler"
)

// wikiCorrections replaces piece names the wiki has wrong, by set and slot.
var wikiCorrections = map[string]map[game.Slot]string{
	"Obsidian Codex": {game.SlotFlower: "Reckoning of the Xenogenic"},
}

// wikiListRequest asks for one page of a wiki menu.
type wikiListRequest struct {
	Filters  []string `json:"filters"`
	MenuID   string   `json:"menu_id"`
	PageNum  int      `json:"page_num"`
	PageSize int      `json:"page_size"`
	UseES    bool     `json:"use_es"`
}

// wikiListItem is one entry of a wiki menu.
type wikiListItem struct {
	ID   string `json:"entry_page_id"`
	Name string `json:"name"`
}

// wikiListData is the payload of a menu page.
type wikiListData struct {
	List  []wikiListItem `json:"list"`
	Total string         `json:"total"`
}

// wikiListResponse is the response to a wikiListRequest.
type wikiListResponse struct {
	Retcode int          `json:"retcode"`
	Message string       `json:"message"`
	Data    wikiListData `json:"data"`
}

// wikiComponent is one block of a wiki page. Data holds JSON encoded as a string.
type wikiComponent struct {
	ID   string `json:"component_id"`
	Data string `json:"data"`
}

// wikiModule is a group of components on a wiki page.
type wikiModule struct {
	Components []wikiComponent `json:"components"`
}

// wikiPage is one wiki page.
type wikiPage struct {
	Modules []wikiModule `json:"modules"`
}

// wikiEntryData is the payload of a page request.
type wikiEntryData struct {
	Page wikiPage `json:"page"`
}

// wikiEntryResponse is the response to a page request.
type wikiEntryResponse struct {
	Data wikiEntryData `json:"data"`
}

// wikiPiece is one entry of the wiki's artifact_list component. The object keys
// (flower_of_life, ...) are shuffled for recent sets, while position is correct,
// so position is what gets used.
type wikiPiece struct {
	Title    string `json:"title"`
	Position string `json:"position"`
}

// wikiSet maps the artifact_list component's keys to the pieces of one set.
type wikiSet map[string]wikiPiece

func wikiRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", wikiReferer)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Rpc-Language", wikiLanguage)
	return req, nil
}

func wikiList(ctx context.Context, client *http.Client, src Source, menu string) ([]wikiListItem, error) {
	var items []wikiListItem
	for page := 1; ; page++ {
		payload, err := json.Marshal(wikiListRequest{
			Filters: []string{}, MenuID: menu, PageNum: page, PageSize: wikiPageSize, UseES: true,
		})
		if err != nil {
			return nil, err
		}
		req, err := wikiRequest(ctx, http.MethodPost, src.WikiListURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		var resp wikiListResponse
		if err := doJSON(client, req, &resp); err != nil {
			return nil, err
		}
		if resp.Retcode != 0 {
			return nil, fmt.Errorf("wiki list: retcode %d: %s", resp.Retcode, resp.Message)
		}
		items = append(items, resp.Data.List...)
		total, err := strconv.Atoi(resp.Data.Total)
		if err != nil {
			return nil, fmt.Errorf("wiki list: bad total %q: %w", resp.Data.Total, err)
		}
		if len(items) >= total || len(resp.Data.List) == 0 {
			return items, nil
		}
	}
}

func fetchCharacters(ctx context.Context, client *http.Client, src Source) ([]string, error) {
	items, err := wikiList(ctx, client, src, wikiMenuCharacters)
	if err != nil {
		return nil, err
	}
	names := []string{travelerName}
	for _, item := range items {
		names = append(names, strings.TrimSpace(item.Name))
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}

func fetchPieces(ctx context.Context, client *http.Client, src Source) ([]Piece, error) {
	sets, err := wikiList(ctx, client, src, wikiMenuArtifacts)
	if err != nil {
		return nil, err
	}
	var pieces []Piece
	for _, set := range sets {
		entries, err := fetchSet(ctx, client, src, set.ID)
		if err != nil {
			return nil, fmt.Errorf("set %q: %w", set.Name, err)
		}
		pieces = append(pieces, piecesFromSet(strings.TrimSpace(set.Name), entries)...)
	}
	return cleanPieces(pieces), nil
}

func fetchSet(ctx context.Context, client *http.Client, src Source, id string) (wikiSet, error) {
	req, err := wikiRequest(ctx, http.MethodGet, src.WikiEntryURL+"?entry_page_id="+id, nil)
	if err != nil {
		return nil, err
	}
	var resp wikiEntryResponse
	if err := doJSON(client, req, &resp); err != nil {
		return nil, err
	}
	for _, module := range resp.Data.Page.Modules {
		for _, component := range module.Components {
			if component.ID != wikiArtifactList {
				continue
			}
			var entries wikiSet
			if err := json.Unmarshal([]byte(component.Data), &entries); err != nil {
				return nil, fmt.Errorf("%s: %w", wikiArtifactList, err)
			}
			return entries, nil
		}
	}
	return nil, errors.New("page has no artifact_list component")
}

func piecesFromSet(set string, entries wikiSet) []Piece {
	var pieces []Piece
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Title)
		if name == "" {
			continue
		}
		slot, _ := game.SlotFromLabel(entry.Position)
		if fixed, ok := wikiCorrections[set][slot]; ok {
			name = fixed
		}
		pieces = append(pieces, Piece{Name: name, Set: set, Slot: slot})
	}
	return pieces
}

// cleanPieces sorts pieces and clears the slot of any name listed under several
// slots, since such an entry cannot be trusted to identify a slot.
func cleanPieces(pieces []Piece) []Piece {
	slices.SortFunc(pieces, func(a, b Piece) int {
		return cmp.Or(cmp.Compare(a.Set, b.Set), cmp.Compare(a.Name, b.Name), cmp.Compare(a.Slot, b.Slot))
	})
	out := pieces[:0]
	for _, p := range pieces {
		if n := len(out); n > 0 && out[n-1].Name == p.Name && out[n-1].Set == p.Set {
			out[n-1].Slot = ""
			continue
		}
		out = append(out, p)
	}
	return out
}

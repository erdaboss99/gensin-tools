package layout

// Achievements locates the achievements screen after a search. Rows aren't evenly
// spaced, because category headers and two-line descriptions push them apart, so
// each one is found from its medal and everything else is placed relative to that.
type Achievements struct {
	// ClearSearch is the button that empties the search box. Clicking it also
	// leaves the box focused for typing.
	ClearSearch Point `json:"clearSearch"`
	// CompletedCount holds the game's own "Achievements completed" total.
	CompletedCount Box `json:"completedCount"`
	// Results is the list area, watched to tell when new results have finished drawing.
	Results Box `json:"results"`
	// NoResults holds the message shown when a search finds nothing.
	NoResults Box `json:"noResults"`
	// Medals is the column searched for each row's gold medal.
	Medals Box `json:"medals"`
	// MedalMinHeight rejects medals cut off by the edge of the list.
	MedalMinHeight int `json:"medalMinHeight"`
	// Title holds the achievement's name. Longer descriptions push the name further
	// up its row, so this is tall enough for all of them, and the name is always
	// the first line of text inside it.
	Title RowBox `json:"title"`
	// Stars surrounds the medal's star markers.
	Stars RowBox `json:"stars"`
	// Status holds either "Completed" or progress such as "1/30".
	Status RowBox `json:"status"`
	// MaxRows is how many result rows are inspected per search.
	MaxRows int `json:"maxRows"`
	// StarMinSize and StarMaxSize bound a star's width and height in reference pixels.
	StarMinSize int `json:"starMinSize"`
	StarMaxSize int `json:"starMaxSize"`
}

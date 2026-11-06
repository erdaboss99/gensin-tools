package layout

// Grid describes the inventory's item grid.
type Grid struct {
	Columns int `json:"columns"`
	// First is the centre of the top-left item.
	First Point `json:"first"`
	// DeltaX and DeltaY are the distances between neighbouring items.
	DeltaX int `json:"deltaX"`
	DeltaY int `json:"deltaY"`
	// VerticalPadding is the screen height not used by the grid.
	VerticalPadding int `json:"verticalPadding"`
	// LastRowY is the centre of the bottom row when scrolled fully down.
	LastRowY int `json:"lastRowY"`
	// ScrollTicksPerRow is how many mouse wheel ticks move the grid by one row.
	ScrollTicksPerRow float64 `json:"scrollTicksPerRow"`
}

// RowsPerPage returns how many complete rows fit on screen.
func (g Grid) RowsPerPage(w Window) int {
	return max((w.ReferenceHeight()-g.VerticalPadding)/g.DeltaY, 1)
}

// Inventory locates the artifact inventory screen.
type Inventory struct {
	Grid      Grid `json:"grid"`
	ItemCount Box  `json:"itemCount"`
	Panel     Box  `json:"panel"`
	Name      Box  `json:"name"`
	Slot      Box  `json:"slot"`
	MainStat  Box  `json:"mainStat"`
	Rarity    Box  `json:"rarity"`
	Level     Box  `json:"level"`
	Substats  Box  `json:"substats"`
	Equipped  Box  `json:"equipped"`
	// LockIcon and AstralIcon sample a corner of each toggle, which is dark when on.
	LockIcon   Box `json:"lockIcon"`
	AstralIcon Box `json:"astralIcon"`
	// ElixirBand samples where the purple Sanctifying Elixir band appears. When it
	// is present, the level, substats and icons sit ElixirShift further down.
	ElixirBand  Box `json:"elixirBand"`
	ElixirShift int `json:"elixirShift"`
	// StarMinArea and StarMaxArea bound one rarity star's size in reference pixels.
	StarMinArea int `json:"starMinArea"`
	StarMaxArea int `json:"starMaxArea"`
}

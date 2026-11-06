package layout

// Point is a position in reference coordinates.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Box is a rectangle in reference coordinates.
type Box struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
}

// Shifted returns the box moved down by dy reference pixels.
func (b Box) Shifted(dy int) Box {
	b.Top += dy
	b.Bottom += dy
	return b
}

// RowBox is a rectangle positioned relative to a result row. Left and Right are
// ordinary horizontal coordinates, while Top and Bottom are offsets from the row's
// vertical centre.
type RowBox struct {
	Left   int `json:"left"`
	Right  int `json:"right"`
	Top    int `json:"top"`
	Bottom int `json:"bottom"`
}

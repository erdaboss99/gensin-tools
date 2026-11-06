package layout

import (
	"fmt"
	"image"
	"math"
)

// ReferenceWidth is the window width every coordinate is written for.
const ReferenceWidth = 1280

// minHeightRatio rejects windows wider than 16:9, whose UI is arranged differently.
const minHeightRatio = 0.56

// Window maps reference coordinates onto a real window size.
type Window struct {
	Width, Height int
	scale         float64
}

// NewWindow validates a window size and returns its coordinate mapping.
func NewWindow(width, height int) (Window, error) {
	if width < ReferenceWidth {
		return Window{}, fmt.Errorf("window is %dx%d: at least %d pixels of width are needed for OCR to be reliable", width, height, ReferenceWidth)
	}
	if float64(height)/float64(width) < minHeightRatio {
		return Window{}, fmt.Errorf("window is %dx%d: aspect ratios wider than 16:9 are not supported", width, height)
	}
	return Window{Width: width, Height: height, scale: float64(width) / ReferenceWidth}, nil
}

// Len scales a distance.
func (w Window) Len(v int) int {
	return int(math.Round(float64(v) * w.scale))
}

// X resolves a horizontal coordinate.
func (w Window) X(v int) int {
	if v < 0 {
		return w.Width + w.Len(v)
	}
	return w.Len(v)
}

// Y resolves a vertical coordinate.
func (w Window) Y(v int) int {
	if v < 0 {
		return w.Height + w.Len(v)
	}
	return w.Len(v)
}

// Rect resolves a box to window pixels.
func (w Window) Rect(b Box) image.Rectangle {
	return image.Rect(w.X(b.Left), w.Y(b.Top), w.X(b.Right), w.Y(b.Bottom))
}

// RowRect resolves a row box for the row centred at centerY, in window pixels.
func (w Window) RowRect(b RowBox, centerY int) image.Rectangle {
	return image.Rect(w.X(b.Left), centerY+w.Len(b.Top), w.X(b.Right), centerY+w.Len(b.Bottom))
}

// Point resolves a point to window pixels.
func (w Window) Point(p Point) image.Point {
	return image.Pt(w.X(p.X), w.Y(p.Y))
}

// ReferenceHeight is the window height expressed in reference pixels.
func (w Window) ReferenceHeight() int {
	return int(math.Round(float64(w.Height) / w.scale))
}

// Area converts an area in reference pixels to window pixels.
func (w Window) Area(v int) int {
	return int(math.Round(float64(v) * w.scale * w.scale))
}

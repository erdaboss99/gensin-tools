package imaging

import "image"

// Span is a vertical range of an image, with Max exclusive.
type Span struct{ Min, Max int }

// Height returns the span's length.
func (s Span) Height() int { return s.Max - s.Min }

// Center returns the span's midpoint.
func (s Span) Center() int { return (s.Min + s.Max) / 2 }

// PixelMatcher decides whether a pixel's colour counts.
type PixelMatcher func(red, green, blue uint8) bool

// VerticalSpans finds runs of consecutive lines in r that each hold at least
// minPerLine pixels accepted by match, keeping runs at least minHeight tall.
func VerticalSpans(src *image.RGBA, r image.Rectangle, match PixelMatcher, minPerLine, minHeight int) []Span {
	r = r.Intersect(src.Bounds())
	var spans []Span
	start := -1
	closeRun := func(end int) {
		if start >= 0 && end-start >= minHeight {
			spans = append(spans, Span{start, end})
		}
		start = -1
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		line := src.Pix[src.PixOffset(r.Min.X, y):]
		count := 0
		for x := range r.Dx() {
			if match(line[x*4], line[x*4+1], line[x*4+2]) {
				count++
			}
		}
		switch {
		case count < minPerLine:
			closeRun(y)
		case start < 0:
			start = y
		}
	}
	closeRun(r.Max.Y)
	return spans
}

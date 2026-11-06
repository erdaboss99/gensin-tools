package imaging

import (
	"image"
	"image/color"
	"image/draw"
)

// Outline draws a rectangle border directly onto dst.
func Outline(dst *image.RGBA, r image.Rectangle, c color.RGBA, thickness int) {
	edges := []image.Rectangle{
		image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+thickness),
		image.Rect(r.Min.X, r.Max.Y-thickness, r.Max.X, r.Max.Y),
		image.Rect(r.Min.X, r.Min.Y, r.Min.X+thickness, r.Max.Y),
		image.Rect(r.Max.X-thickness, r.Min.Y, r.Max.X, r.Max.Y),
	}
	for _, e := range edges {
		draw.Draw(dst, e, image.NewUniform(c), image.Point{}, draw.Src)
	}
}

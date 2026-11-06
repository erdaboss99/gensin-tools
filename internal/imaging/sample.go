package imaging

import (
	"hash/fnv"
	"image"
	"image/color"
)

// MeanColor returns the average colour of src inside r.
func MeanColor(src *image.RGBA, r image.Rectangle) color.RGBA {
	r = r.Intersect(src.Bounds())
	if r.Empty() {
		return color.RGBA{A: 255}
	}
	var sumR, sumG, sumB int
	for y := r.Min.Y; y < r.Max.Y; y++ {
		line := src.Pix[src.PixOffset(r.Min.X, y):]
		for x := range r.Dx() {
			sumR += int(line[x*4])
			sumG += int(line[x*4+1])
			sumB += int(line[x*4+2])
		}
	}
	n := r.Dx() * r.Dy()
	return color.RGBA{R: uint8(sumR / n), G: uint8(sumG / n), B: uint8(sumB / n), A: 255}
}

// Fingerprint hashes the pixels inside r, so a scanner can notice that a click
// did not change what is on screen.
func Fingerprint(src *image.RGBA, r image.Rectangle) uint64 {
	r = r.Intersect(src.Bounds())
	h := fnv.New64a()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		i := src.PixOffset(r.Min.X, y)
		h.Write(src.Pix[i : i+r.Dx()*4])
	}
	return h.Sum64()
}

package imaging

import (
	"image"
	"image/color"
)

// ColorRange matches pixels within Tolerance of Target on every channel.
type ColorRange struct {
	Target    color.RGBA
	Tolerance int
}

func (c ColorRange) contains(r, g, b uint8) bool {
	return abs(int(r)-int(c.Target.R)) <= c.Tolerance &&
		abs(int(g)-int(c.Target.G)) <= c.Tolerance &&
		abs(int(b)-int(c.Target.B)) <= c.Tolerance
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// Blob is a connected region of matching pixels.
type Blob struct {
	Area   int
	Bounds image.Rectangle
}

// FindBlobs returns the 8-connected regions of src inside r that match c.
func FindBlobs(src *image.RGBA, r image.Rectangle, c ColorRange) []Blob {
	r = r.Intersect(src.Bounds())
	w, h := r.Dx(), r.Dy()
	if w == 0 || h == 0 {
		return nil
	}
	mask := make([]bool, w*h)
	for y := range h {
		line := src.Pix[src.PixOffset(r.Min.X, r.Min.Y+y):]
		for x := range w {
			mask[y*w+x] = c.contains(line[x*4], line[x*4+1], line[x*4+2])
		}
	}

	var blobs []Blob
	var stack []int
	for start, set := range mask {
		if !set {
			continue
		}
		mask[start] = false
		stack = append(stack[:0], start)
		blob := Blob{Bounds: image.Rect(start%w, start/w, start%w+1, start/w+1)}
		for len(stack) > 0 {
			at := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := at%w, at/w
			blob.Area++
			blob.Bounds = blob.Bounds.Union(image.Rect(x, y, x+1, y+1))
			for ny := max(y-1, 0); ny <= min(y+1, h-1); ny++ {
				for nx := max(x-1, 0); nx <= min(x+1, w-1); nx++ {
					if next := ny*w + nx; mask[next] {
						mask[next] = false
						stack = append(stack, next)
					}
				}
			}
		}
		blobs = append(blobs, blob)
	}
	return blobs
}

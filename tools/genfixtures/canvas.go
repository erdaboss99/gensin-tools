package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	// referenceWidth is the window width layout coordinates are written for.
	referenceWidth = 1280
	// lineSpacing is the distance between lines of text, relative to the font size.
	lineSpacing = 1.2
	// starInnerRadius is the size of a star's inner points relative to its outer ones.
	starInnerRadius = 0.5
	// starPoints is the number of corners on a star's outline.
	starPoints = 10
)

// fontWeight selects a typeface.
type fontWeight int

const (
	weightRegular fontWeight = iota
	weightBold
)

// fontSet holds the parsed typefaces.
type fontSet map[fontWeight]*opentype.Font

func loadFonts() (fontSet, error) {
	regular, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse regular font: %w", err)
	}
	bold, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse bold font: %w", err)
	}
	return fontSet{weightRegular: regular, weightBold: bold}, nil
}

// vertex is a point with sub-pixel precision.
type vertex struct{ x, y float64 }

// canvas draws in layout reference coordinates, scaled to the output width. As in
// the layout, a negative coordinate is measured from the right or bottom edge.
type canvas struct {
	image *image.RGBA
	fonts fontSet
	scale float64
}

func newCanvas(fonts fontSet, width, height int, background color.RGBA) *canvas {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
	return &canvas{image: img, fonts: fonts, scale: float64(width) / referenceWidth}
}

func (c *canvas) x(v float64) float64 { return c.resolve(v, c.image.Bounds().Dx()) }

func (c *canvas) y(v float64) float64 { return c.resolve(v, c.image.Bounds().Dy()) }

func (c *canvas) resolve(v float64, size int) float64 {
	if v < 0 {
		return math.Round(float64(size) + v*c.scale)
	}
	return math.Round(v * c.scale)
}

// rect fills a rectangle whose right and bottom edges are included.
func (c *canvas) rect(left, top, right, bottom float64, fill color.RGBA) {
	r := image.Rect(int(c.x(left)), int(c.y(top)), int(c.x(right))+1, int(c.y(bottom))+1)
	draw.Draw(c.image, r, image.NewUniform(fill), image.Point{}, draw.Src)
}

// text draws text with its top-left corner at (x, y), one line per "\n".
func (c *canvas) text(x, y float64, text string, size float64, fill color.RGBA, weight fontWeight) error {
	pixels := math.Round(size * c.scale)
	face, err := opentype.NewFace(c.fonts[weight], &opentype.FaceOptions{Size: pixels, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return fmt.Errorf("create font face: %w", err)
	}
	defer face.Close()

	drawer := font.Drawer{Dst: c.image, Src: image.NewUniform(fill), Face: face}
	baseline := c.y(y) + float64(face.Metrics().Ascent.Round())
	for line := range strings.SplitSeq(text, "\n") {
		drawer.Dot = fixed.P(int(c.x(x)), int(baseline))
		drawer.DrawString(line)
		baseline += math.Round(pixels * lineSpacing)
	}
	return nil
}

// star fills a five-pointed star without anti-aliasing, because the scanner counts
// stars by their exact colour.
func (c *canvas) star(centerX, centerY, radius float64, fill color.RGBA) {
	cx, cy, outer := c.x(centerX), c.y(centerY), radius*c.scale
	points := make([]vertex, starPoints)
	for i := range points {
		r := outer
		if i%2 == 1 {
			r = outer * starInnerRadius
		}
		angle := -math.Pi/2 + float64(i)*math.Pi/5
		points[i] = vertex{cx + r*math.Cos(angle), cy + r*math.Sin(angle)}
	}
	c.fillPolygon(points, fill)
}

// fillPolygon sets every pixel whose centre is inside the polygon.
func (c *canvas) fillPolygon(points []vertex, fill color.RGBA) {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range points {
		minX, minY = min(minX, p.x), min(minY, p.y)
		maxX, maxY = max(maxX, p.x), max(maxY, p.y)
	}
	for y := int(minY); y <= int(maxY); y++ {
		for x := int(minX); x <= int(maxX); x++ {
			if contains(points, vertex{float64(x) + 0.5, float64(y) + 0.5}) {
				c.image.SetRGBA(x, y, fill)
			}
		}
	}
}

// contains reports whether p is inside the polygon, using the even-odd rule.
func contains(points []vertex, p vertex) bool {
	inside := false
	for i, a := range points {
		b := points[(i+1)%len(points)]
		if (a.y > p.y) != (b.y > p.y) && p.x < (b.x-a.x)*(p.y-a.y)/(b.y-a.y)+a.x {
			inside = !inside
		}
	}
	return inside
}

// ellipseOutline draws the outline of the ellipse inside the given bounds.
func (c *canvas) ellipseOutline(left, top, right, bottom, thickness float64, stroke color.RGBA) {
	x0, y0, x1, y1 := c.x(left), c.y(top), c.x(right), c.y(bottom)
	cx, cy := (x0+x1)/2, (y0+y1)/2
	outerX, outerY := (x1-x0)/2, (y1-y0)/2
	width := math.Round(thickness * c.scale)
	for y := int(y0); y <= int(y1); y++ {
		for x := int(x0); x <= int(x1); x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			inOuter := dx*dx/(outerX*outerX)+dy*dy/(outerY*outerY) <= 1
			inInner := dx*dx/((outerX-width)*(outerX-width))+dy*dy/((outerY-width)*(outerY-width)) < 1
			if inOuter && !inInner {
				c.image.SetRGBA(x, y, stroke)
			}
		}
	}
}

func (c *canvas) save(path string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	return encoder.Encode(f, c.image)
}

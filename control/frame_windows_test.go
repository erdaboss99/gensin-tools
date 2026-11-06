package control

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func solidFrame(c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	return img
}

func TestAllBlack(t *testing.T) {
	if !allBlack(solidFrame(color.RGBA{0, 0, 0, 255})) {
		t.Error("black image not detected")
	}
	if allBlack(solidFrame(color.RGBA{1, 0, 0, 255})) {
		t.Error("non-black image reported as black")
	}
}

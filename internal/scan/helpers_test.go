package scan

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"sync/atomic"

	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/layout"
	"github.com/erdaboss99/gensin-scan/ocr"
)

// scriptedOCR returns canned answers in call order.
type scriptedOCR struct {
	replies []string
	calls   atomic.Int32
}

func (s *scriptedOCR) Recognize(context.Context, ocr.Request) (string, error) {
	i := int(s.calls.Add(1)) - 1
	if i >= len(s.replies) {
		return "", nil
	}
	return s.replies[i], nil
}

func blankFrame() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{40, 40, 40, 255}), image.Point{}, draw.Src)
	return img
}

func emptyParser() ArtifactParser {
	return ArtifactParser{Layout: layout.Default().Inventory, OCR: &scriptedOCR{}, Catalog: catalog.New(catalog.Data{})}
}

type failingOCR struct{ err error }

func (f failingOCR) Recognize(context.Context, ocr.Request) (string, error) { return "", f.err }

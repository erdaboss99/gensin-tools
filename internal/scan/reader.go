package scan

import (
	"context"
	"image"
	"strings"

	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
	"github.com/erdaboss99/gensin-scan/ocr"
)

// reader performs OCR on regions of one frame.
type reader struct {
	ctx   context.Context
	ocr   ocr.Recognizer
	frame *image.RGBA
	win   layout.Window
}

func newReader(ctx context.Context, rec ocr.Recognizer, frame *image.RGBA) (reader, error) {
	b := frame.Bounds()
	win, err := layout.NewWindow(b.Dx(), b.Dy())
	if err != nil {
		return reader{}, err
	}
	return reader{ctx: ctx, ocr: rec, frame: frame, win: win}, nil
}

// read recognizes the text in a box.
func (r reader) read(box layout.Box, mode ocr.Mode, allow string) (string, error) {
	return r.readRect(r.win.Rect(box), mode, allow)
}

func (r reader) readRect(rect image.Rectangle, mode ocr.Mode, allow string) (string, error) {
	img := imaging.PrepareForOCR(r.frame, rect, imaging.OCRScale(r.win.Width))
	return r.ocr.Recognize(r.ctx, ocr.Request{Image: img, Mode: mode, Allow: allow})
}

// readName reads an item name, which the game wraps onto two lines when it is long.
func (r reader) readName(box layout.Box) (string, error) {
	text, err := r.read(box, ocr.ModeBlock, "")
	if err != nil {
		return "", err
	}
	return strings.Join(strings.Fields(text), " "), nil
}

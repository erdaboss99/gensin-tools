package ocr

import (
	"context"
	"image"
)

// Mode is a Tesseract page segmentation mode.
type Mode int

// The page segmentation modes the scanners use.
const (
	ModeBlock Mode = 6 // A uniform block of text, such as the substat list.
	ModeLine  Mode = 7 // A single line of text.
)

// Request describes one recognition job.
type Request struct {
	Image *image.Gray
	Mode  Mode
	// Allow restricts output to these characters. Empty allows everything.
	Allow string
}

// Recognizer extracts text from an image.
type Recognizer interface {
	Recognize(ctx context.Context, req Request) (string, error)
}

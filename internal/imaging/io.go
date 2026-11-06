package imaging

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
)

// Load reads a PNG or JPEG file into an RGBA image whose origin is (0, 0).
func Load(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst, nil
}

// SavePNG writes img to path, favouring speed over compression.
func SavePNG(path string, img image.Image) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(f, img)
}

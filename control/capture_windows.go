package control

import (
	"fmt"
	"image"
	"unsafe"
)

func (w *window) Capture() (*image.RGBA, error) {
	img, err := captureScreen(w.origin, w.width, w.height)
	if err != nil {
		return nil, err
	}
	if allBlack(img) {
		return nil, ErrBlackFrame
	}
	return img, nil
}

// captureScreen copies a region of the screen into a new top-down RGBA image.
func captureScreen(origin point, width, height int) (*image.RGBA, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("cannot capture a %dx%d area", width, height)
	}
	screen, _, err := procGetDC.Call(0)
	if screen == 0 {
		return nil, fmt.Errorf("GetDC: %w", err)
	}
	defer procReleaseDC.Call(0, screen)

	memory, _, err := procCreateCompatibleDC.Call(screen)
	if memory == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC: %w", err)
	}
	defer procDeleteDC.Call(memory)

	bitmap, _, err := procCreateCompatibleBitmap.Call(screen, uintptr(width), uintptr(height))
	if bitmap == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap: %w", err)
	}
	defer procDeleteObject.Call(bitmap)

	previous, _, _ := procSelectObject.Call(memory, bitmap)
	ok, _, err := procBitBlt.Call(memory, 0, 0, uintptr(width), uintptr(height),
		screen, uintptr(origin.x), uintptr(origin.y), srcCopy|captureBlt)
	procSelectObject.Call(memory, previous)
	if ok == 0 {
		return nil, fmt.Errorf("BitBlt: %w", err)
	}

	header := bitmapInfoHeader{
		width:       int32(width),
		height:      -int32(height), // Negative selects a top-down image.
		planes:      1,
		bitCount:    32,
		compression: biRGB,
	}
	header.size = uint32(unsafe.Sizeof(header))
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	lines, _, err := procGetDIBits.Call(memory, bitmap, 0, uintptr(height),
		uintptr(unsafe.Pointer(&img.Pix[0])), uintptr(unsafe.Pointer(&header)), dibRGB)
	if lines == 0 {
		return nil, fmt.Errorf("GetDIBits: %w", err)
	}
	// GDI delivers BGRX, and image.RGBA needs RGBA with a real alpha value.
	for i := 0; i+3 < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+2] = img.Pix[i+2], img.Pix[i]
		img.Pix[i+3] = 0xFF
	}
	return img, nil
}

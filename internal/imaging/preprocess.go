package imaging

import "image"

// minChannel converts the part of src inside r to 8 bits using each pixel's darkest
// colour channel. Game text is usually grey, white or dark, while the backgrounds
// behind low-contrast labels are saturated orange or purple. Saturated colours have
// a dark channel and neutral ones don't, so this separates text that luminance
// would blend together.
func minChannel(src *image.RGBA, r image.Rectangle) *image.Gray {
	r = r.Intersect(src.Bounds())
	dst := image.NewGray(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := range r.Dy() {
		line := src.Pix[src.PixOffset(r.Min.X, r.Min.Y+y):]
		row := dst.Pix[y*dst.Stride:]
		for x := range r.Dx() {
			p := line[x*4 : x*4+3 : x*4+3]
			row[x] = min(p[0], p[1], p[2])
		}
	}
	return dst
}

const (
	// minTextContrast is the smallest luminance range that can contain readable text.
	minTextContrast = 32
	// ocrMargin is the clean border added around text, which Tesseract relies on.
	ocrMargin = 10
	// ocrTargetWidth is the window width at which game text needs no enlarging.
	ocrTargetWidth = 3840
)

// OCRScale returns the enlargement that brings a window's text to the height
// Tesseract reads best. Larger windows need less of it.
func OCRScale(windowWidth int) int {
	return (ocrTargetWidth + windowWidth - 1) / windowWidth
}

// PrepareForOCR turns a screen crop into what Tesseract reads best: dark text on
// a light background, full contrast, enlarged, with a clean margin. Polarity is
// detected from the image, because the panel mixes light-on-dark and
// dark-on-light text. A region with no contrast comes back uniformly white.
func PrepareForOCR(src *image.RGBA, r image.Rectangle, scale int) *image.Gray {
	gray := minChannel(src, r)
	if len(gray.Pix) == 0 {
		return gray
	}
	lo, hi, sum := uint8(255), uint8(0), 0
	for _, v := range gray.Pix {
		lo, hi = min(lo, v), max(hi, v)
		sum += int(v)
	}
	spread := int(hi) - int(lo)
	if spread < minTextContrast {
		// Stretching a blank region would amplify noise into shapes that Tesseract
		// happily reads as text, so it stays blank.
		for i := range gray.Pix {
			gray.Pix[i] = 255
		}
		return gray
	}
	// Text covers a minority of pixels, so the mean sits near the background.
	mean := sum / len(gray.Pix)
	invert := mean < (int(lo)+int(hi))/2
	for i, v := range gray.Pix {
		n := (int(v) - int(lo)) * 255 / spread
		if invert {
			n = 255 - n
		}
		gray.Pix[i] = uint8(n)
	}
	return addMargin(upscale(gray, scale), ocrMargin)
}

// upscale enlarges src by an integer factor using bilinear interpolation.
func upscale(src *image.Gray, factor int) *image.Gray {
	if factor <= 1 {
		return src
	}
	w, h := src.Rect.Dx(), src.Rect.Dy()
	dst := image.NewGray(image.Rect(0, 0, w*factor, h*factor))
	for y := range h * factor {
		sy := y / factor
		fy := y % factor
		y0 := src.Pix[sy*src.Stride:]
		y1 := y0
		if sy+1 < h {
			y1 = src.Pix[(sy+1)*src.Stride:]
		}
		row := dst.Pix[y*dst.Stride:]
		for x := range w * factor {
			sx := x / factor
			fx := x % factor
			nx := min(sx+1, w-1)
			top := int(y0[sx])*(factor-fx) + int(y0[nx])*fx
			bottom := int(y1[sx])*(factor-fx) + int(y1[nx])*fx
			row[x] = uint8((top*(factor-fy) + bottom*fy) / (factor * factor))
		}
	}
	return dst
}

func addMargin(src *image.Gray, margin int) *image.Gray {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	dst := image.NewGray(image.Rect(0, 0, w+2*margin, h+2*margin))
	for i := range dst.Pix {
		dst.Pix[i] = 255
	}
	for y := range h {
		copy(dst.Pix[(y+margin)*dst.Stride+margin:], src.Pix[y*src.Stride:y*src.Stride+w])
	}
	return dst
}

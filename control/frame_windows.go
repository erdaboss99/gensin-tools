package control

import "image"

// blackSampleStep is how many pixels allBlack skips between samples.
const blackSampleStep = 97

// allBlack reports whether every sampled pixel is black, which is what screen
// capture returns for an exclusive-fullscreen game.
func allBlack(src *image.RGBA) bool {
	for i := 0; i+2 < len(src.Pix); i += 4 * blackSampleStep {
		if src.Pix[i] != 0 || src.Pix[i+1] != 0 || src.Pix[i+2] != 0 {
			return false
		}
	}
	return true
}

package imaging

import (
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
	"testing"
)

func isBlank(img *image.Gray) bool {
	for _, v := range img.Pix {
		if v != img.Pix[0] {
			return false
		}
	}
	return true
}

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	return img
}

func fill(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
}

func TestFindBlobs(t *testing.T) {
	gold := color.RGBA{255, 204, 50, 255}
	img := solid(100, 40, color.RGBA{20, 20, 20, 255})
	fill(img, image.Rect(5, 5, 15, 15), gold)
	fill(img, image.Rect(30, 5, 40, 15), color.RGBA{250, 200, 55, 255}) // Within tolerance.
	fill(img, image.Rect(60, 5, 70, 15), color.RGBA{100, 200, 50, 255}) // Outside tolerance.
	fill(img, image.Rect(80, 5, 83, 8), gold)                           // A small, separate blob.

	blobs := FindBlobs(img, img.Bounds(), ColorRange{gold, 10})
	if len(blobs) != 3 {
		t.Fatalf("found %d blobs, want 3: %+v", len(blobs), blobs)
	}
	if blobs[0].Area != 100 || blobs[0].Bounds != image.Rect(5, 5, 15, 15) {
		t.Errorf("first blob = %+v", blobs[0])
	}
	if blobs[2].Area != 9 {
		t.Errorf("small blob area = %d, want 9", blobs[2].Area)
	}

	if got := FindBlobs(img, image.Rect(0, 0, 20, 40), ColorRange{gold, 10}); len(got) != 1 {
		t.Errorf("region-limited search found %d blobs, want 1", len(got))
	}
	if got := FindBlobs(img, image.Rect(500, 500, 600, 600), ColorRange{gold, 10}); got != nil {
		t.Errorf("out-of-bounds search = %+v, want nil", got)
	}
}

func TestPrepareForOCRNormalizesPolarity(t *testing.T) {
	cases := []struct {
		name         string
		bg, fg       color.RGBA
		wantBG, want uint8
	}{
		{"light text on dark", color.RGBA{30, 30, 30, 255}, color.RGBA{230, 230, 230, 255}, 255, 0},
		{"dark text on light", color.RGBA{230, 230, 230, 255}, color.RGBA{30, 30, 30, 255}, 255, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			img := solid(40, 20, tc.bg)
			fill(img, image.Rect(15, 8, 25, 12), tc.fg)

			got := PrepareForOCR(img, img.Bounds(), 2)
			if want := image.Rect(0, 0, 40*2+20, 20*2+20); got.Rect != want {
				t.Fatalf("size = %v, want %v", got.Rect, want)
			}
			if bg := got.GrayAt(12, 12).Y; bg != tc.wantBG {
				t.Errorf("background = %d, want %d", bg, tc.wantBG)
			}
			if fg := got.GrayAt(10+40, 10+20).Y; fg != tc.want {
				t.Errorf("text = %d, want %d", fg, tc.want)
			}
		})
	}
}

func TestPrepareForOCRHandlesEdgeInputs(t *testing.T) {
	flat := solid(10, 10, color.RGBA{128, 128, 128, 255})
	if got := PrepareForOCR(flat, flat.Bounds(), 2); !isBlank(got) {
		t.Error("uniform image was not left blank")
	}
	faint := solid(10, 10, color.RGBA{128, 128, 128, 255})
	fill(faint, image.Rect(2, 2, 4, 4), color.RGBA{140, 140, 140, 255})
	if got := PrepareForOCR(faint, faint.Bounds(), 2); !isBlank(got) {
		t.Error("compression-level noise was amplified")
	}
	if got := PrepareForOCR(flat, image.Rect(50, 50, 60, 60), 2); !got.Rect.Empty() {
		t.Errorf("out-of-bounds crop = %v, want empty", got.Rect)
	}
}

func TestFingerprint(t *testing.T) {
	a := solid(20, 20, color.RGBA{1, 2, 3, 255})
	b := solid(20, 20, color.RGBA{1, 2, 3, 255})
	if Fingerprint(a, a.Bounds()) != Fingerprint(b, b.Bounds()) {
		t.Error("identical images hashed differently")
	}
	fill(b, image.Rect(10, 10, 11, 11), color.RGBA{9, 9, 9, 255})
	if Fingerprint(a, a.Bounds()) == Fingerprint(b, b.Bounds()) {
		t.Error("different images hashed the same")
	}
	if Fingerprint(a, image.Rect(0, 0, 5, 5)) != Fingerprint(b, image.Rect(0, 0, 5, 5)) {
		t.Error("change outside the region affected the hash")
	}
}

func TestMinChannelSeparatesGreyTextFromSaturatedBackground(t *testing.T) {
	// Roughly equal luminance, as with the game's main stat label on its banner.
	img := solid(40, 20, color.RGBA{200, 120, 30, 255})
	fill(img, image.Rect(15, 8, 25, 12), color.RGBA{150, 150, 150, 255})
	got := PrepareForOCR(img, img.Bounds(), 1)
	if bg, fg := got.GrayAt(12, 12).Y, got.GrayAt(10+20, 10+10).Y; bg != 255 || fg != 0 {
		t.Errorf("background = %d, text = %d; want 255 and 0", bg, fg)
	}
}

func TestMeanColor(t *testing.T) {
	img := solid(10, 10, color.RGBA{10, 20, 30, 255})
	fill(img, image.Rect(0, 0, 5, 10), color.RGBA{30, 40, 50, 255})
	if got, want := MeanColor(img, img.Bounds()), (color.RGBA{20, 30, 40, 255}); got != want {
		t.Errorf("MeanColor = %v, want %v", got, want)
	}
	if got := MeanColor(img, image.Rect(50, 50, 60, 60)); got.R != 0 {
		t.Errorf("out-of-bounds MeanColor = %v", got)
	}
}

func TestVerticalSpans(t *testing.T) {
	gold := color.RGBA{230, 190, 90, 255}
	img := solid(30, 100, color.RGBA{240, 233, 220, 255})
	fill(img, image.Rect(5, 10, 20, 30), gold)
	fill(img, image.Rect(5, 40, 20, 43), gold)  // Too short to count.
	fill(img, image.Rect(5, 50, 6, 70), gold)   // Too few pixels per line.
	fill(img, image.Rect(5, 80, 20, 100), gold) // Runs off the bottom edge.
	isGold := func(r, _, _ uint8) bool { return r == 230 }

	got := VerticalSpans(img, img.Bounds(), isGold, 4, 10)
	if len(got) != 2 || got[0] != (Span{10, 30}) || got[1] != (Span{80, 100}) {
		t.Fatalf("spans = %+v", got)
	}
	if got[0].Center() != 20 || got[0].Height() != 20 {
		t.Errorf("span helpers: center %d, height %d", got[0].Center(), got[0].Height())
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.png")
	img := solid(8, 6, color.RGBA{10, 20, 30, 255})
	if err := SavePNG(path, img); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds() != img.Bounds() || got.RGBAAt(3, 3) != img.RGBAAt(3, 3) {
		t.Errorf("round trip mismatch: %v %v", got.Bounds(), got.RGBAAt(3, 3))
	}
}

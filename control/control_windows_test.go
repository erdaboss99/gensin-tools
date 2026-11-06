package control

import (
	"testing"
	"unsafe"
)

// SendInput rejects calls whose structure does not match Win32's INPUT.
func TestInputStructLayout(t *testing.T) {
	if got := unsafe.Sizeof(mouseInput{}); got != 40 {
		t.Errorf("mouseInput is %d bytes, want 40", got)
	}
	if got := unsafe.Sizeof(keyboardInput{}); got != 40 {
		t.Errorf("keyboardInput is %d bytes, want 40", got)
	}
	var mouse mouseInput
	for name, tt := range map[string]struct{ got, want uintptr }{
		"dx":        {unsafe.Offsetof(mouse.dx), 8},
		"dy":        {unsafe.Offsetof(mouse.dy), 12},
		"mouseData": {unsafe.Offsetof(mouse.mouseData), 16},
		"flags":     {unsafe.Offsetof(mouse.flags), 20},
		"time":      {unsafe.Offsetof(mouse.time), 24},
		"extra":     {unsafe.Offsetof(mouse.extra), 32},
	} {
		if tt.got != tt.want {
			t.Errorf("mouseInput.%s is at offset %d, want %d", name, tt.got, tt.want)
		}
	}
	var key keyboardInput
	for name, tt := range map[string]struct{ got, want uintptr }{
		"vk":    {unsafe.Offsetof(key.vk), 8},
		"scan":  {unsafe.Offsetof(key.scan), 10},
		"flags": {unsafe.Offsetof(key.flags), 12},
		"time":  {unsafe.Offsetof(key.time), 16},
		"extra": {unsafe.Offsetof(key.extra), 24},
	} {
		if tt.got != tt.want {
			t.Errorf("keyboardInput.%s is at offset %d, want %d", name, tt.got, tt.want)
		}
	}
}

// A zero-distance relative move changes nothing on screen, but makes Windows
// validate the structure for real.
func TestSendInputAcceptsStructure(t *testing.T) {
	if err := sendMouse(mouseInput{flags: mouseMove}); err != nil {
		t.Fatal(err)
	}
}

// Verifies the GDI capture path against the real desktop. Nothing is saved.
func TestCaptureScreen(t *testing.T) {
	img, err := captureScreen(point{0, 0}, 320, 200)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 320 || img.Bounds().Dy() != 200 {
		t.Fatalf("bounds = %v", img.Bounds())
	}
	if img.Pix[3] != 0xFF {
		t.Errorf("alpha = %d, want opaque", img.Pix[3])
	}
	if allBlack(img) {
		t.Skip("desktop is black; the session may be locked or headless")
	}
}

func TestProbeDoesNotCrash(t *testing.T) {
	t.Logf("%+v", Probe())
}

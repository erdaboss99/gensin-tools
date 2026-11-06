package scan

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/erdaboss99/gensin-scan/control"
	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
)

func TestScanArtifactsKeepsResultsWhenFocusIsLost(t *testing.T) {
	sim := &simulatedInventory{width: 1280, height: 720, grid: layout.Default().Inventory.Grid, panel: layout.Default().Inventory.Panel, artifacts: 100, failAfter: 12}
	records, err := ScanArtifacts(t.Context(), sim.walker().Walk, emptyParser(), ScanOptions{Workers: 4})
	if !errors.Is(err, control.ErrFocusLost) {
		t.Fatalf("err = %v, want ErrFocusLost", err)
	}
	if len(records) != 12 {
		t.Errorf("kept %d records, want 12", len(records))
	}
	for i, r := range records {
		if r.Index != i {
			t.Fatalf("records out of order at %d: %d", i, r.Index)
		}
	}
}

func TestScanArtifactsStopsOnOCRFailure(t *testing.T) {
	boom := errors.New("tesseract exploded")
	parser := emptyParser()
	parser.OCR = failingOCR{boom}
	sim := &simulatedInventory{width: 1280, height: 720, grid: layout.Default().Inventory.Grid, panel: layout.Default().Inventory.Panel, artifacts: 500}

	_, err := ScanArtifacts(t.Context(), sim.walker().Walk, parser, ScanOptions{Workers: 3})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the OCR failure", err)
	}
	if sim.clicks >= 500 {
		t.Errorf("walker kept going for %d clicks after OCR failed", sim.clicks)
	}
}

func TestScanArtifactsHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	emitted := 0
	source := func(ctx context.Context, emit EmitFrame) error {
		for {
			if err := emit(Frame{Index: emitted, Image: blankFrame()}); err != nil {
				return err
			}
			if emitted++; emitted == 5 {
				cancel()
			}
		}
	}
	_, err := ScanArtifacts(ctx, source, emptyParser(), ScanOptions{Workers: 2})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestScanArtifactsSavesFrames(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "frames")
	source := func(_ context.Context, emit EmitFrame) error {
		return emit(Frame{Index: 3, Source: "live:3", Image: blankFrame()})
	}
	records, err := ScanArtifacts(t.Context(), source, emptyParser(), ScanOptions{SaveFramesDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "frame-00003.png")
	if len(records) != 1 || records[0].Source != want {
		t.Fatalf("records = %+v", records)
	}
	if _, err := imaging.Load(want); err != nil {
		t.Errorf("saved frame unreadable: %v", err)
	}
}

func TestDirFrames(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.png", "a.png", "c.PNG"} {
		if err := imaging.SavePNG(filepath.Join(dir, name), blankFrame()); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("skip me"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got []string
	err := DirFrames(dir)(t.Context(), func(f Frame) error {
		got = append(got, filepath.Base(f.Source))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"a.png", "b.png", "c.PNG"}) {
		t.Errorf("order = %v", got)
	}
	if err := DirFrames(t.TempDir())(t.Context(), func(Frame) error { return nil }); err == nil {
		t.Error("empty directory did not produce an error")
	}
}

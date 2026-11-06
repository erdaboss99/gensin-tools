package layout

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWindowValidatesSize(t *testing.T) {
	for _, size := range [][2]int{{1280, 720}, {1920, 1080}, {2560, 1440}, {1920, 1200}, {1600, 1200}} {
		if _, err := NewWindow(size[0], size[1]); err != nil {
			t.Errorf("NewWindow(%v) = %v", size, err)
		}
	}
	for _, size := range [][2]int{{1024, 768}, {2560, 1080}, {3440, 1440}} {
		if _, err := NewWindow(size[0], size[1]); err == nil {
			t.Errorf("NewWindow(%v) accepted an unsupported size", size)
		}
	}
}

func TestWindowScalesAndAnchors(t *testing.T) {
	w, err := NewWindow(1920, 1080)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.X(100); got != 150 {
		t.Errorf("X(100) = %d", got)
	}
	if got := w.X(-100); got != 1770 {
		t.Errorf("X(-100) = %d", got)
	}
	if got := w.Y(-100); got != 930 {
		t.Errorf("Y(-100) = %d", got)
	}
	if got := w.Rect(Box{-400, 20, -200, -40}); got != image.Rect(1320, 30, 1620, 1020) {
		t.Errorf("Rect = %v", got)
	}
	if got := w.RowRect(RowBox{Left: 100, Right: 200, Top: -10, Bottom: 20}, 500); got != image.Rect(150, 485, 300, 530) {
		t.Errorf("RowRect = %v", got)
	}
	if got := w.ReferenceHeight(); got != 720 {
		t.Errorf("ReferenceHeight = %d", got)
	}
	if got := w.Area(100); got != 225 {
		t.Errorf("Area(100) = %d", got)
	}
}

func TestRowsPerPage(t *testing.T) {
	grid := Default().Inventory.Grid
	w720, _ := NewWindow(1280, 720)
	w800, _ := NewWindow(1280, 800)
	if got := grid.RowsPerPage(w720); got != 4 {
		t.Errorf("16:9 rows = %d, want 4", got)
	}
	if got := grid.RowsPerPage(w800); got != 5 {
		t.Errorf("16:10 rows = %d, want 5", got)
	}
}

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	write := func(t *testing.T, doc string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "layout.json")
		if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Run("partial override keeps defaults", func(t *testing.T) {
		l, err := Load(write(t, `{"inventory":{"slot":{"left":-400,"top":121,"right":-200,"bottom":145}}}`))
		if err != nil {
			t.Fatal(err)
		}
		if l.Inventory.Slot.Left != -400 {
			t.Errorf("override ignored: %+v", l.Inventory.Slot)
		}
		if l.Inventory.Grid.Columns != 8 || l.Inventory.Name != Default().Inventory.Name {
			t.Error("override clobbered defaults")
		}
	})
	t.Run("misspelled field", func(t *testing.T) {
		if _, err := Load(write(t, `{"inventory":{"slott":{}}}`)); err == nil || !strings.Contains(err.Error(), "slott") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("invalid values", func(t *testing.T) {
		if _, err := Load(write(t, `{"inventory":{"grid":{"columns":0}}}`)); err == nil {
			t.Error("zero columns accepted")
		}
	})
}

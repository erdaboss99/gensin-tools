package cli

import (
	"context"
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/layout"
	"github.com/erdaboss99/gensin-scan/ocr"
)

// engineFlags selects and configures the OCR engine and its reference data.
type engineFlags struct {
	tesseract string
	tessdata  string
	lang      string
	layout    string
	dataDir   string
}

func (f *engineFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&f.tesseract, "tesseract", "", "path to the tesseract executable (default: search PATH and standard install locations)")
	fs.StringVar(&f.tessdata, "tessdata", "", "directory holding traineddata files (default: Tesseract's own)")
	fs.StringVar(&f.lang, "lang", "eng", "traineddata name to use")
	fs.StringVar(&f.layout, "layout", "", "JSON file overriding screen coordinates")
	fs.StringVar(&f.dataDir, "data-dir", "", "directory holding update-data output (default: user cache directory)")
}

// engine is the fully loaded OCR setup.
type engine struct {
	ocr     ocr.Tesseract
	layout  layout.Layout
	catalog *catalog.Catalog
}

func (f *engineFlags) load(ctx context.Context) (engine, error) {
	var e engine
	t, err := f.newTesseract()
	if err != nil {
		return e, err
	}
	langs, err := t.Languages(ctx)
	if err != nil {
		return e, err
	}
	if !slices.Contains(langs, t.Lang) {
		return e, fmt.Errorf("tesseract has no %q language data (found: %s)", t.Lang, strings.Join(langs, ", "))
	}
	e.ocr = t
	if e.layout, err = f.loadLayout(); err != nil {
		return e, err
	}
	dir := f.dataDir
	if dir == "" {
		if dir, err = catalog.DefaultDir(); err != nil {
			dir = "" // No cache directory is available, so use the embedded snapshot.
		}
	}
	if e.catalog, err = catalog.Load(dir); err != nil {
		return e, err
	}
	return e, nil
}

func (f *engineFlags) newTesseract() (ocr.Tesseract, error) {
	path, err := ocr.Locate(f.tesseract)
	if err != nil {
		return ocr.Tesseract{}, err
	}
	return ocr.Tesseract{Path: path, Lang: f.lang, TessdataDir: f.tessdata}, nil
}

func (f *engineFlags) loadLayout() (layout.Layout, error) {
	if f.layout == "" {
		return layout.Default(), nil
	}
	return layout.Load(f.layout)
}

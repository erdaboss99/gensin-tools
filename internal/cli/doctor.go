package cli

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/erdaboss99/gensin-scan/control"
	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/layout"
)

// doctorReport prints check results and counts the failures.
type doctorReport struct {
	out      io.Writer
	problems int
}

func (r *doctorReport) check(ok bool, format string, values ...any) {
	mark := "ok  "
	if !ok {
		mark = "FAIL"
		r.problems++
	}
	fmt.Fprintf(r.out, "[%s] %s\n", mark, fmt.Sprintf(format, values...))
}

func (r *doctorReport) info(message string) {
	fmt.Fprintf(r.out, "[info] %s\n", message)
}

func (a *app) runDoctor(ctx context.Context, args []string) error {
	flags := a.newFlagSet("doctor")
	var engineOpts engineFlags
	engineOpts.register(flags)
	if err := parseFlags(flags, args); err != nil {
		return err
	}

	report := &doctorReport{out: a.stdout}
	report.checkTesseract(ctx, &engineOpts)
	_, err := engineOpts.loadLayout()
	report.check(err == nil, "screen layout: %s", describe("loaded", err))
	report.checkCatalog(engineOpts.dataDir)
	report.checkGame(control.Probe())

	if report.problems > 0 {
		return fmt.Errorf("%d check(s) failed", report.problems)
	}
	fmt.Fprintln(a.stdout, "All checks passed.")
	return nil
}

func (r *doctorReport) checkTesseract(ctx context.Context, opts *engineFlags) {
	t, err := opts.newTesseract()
	r.check(err == nil, "tesseract executable: %s", describe(t.Path, err))
	if err != nil {
		return
	}
	version, err := t.Version(ctx)
	r.check(err == nil, "tesseract version: %s", describe(version, err))
	langs, err := t.Languages(ctx)
	hasLang := err == nil && slices.Contains(langs, t.Lang)
	r.check(hasLang, "language data %q (available: %s)", t.Lang, describe(strings.Join(langs, ", "), err))
}

func (r *doctorReport) checkCatalog(dir string) {
	if dir == "" {
		dir, _ = catalog.DefaultDir()
	}
	cat, err := catalog.Load(dir)
	if err != nil {
		r.check(false, "reference data: %v", err)
		return
	}
	r.check(true, "reference data: %d artifact pieces, %d characters, %d achievements", len(cat.Pieces), len(cat.Characters), len(cat.Achievements))
}

func (r *doctorReport) checkGame(status control.Status) {
	r.check(status.Elevated, "running as Administrator (the game ignores input otherwise)")
	switch {
	case status.Minimized:
		r.info("the game is minimized; it will be restored and checked when a scan starts")
	case status.GameRunning:
		_, err := layout.NewWindow(status.Width, status.Height)
		r.check(err == nil, "game window %dx%d: %s", status.Width, status.Height, describe("supported", err))
	default:
		r.info("the game is not running, so its window could not be checked")
	}
}

func describe(value string, err error) string {
	if err != nil {
		return err.Error()
	}
	return value
}

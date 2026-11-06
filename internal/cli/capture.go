package cli

import (
	"context"
	"fmt"

	"github.com/erdaboss99/gensin-scan/internal/imaging"
)

func (a *app) runCapture(ctx context.Context, args []string) error {
	flags := a.newFlagSet("capture")
	out := flags.String("o", "screenshot.png", "file to write")
	if err := parseFlags(flags, args); err != nil {
		return err
	}
	frame, err := a.frameFor(ctx, "")
	if err != nil {
		return err
	}
	if err := imaging.SavePNG(*out, frame); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Saved %s (%dx%d).\n", *out, frame.Bounds().Dx(), frame.Bounds().Dy())
	return nil
}

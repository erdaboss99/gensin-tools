package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/erdaboss99/gensin-scan/control"
	"github.com/erdaboss99/gensin-scan/internal/layout"
)

// attachForInput connects to the game and gives the user time to let go of the mouse.
func (a *app) attachForInput(ctx context.Context, countdown time.Duration) (control.Game, error) {
	game, err := control.Attach(ctx, control.AttachOptions{NeedInput: true})
	if err != nil {
		return nil, err
	}
	width, height := game.Size()
	if _, err := layout.NewWindow(width, height); err != nil {
		return nil, err
	}
	fmt.Fprintf(a.stdout, "Attached to the game (%dx%d). Starting in %s: hands off the mouse and keyboard.\n", width, height, countdown)
	fmt.Fprintln(a.stdout, "To stop at any time, Alt-Tab away from the game. Progress so far is saved.")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(countdown):
	}
	return game, nil
}

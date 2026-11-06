package scan

import (
	"context"
	"time"
)

// Pace scales every wait between actions, for slower machines.
type Pace float64

func (p Pace) sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(time.Duration(float64(d) * float64(p)))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

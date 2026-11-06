package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/erdaboss99/gensin-scan/internal/catalog"
)

func (a *app) runUpdateData(ctx context.Context, args []string) error {
	flags := a.newFlagSet("update-data")
	out := flags.String("o", "", "directory to write to (default: user cache directory)")
	if err := parseFlags(flags, args); err != nil {
		return err
	}
	dir := *out
	if dir == "" {
		var err error
		if dir, err = catalog.DefaultDir(); err != nil {
			return errors.Join(errors.New("no user cache directory; pass -o"), err)
		}
	}
	fmt.Fprintln(a.stdout, "Downloading reference data. This takes about a minute.")
	data, err := catalog.Fetch(ctx, catalog.NewHTTPClient(), catalog.DefaultSource())
	if err != nil {
		return err
	}
	if err := catalog.Save(dir, data); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Saved %d artifact pieces, %d characters and %d achievements to %s\n",
		len(data.Pieces), len(data.Characters), len(data.Achievements), dir)
	return nil
}

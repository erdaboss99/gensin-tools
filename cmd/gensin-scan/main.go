// Command gensin-scan reads Genshin Impact artifacts and achievements from the
// game's screen using OCR and exports them for use in other tools.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/erdaboss99/gensin-scan/internal/cli"
)

// version is set at build time by the Makefile.
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.Run(ctx, version, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

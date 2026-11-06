package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
)

const usage = `gensin-scan reads Genshin Impact artifacts and achievements using OCR.

Usage:
  gensin-scan <command> [flags]

Scanning:
  artifacts     Scan artifacts from the live game or saved screenshots
  achievements  Scan achievements from the live game or saved screenshots

Exporting:
  export good    Convert an artifact scan to GOOD (Genshin Optimizer and others)
  export paimon  Convert an achievement scan to Paimon.moe
  export seelie  Convert an achievement scan to Seelie.me

Setup:
  doctor        Check that everything needed for scanning is in place
  calibrate     Draw the scan regions on a screenshot and show what OCR reads
  capture       Save a screenshot of the game window
  update-data   Download the latest artifact, character and achievement names
  version       Print the version

Run "gensin-scan <command> -h" for a command's flags.
`

// errUsage marks errors whose message has already been shown by the flag package.
var errUsage = errors.New("invalid usage")

// app holds what every command needs.
type app struct {
	version string
	stdout  io.Writer
	stderr  io.Writer
	log     *slog.Logger
}

// Run executes the command line and returns the process exit code.
func Run(ctx context.Context, version string, args []string, stdout, stderr io.Writer) int {
	a := &app{
		version: version,
		stdout:  stdout,
		stderr:  stderr,
		log:     slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}
	if err := a.dispatch(ctx, args); err != nil {
		switch {
		case errors.Is(err, flag.ErrHelp):
			return 0
		case errors.Is(err, errUsage):
			return 2
		}
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func (a *app) dispatch(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(a.stderr, usage)
		return errUsage
	}
	command, rest := args[0], args[1:]
	switch command {
	case "artifacts":
		return a.runArtifacts(ctx, rest)
	case "achievements":
		return a.runAchievements(ctx, rest)
	case "export":
		return a.runExport(rest)
	case "doctor":
		return a.runDoctor(ctx, rest)
	case "calibrate":
		return a.runCalibrate(ctx, rest)
	case "capture":
		return a.runCapture(ctx, rest)
	case "update-data":
		return a.runUpdateData(ctx, rest)
	case "version":
		fmt.Fprintf(a.stdout, "gensin-scan %s\n", a.version)
		return nil
	case "help", "-h", "--help":
		fmt.Fprint(a.stdout, usage)
		return nil
	}
	fmt.Fprintf(a.stderr, "unknown command %q\n\n%s", command, usage)
	return errUsage
}

// newFlagSet returns a flag set whose errors surface as errUsage.
func (a *app) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("gensin-scan "+name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return flag.ErrHelp
		}
		return errUsage
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(fs.Output(), "unexpected argument %q\n", fs.Arg(0))
		return errUsage
	}
	return nil
}

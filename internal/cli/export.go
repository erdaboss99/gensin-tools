package cli

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/erdaboss99/gensin-scan/internal/export"
	"github.com/erdaboss99/gensin-scan/internal/game"
	"github.com/erdaboss99/gensin-scan/internal/scan"
)

func (a *app) runExport(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(a.stderr, "usage: gensin-scan export <good|paimon|seelie> [flags]")
		return errUsage
	}
	format, rest := args[0], args[1:]
	switch format {
	case "good":
		return a.exportGOOD(rest)
	case "paimon":
		return a.exportAchievements("paimon", "paimon.json", export.PaimonMoe, rest)
	case "seelie":
		return a.exportAchievements("seelie", "seelie.json", export.SeelieMe, rest)
	}
	fmt.Fprintf(a.stderr, "unknown export format %q (want good, paimon or seelie)\n", format)
	return errUsage
}

// goodExportOptions holds the export good command's flags.
type goodExportOptions struct {
	in            string
	out           string
	includeIssues bool
}

func (o *goodExportOptions) register(flags *flag.FlagSet) {
	flags.StringVar(&o.in, "i", "artifacts.json", "artifact report to read")
	flags.StringVar(&o.out, "o", "good.json", "GOOD file to write")
	flags.BoolVar(&o.includeIssues, "include-issues", false, "keep artifacts with scan issues, as long as they pass the game's rules; these may have wrong or missing values")
}

func (a *app) exportGOOD(args []string) error {
	var opts goodExportOptions
	flags := a.newFlagSet("export good")
	opts.register(flags)
	if err := parseFlags(flags, args); err != nil {
		return err
	}

	report, err := scan.ReadReport[scan.ArtifactRecord](opts.in)
	if err != nil {
		return err
	}
	artifacts := make([]game.Artifact, 0, len(report.Items))
	skipped := 0
	for _, record := range report.Items {
		if !opts.includeIssues && len(record.Issues) > 0 {
			skipped++
			continue
		}
		artifacts = append(artifacts, record.Artifact)
	}

	doc, rejected := export.NewGOOD(artifacts)
	if err := scan.WriteJSON(opts.out, doc); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Wrote %d artifacts to %s (%d invalid and %d with issues were left out).\n",
		len(doc.Artifacts), opts.out, len(rejected), skipped)
	if !report.Complete {
		fmt.Fprintln(a.stdout, "Warning: this report comes from a scan that stopped early.")
	}
	return nil
}

// achievementExporter merges completed achievements into an export document.
type achievementExporter func(completed []game.CompletedAchievement, existing []byte) ([]byte, error)

// achievementExportOptions holds the flags of the achievement export commands.
type achievementExportOptions struct {
	in    string
	out   string
	merge string
}

func (o *achievementExportOptions) register(flags *flag.FlagSet, defaultOut string) {
	flags.StringVar(&o.in, "i", "achievements.json", "achievement report to read")
	flags.StringVar(&o.out, "o", defaultOut, "file to write")
	flags.StringVar(&o.merge, "merge", "", "the site's own export, to keep everything except achievements")
}

func (a *app) exportAchievements(name, defaultOut string, exporter achievementExporter, args []string) error {
	var opts achievementExportOptions
	flags := a.newFlagSet("export " + name)
	opts.register(flags, defaultOut)
	if err := parseFlags(flags, args); err != nil {
		return err
	}

	report, err := scan.ReadReport[scan.AchievementRecord](opts.in)
	if err != nil {
		return err
	}
	var existing []byte
	if opts.merge != "" {
		if existing, err = os.ReadFile(opts.merge); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("--merge: %w", err)
			}
			return err
		}
	}

	completed := scan.CompletedAchievements(report.Items)
	raw, err := exporter(completed, existing)
	if err != nil {
		return err
	}
	if err := os.WriteFile(opts.out, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "Wrote %d completed achievement stages to %s.\n", len(completed), opts.out)
	if !report.Complete {
		fmt.Fprintln(a.stdout, "Warning: this report comes from a scan that stopped early.")
	}
	return nil
}

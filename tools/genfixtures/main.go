// Command genfixtures draws the synthetic screenshots used by the real-OCR tests.
//
// These are mock-ups drawn at the coordinates in internal/layout. They are not game
// screenshots and contain no game art. They exercise the OCR pipeline end to end, but
// passing against them says nothing about whether the layout matches the current game
// client; only "gensin-scan calibrate" on a real screenshot can show that.
//
// Drawing text needs golang.org/x/image, because the standard library has no font
// rendering. This is a separate module so that the scanner itself stays free of
// third-party dependencies.
//
// Run it through "make fixtures".
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	out := flag.String("o", "internal/scan/testdata", "directory to write the images to")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(dir string) error {
	fonts, err := loadFonts()
	if err != nil {
		return err
	}
	for _, fixture := range inventoryFixtures {
		canvas := drawInventory(fonts, fixture)
		if err := canvas.save(filepath.Join(dir, fixture.file)); err != nil {
			return err
		}
	}
	for _, fixture := range achievementFixtures {
		canvas := drawAchievements(fonts, fixture)
		if err := canvas.save(filepath.Join(dir, fixture.file)); err != nil {
			return err
		}
	}
	return drawEmptySearch(fonts).save(filepath.Join(dir, emptySearchFile))
}

package scan

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/erdaboss99/gensin-scan/internal/imaging"
)

// Frame is one screenshot awaiting analysis.
type Frame struct {
	Index  int
	Source string
	Image  *image.RGBA
	// Duplicate marks a frame identical to the previous one, which usually means
	// a click did not register.
	Duplicate bool
}

// EmitFrame hands one frame to the pipeline.
type EmitFrame func(Frame) error

// FrameSource delivers frames to emit until it runs out or emit returns an error.
type FrameSource func(ctx context.Context, emit EmitFrame) error

// DirFrames reads every PNG and JPEG in dir in name order.
func DirFrames(dir string) FrameSource {
	return func(ctx context.Context, emit EmitFrame) error {
		paths, err := framePaths(dir)
		if err != nil {
			return err
		}
		for i, path := range paths {
			if err := ctx.Err(); err != nil {
				return err
			}
			img, err := imaging.Load(path)
			if err != nil {
				return err
			}
			if err := emit(Frame{Index: i, Source: path, Image: img}); err != nil {
				return err
			}
		}
		return nil
	}
}

func framePaths(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".png", ".jpg", ".jpeg":
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no PNG or JPEG files in %s", dir)
	}
	slices.Sort(paths)
	return paths, nil
}

package ocr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ErrNotInstalled is returned when no Tesseract executable can be found.
var ErrNotInstalled = errors.New("tesseract not found: install it or pass --tesseract")

// Tesseract runs the tesseract executable once per request.
type Tesseract struct {
	// Path is the executable to run.
	Path string
	// Lang is the traineddata name, usually "eng".
	Lang string
	// TessdataDir optionally overrides where traineddata files are loaded from.
	TessdataDir string
}

// Locate finds the tesseract executable. An explicit path wins, then PATH, then
// the default install locations.
func Locate(explicit string) (string, error) {
	if explicit != "" {
		path, err := exec.LookPath(explicit)
		if err != nil {
			return "", fmt.Errorf("--tesseract %q: %w", explicit, err)
		}
		return path, nil
	}
	if path, err := exec.LookPath("tesseract"); err == nil {
		return path, nil
	}
	for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA") + `\Programs`} {
		path := base + `\Tesseract-OCR\tesseract.exe`
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", ErrNotInstalled
}

// isBlank reports whether img has no content, so there is nothing for OCR to read.
func isBlank(img *image.Gray) bool {
	for _, v := range img.Pix {
		if v != img.Pix[0] {
			return false
		}
	}
	return true
}

// Recognize implements Recognizer.
func (t Tesseract) Recognize(ctx context.Context, req Request) (string, error) {
	if req.Image == nil || req.Image.Rect.Empty() || isBlank(req.Image) {
		return "", nil
	}
	var input bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&input, req.Image); err != nil {
		return "", fmt.Errorf("encode OCR input: %w", err)
	}

	args := []string{"stdin", "stdout", "--oem", "1", "--psm", strconv.Itoa(int(req.Mode)), "--dpi", "150", "-l", t.Lang}
	if t.TessdataDir != "" {
		args = append(args, "--tessdata-dir", t.TessdataDir)
	}
	if req.Allow != "" {
		args = append(args, "-c", "tessedit_char_whitelist="+req.Allow)
	}

	cmd := exec.CommandContext(ctx, t.Path, args...)
	// Parallelism comes from running several processes; thread pools inside
	// each one only make them compete for the same cores.
	cmd.Env = append(os.Environ(), "OMP_THREAD_LIMIT=1")
	cmd.Stdin = &input
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("run tesseract: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Version returns the first line of `tesseract --version`.
func (t Tesseract) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, t.Path, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run tesseract --version: %w", err)
	}
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(line), nil
}

// Languages returns the traineddata names Tesseract can load.
func (t Tesseract) Languages(ctx context.Context) ([]string, error) {
	args := []string{"--list-langs"}
	if t.TessdataDir != "" {
		args = append(args, "--tessdata-dir", t.TessdataDir)
	}
	out, err := exec.CommandContext(ctx, t.Path, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run tesseract --list-langs: %w", err)
	}
	var langs []string
	for line := range strings.Lines(string(out)) {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, " ") {
			langs = append(langs, line)
		}
	}
	return langs, nil
}

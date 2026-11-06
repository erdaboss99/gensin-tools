package ocr

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeModeEnv makes the test binary act as a fake tesseract.exe when it is set. The
// variable passes through to the child process along with the rest of the environment.
const fakeModeEnv = "GENSIN_FAKE_TESSERACT_MODE"

// The behaviours the fake can take on.
const (
	fakeEcho      = "echo"
	fakeFail      = "fail"
	fakeHang      = "hang"
	fakeLanguages = "languages"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeModeEnv); mode != "" {
		os.Exit(runFake(mode))
	}
	os.Exit(m.Run())
}

func runFake(mode string) int {
	switch mode {
	case fakeEcho:
		_, _ = io.Copy(io.Discard, os.Stdin)
		fmt.Printf("args: %s omp=%s\n\n\n", strings.Join(os.Args[1:], " "), os.Getenv("OMP_THREAD_LIMIT"))
	case fakeFail:
		fmt.Fprintln(os.Stderr, "bad traineddata")
		return 1
	case fakeHang:
		time.Sleep(30 * time.Second)
	case fakeLanguages:
		fmt.Println(`List of available languages in "C:\x\" (2):`)
		fmt.Println("eng")
		fmt.Println("osd")
	}
	return 0
}

// fakeTesseract returns an executable that behaves as mode describes.
func fakeTesseract(t *testing.T, mode string) string {
	t.Helper()
	t.Setenv(fakeModeEnv, mode)
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func grayWithContent() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, 4, 4))
	img.Pix[5] = 200
	return img
}

func TestRecognizePassesOptionsAndTrimsOutput(t *testing.T) {
	tess := Tesseract{Path: fakeTesseract(t, fakeEcho), Lang: "genshin_eng", TessdataDir: `C:\data`}

	got, err := tess.Recognize(t.Context(), Request{Image: grayWithContent(), Mode: ModeLine, Allow: "0123"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--psm 7", "-l genshin_eng", `--tessdata-dir C:\data`, "tessedit_char_whitelist=0123", "omp=1"} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q lacks %q", got, want)
		}
	}
	if strings.HasSuffix(got, "\n") {
		t.Errorf("output not trimmed: %q", got)
	}
}

func TestRecognizeSkipsBlankImages(t *testing.T) {
	tess := Tesseract{Path: `C:\does\not\exist.exe`, Lang: "eng"}
	blank := image.NewGray(image.Rect(0, 0, 4, 4))
	for _, img := range []*image.Gray{nil, image.NewGray(image.Rect(0, 0, 0, 0)), blank} {
		if got, err := tess.Recognize(t.Context(), Request{Image: img}); err != nil || got != "" {
			t.Errorf("Recognize = %q, %v; want no process to be started", got, err)
		}
	}
}

func TestRecognizeReportsFailure(t *testing.T) {
	_, err := Tesseract{Path: fakeTesseract(t, fakeFail), Lang: "eng"}.Recognize(t.Context(), Request{Image: grayWithContent()})
	if err == nil || !strings.Contains(err.Error(), "bad traineddata") {
		t.Errorf("err = %v, want stderr included", err)
	}
}

func TestRecognizeHonorsCancellation(t *testing.T) {
	path := fakeTesseract(t, fakeHang)
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := Tesseract{Path: path, Lang: "eng"}.Recognize(ctx, Request{Image: grayWithContent()})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 15*time.Second {
		t.Errorf("took %s, so the process wasn't killed", elapsed)
	}
}

func TestLanguages(t *testing.T) {
	langs, err := Tesseract{Path: fakeTesseract(t, fakeLanguages)}.Languages(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(langs) != 2 || langs[0] != "eng" || langs[1] != "osd" {
		t.Errorf("langs = %q", langs)
	}
}

// hideInstalledTesseract makes the machine look like it has no Tesseract.
func hideInstalledTesseract(t *testing.T) {
	t.Helper()
	empty := t.TempDir()
	for _, name := range []string{"PATH", "ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
		t.Setenv(name, empty)
	}
}

func TestLocateExplicitPath(t *testing.T) {
	if _, err := Locate(`C:\definitely\not\here.exe`); err == nil {
		t.Error("missing explicit path accepted")
	}
	path := fakeTesseract(t, fakeEcho)
	if got, err := Locate(path); err != nil || got != path {
		t.Errorf("Locate = %q, %v", got, err)
	}
}

func TestLocateReportsMissingInstall(t *testing.T) {
	hideInstalledTesseract(t)
	if _, err := Locate(""); !errors.Is(err, ErrNotInstalled) {
		t.Errorf("err = %v, want ErrNotInstalled", err)
	}
}

func TestLocateFindsDefaultInstallFolder(t *testing.T) {
	hideInstalledTesseract(t)
	programFiles := t.TempDir()
	t.Setenv("ProgramFiles", programFiles)
	want := filepath.Join(programFiles, "Tesseract-OCR", "tesseract.exe")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := Locate(""); err != nil || got != want {
		t.Errorf("Locate = %q, %v; want %q", got, err, want)
	}
}

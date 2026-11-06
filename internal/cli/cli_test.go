package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/erdaboss99/gensin-scan/internal/export"
	"github.com/erdaboss99/gensin-scan/internal/game"
	"github.com/erdaboss99/gensin-scan/internal/layout"
	"github.com/erdaboss99/gensin-scan/internal/scan"
)

func run(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = Run(t.Context(), "test", args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestExitCodes(t *testing.T) {
	tests := []struct {
		args []string
		code int
	}{
		{nil, 2},
		{[]string{"nonsense"}, 2},
		{[]string{"version"}, 0},
		{[]string{"help"}, 0},
		{[]string{"export"}, 2},
		{[]string{"export", "csv"}, 2},
		{[]string{"export", "good", "-h"}, 0},
		{[]string{"export", "good", "--no-such-flag"}, 2},
		{[]string{"export", "good", "stray"}, 2},
		{[]string{"export", "good", "-i", "/does/not/exist.json"}, 1},
		{[]string{"calibrate", "--screen", "bogus"}, 2},
	}
	for _, tt := range tests {
		if code, _, _ := run(t, tt.args...); code != tt.code {
			t.Errorf("Run(%v) = %d, want %d", tt.args, code, tt.code)
		}
	}
}

func TestExportGOODFromReport(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "artifacts.json")
	output := filepath.Join(dir, "good.json")

	valid := game.Artifact{
		SetKey: "HeartOfDepth", Slot: game.SlotFlower, Rarity: 5, Level: 20, MainStat: game.StatHP,
		Substats: []game.Substat{
			{Key: game.StatATK, Value: 19}, {Key: game.StatDEF, Value: 23},
			{Key: game.StatCritRate, Value: 3.9}, {Key: game.StatCritDMG, Value: 7.8},
		},
	}
	records := []scan.ArtifactRecord{
		{Index: 0, Artifact: valid},
		{Index: 1, Artifact: valid, Issues: []string{"substat has an implausible value"}},
		{Index: 2, Artifact: game.Artifact{Slot: game.SlotFlower}},
	}
	if err := scan.WriteJSON(input, scan.NewReport(records, false)); err != nil {
		t.Fatal(err)
	}

	count := func(extra ...string) (int, string) {
		t.Helper()
		code, stdout, stderr := run(t, append([]string{"export", "good", "-i", input, "-o", output}, extra...)...)
		if code != 0 {
			t.Fatalf("export failed: %s", stderr)
		}
		raw, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		var doc export.GOOD
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		return len(doc.Artifacts), stdout
	}

	n, stdout := count()
	if n != 1 {
		t.Errorf("default export wrote %d artifacts, want 1", n)
	}
	if !strings.Contains(stdout, "stopped early") {
		t.Error("no warning for an incomplete report")
	}
	if n, _ := count("--include-issues"); n != 2 {
		t.Errorf("--include-issues wrote %d artifacts, want 2", n)
	}
}

func TestExportAchievementsMerges(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "achievements.json")
	merge := filepath.Join(dir, "site.json")
	output := filepath.Join(dir, "paimon.json")

	records := []scan.AchievementRecord{
		{Name: "Tiered", Category: 3, IDs: []game.AchievementID{10, 11, 12}, Stars: 2, Verified: true},
		{Name: "None", Category: 3, IDs: []game.AchievementID{20}},
		{Name: "Over", Category: 4, IDs: []game.AchievementID{30}, Stars: 5},
	}
	if err := scan.WriteJSON(input, scan.NewReport(records, true)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(merge, []byte(`{"server":"Europe"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := run(t, "export", "paimon", "-i", input, "-o", output, "--merge", merge)
	if code != 0 {
		t.Fatalf("export failed: %s", stderr)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Server      string                     `json:"server"`
		Achievement map[string]map[string]bool `json:"achievement"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Server != "Europe" {
		t.Error("merge lost existing data")
	}
	want := map[string]map[string]bool{"3": {"10": true, "11": true}, "4": {"30": true}}
	if got, _ := json.Marshal(doc.Achievement); string(got) != mustJSON(t, want) {
		t.Errorf("achievement = %s", got)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestPrintLayoutRoundTrips(t *testing.T) {
	code, stdout, stderr := run(t, "calibrate", "--print-layout")
	if code != 0 {
		t.Fatalf("calibrate failed: %s", stderr)
	}
	path := filepath.Join(t.TempDir(), "layout.json")
	if err := os.WriteFile(path, []byte(stdout), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := layout.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != layout.Default() {
		t.Error("printed layout does not round-trip")
	}
}

// Setting these makes the test binary act as a fake tesseract.exe. They pass through
// to the child process along with the rest of the environment.
const (
	fakeGoodCallsEnv = "GENSIN_FAKE_TESSERACT_GOOD_CALLS"
	fakeCountFileEnv = "GENSIN_FAKE_TESSERACT_COUNT_FILE"
)

func TestMain(m *testing.M) {
	if goodCalls := os.Getenv(fakeGoodCallsEnv); goodCalls != "" {
		os.Exit(runFakeTesseract(goodCalls, os.Getenv(fakeCountFileEnv)))
	}
	os.Exit(m.Run())
}

// runFakeTesseract answers goodCalls OCR requests and then fails, as a crashed
// install or a broken language file would.
func runFakeTesseract(goodCalls, countFile string) int {
	if slices.Contains(os.Args[1:], "--list-langs") {
		fmt.Println("eng")
		return 0
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	limit, _ := strconv.Atoi(goodCalls)
	raw, _ := os.ReadFile(countFile)
	calls, _ := strconv.Atoi(string(raw))
	calls++
	if err := os.WriteFile(countFile, []byte(strconv.Itoa(calls)), 0o600); err != nil {
		return 1
	}
	if calls > limit {
		fmt.Fprintln(os.Stderr, "tesseract crashed")
		return 1
	}
	fmt.Println("text")
	return 0
}

// failingTesseract returns an executable that answers goodCalls OCR requests and then fails.
func failingTesseract(t *testing.T, goodCalls int) string {
	t.Helper()
	t.Setenv(fakeGoodCallsEnv, strconv.Itoa(goodCalls))
	t.Setenv(fakeCountFileEnv, filepath.Join(t.TempDir(), "count"))
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// artifactFrames copies test screenshots to a new directory.
func artifactFrames(t *testing.T, count int) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "scan", "testdata", "artifact_circlet_1280.png"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for i := range count {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("frame-%d.png", i)), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestFailedScanSavesPartialResults(t *testing.T) {
	frames := artifactFrames(t, 3)
	out := filepath.Join(t.TempDir(), "artifacts.json")

	// Enough for the first frame's reads, then a failure partway through the second.
	code, stdout, stderr := run(t, "artifacts", "--from-dir", frames, "--workers", "1", "--min-rarity", "1",
		"--tesseract", failingTesseract(t, 8), "-o", out)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "tesseract crashed") || !strings.Contains(stderr, "partial results saved") {
		t.Errorf("stderr = %q", stderr)
	}
	if !strings.Contains(stdout, "Scanned 1 artifacts") {
		t.Errorf("stdout = %q", stdout)
	}
	report, err := scan.ReadReport[scan.ArtifactRecord](out)
	if err != nil {
		t.Fatalf("no report was saved: %v", err)
	}
	if report.Complete || len(report.Items) != 1 {
		t.Errorf("report: complete=%v, %d items; want incomplete with 1 item", report.Complete, len(report.Items))
	}
}

func TestFailedScanKeepsEarlierReportWhenNothingWasRead(t *testing.T) {
	frames := artifactFrames(t, 1)
	out := filepath.Join(t.TempDir(), "artifacts.json")
	const earlier = "earlier report"
	if err := os.WriteFile(out, []byte(earlier), 0o600); err != nil {
		t.Fatal(err)
	}

	code, _, _ := run(t, "artifacts", "--from-dir", frames, "--tesseract", failingTesseract(t, 0), "-o", out)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != earlier {
		t.Errorf("earlier report was replaced: %q", raw)
	}
}

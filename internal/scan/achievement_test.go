package scan

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/erdaboss99/gensin-scan/control"
	"github.com/erdaboss99/gensin-scan/internal/catalog"
	"github.com/erdaboss99/gensin-scan/internal/game"
	"github.com/erdaboss99/gensin-scan/internal/imaging"
	"github.com/erdaboss99/gensin-scan/internal/layout"
)

// achievementFrame draws one medal per row, with the given number of stars, at the
// uneven spacing the real list has.
func achievementFrame(stars ...int) *image.RGBA {
	img := blankFrame()
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{240, 233, 220, 255}), image.Point{}, draw.Src)
	center := 203
	for i, count := range stars {
		medal := image.Rect(505, center-28, 565, center+28)
		draw.Draw(img, medal, image.NewUniform(color.RGBA{230, 190, 90, 255}), image.Point{}, draw.Src)
		for s := range count {
			star := image.Rect(522+s*14, center-6, 534+s*14, center+6)
			draw.Draw(img, star, image.NewUniform(color.RGBA{254, 242, 212, 255}), image.Point{}, draw.Src)
		}
		center += 83 + 27*(i%2)
	}
	return img
}

func TestFindRows(t *testing.T) {
	win, _ := layout.NewWindow(1280, 720)
	l := layout.Default().Achievements
	if got := findRows(l, win, blankFrame()); len(got) != 0 {
		t.Errorf("found rows in a frame without medals: %v", got)
	}
	if got := findRows(l, win, achievementFrame(1, 0, 3)); !slices.Equal(got, []int{203, 286, 396}) {
		t.Errorf("rows = %v", got)
	}

	// A medal cut off by the bottom of the list isn't a usable row.
	cut := achievementFrame(1)
	draw.Draw(cut, image.Rect(505, 640, 565, 660), image.NewUniform(color.RGBA{230, 190, 90, 255}), image.Point{}, draw.Src)
	if got := findRows(l, win, cut); len(got) != 1 {
		t.Errorf("rows = %v, want the clipped medal ignored", got)
	}

	l.MaxRows = 2
	if got := findRows(l, win, achievementFrame(0, 0, 0, 0)); len(got) != 2 {
		t.Errorf("rows = %v, want MaxRows respected", got)
	}
}

func TestStagesDone(t *testing.T) {
	tests := []struct {
		status string
		stars  int
		stages int
		want   int
	}{
		{"Completed", 1, 1, 1},
		{"[Completed", 1, 1, 1},
		{"Completed", 0, 3, 3},
		{"Compieted", 1, 3, 3},
		{"1/30", 1, 3, 1},
		{"0/1", 0, 1, 0},
		{"0/1", 1, 1, 0},
		{"2/100", 3, 3, 2},
		{"", 2, 3, 2},
		{"Oo”", 0, 1, 0},
		{"", 9, 3, 3},
	}
	for _, tt := range tests {
		row := resultRow{status: tt.status, stars: tt.stars}
		if got := row.stagesDone(tt.stages); got != tt.want {
			t.Errorf("status %q, %d stars, %d stages: got %d, want %d", tt.status, tt.stars, tt.stages, got, tt.want)
		}
	}
}

func TestClassify(t *testing.T) {
	entry := catalog.Achievement{Name: "To Brave the Lightning's Glow", Category: 4, IDs: []game.AchievementID{1, 2, 3}}
	lay := layout.Default().Achievements

	tests := []struct {
		name      string
		frame     *image.RGBA
		replies   []string
		ambiguous bool
		verified  bool
		review    bool
		notFound  bool
		stars     int
		title     string
	}{
		{
			name: "nothing recognizable on screen", frame: blankFrame(), review: true,
		},
		{
			name: "game says no results", frame: blankFrame(), notFound: true,
			replies: []string{`No achievements related to "To Brave the Lightning's Glow"`},
		},
		{
			name: "single match", frame: achievementFrame(1), replies: []string{"To Brave the Lightning's Glow", "1/30"},
			verified: true, stars: 1, title: "To Brave the Lightning's Glow",
		},
		{
			name: "noisy title still matches", frame: achievementFrame(0), replies: []string{"To Brave the Lightnlng's Glow", "0/1"},
			verified: true, stars: 0, title: "To Brave the Lightnlng's Glow",
		},
		{
			name: "single unreadable row counts unverified", frame: achievementFrame(2), replies: []string{"qqqq vvvv", "2/10"},
			stars: 2, title: "qqqq vvvv",
		},
		{
			name: "closest row wins, not the first close one", frame: achievementFrame(0, 3),
			replies:  []string{"Brave the Lightning's Glow", "To Brave the Lightning's Glow", "Completed"},
			verified: true, stars: 3, ambiguous: true, title: "To Brave the Lightning's Glow",
		},
		{
			name: "ambiguous name not found", frame: achievementFrame(1), replies: []string{"Something Else"},
			ambiguous: true, review: true, title: "Something Else",
		},
		{
			name: "several rows, none matching", frame: achievementFrame(1, 1), replies: []string{"Something", "Else"},
			review: true, title: "Something",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := AchievementReader{Layout: lay, OCR: &scriptedOCR{replies: tt.replies}}
			record, err := reader.Classify(t.Context(), tt.frame, entry, tt.ambiguous)
			if err != nil {
				t.Fatal(err)
			}
			if record.Verified != tt.verified || record.NeedsReview != tt.review || record.NotFound != tt.notFound ||
				record.Stars != tt.stars || record.Title != tt.title {
				t.Errorf("record = %+v, want verified=%v review=%v notFound=%v stars=%d title=%q", record, tt.verified, tt.review, tt.notFound, tt.stars, tt.title)
			}
		})
	}
}

func TestTitleScore(t *testing.T) {
	long := "Trees Should Blend Their Roots and Shade, for That Is Where the Home Is Made"
	tests := []struct {
		title, name string
		min, max    int
	}{
		{"Deja Vu!", "Déjà Vu!", 100, 100},
		{"Trees Should Blend Their Roots and Shade, for That", long, 100, 100},
		{"Fire Rat's Robe, Dragon-Head Pearl, Sacred Offering", "Fire Rat's Robe, Dragon-Head Pearl, Sacred Offering Bowl, and...", 100, 100},
		// A short title must not match the longer name that merely starts with it.
		{"Zoo Tycoon", "Zoo Tycoon Deluxe Edition With Extras", 0, 60},
		{"Brave the Lightning's Glow...", "To Brave the Lightning's Glow", 80, 99},
		{"To Brave the Lightning's Glow", "To Brave the Lightning's Glow", 100, 100},
	}
	for _, tt := range tests {
		if got := titleScore(tt.title, tt.name); got < tt.min || got > tt.max {
			t.Errorf("titleScore(%q, %q) = %d, want %d..%d", tt.title, tt.name, got, tt.min, tt.max)
		}
	}
}

func TestReadCompletedCount(t *testing.T) {
	tests := map[string]struct {
		count int
		ok    bool
	}{
		"Achievements completed: 495":  {495, true},
		"Achievements completed: 495.": {495, true},
		"495":                          {495, true},
		"Achievements completed":       {0, false},
		"":                             {0, false},
	}
	for text, want := range tests {
		reader := AchievementReader{Layout: layout.Default().Achievements, OCR: &scriptedOCR{replies: []string{text}}}
		got, ok, err := reader.ReadCompletedCount(t.Context(), blankFrame())
		if err != nil {
			t.Fatal(err)
		}
		if got != want.count || ok != want.ok {
			t.Errorf("%q: got %d, %v; want %d, %v", text, got, ok, want.count, want.ok)
		}
	}
}

// slowResults shows a blank list for a while before results appear, then changes
// once more as they fade in, the way the real list does.
type slowResults struct {
	recordingGame
	captures int
	blank    int
}

func (g *slowResults) Capture() (*image.RGBA, error) {
	g.captures++
	switch {
	case g.captures <= g.blank:
		return blankFrame(), nil
	case g.captures == g.blank+1:
		frame := achievementFrame(1)
		frame.Pix[frame.PixOffset(800, 300)] = 1 // Still fading in.
		return frame, nil
	}
	return achievementFrame(1), nil
}

func TestAwaitResultsWaitsForStableRows(t *testing.T) {
	win, _ := layout.NewWindow(1280, 720)
	g := &slowResults{blank: 6}
	w := AchievementWalker{
		Game: g, Layout: layout.Default().Achievements,
		Reader: AchievementReader{Layout: layout.Default().Achievements, OCR: &scriptedOCR{}},
	}
	frame, err := w.awaitResults(t.Context(), win)
	if err != nil {
		t.Fatal(err)
	}
	if rows := findRows(w.Layout, win, frame); len(rows) != 1 {
		t.Errorf("rows = %v, want the result that appeared late", rows)
	}
	if g.captures != 9 {
		t.Errorf("captures = %d, want 9: 6 blank, 1 fading, 2 matching", g.captures)
	}
}

func TestAwaitResultsGivesUpWhenNothingAppears(t *testing.T) {
	win, _ := layout.NewWindow(1280, 720)
	g := &slowResults{blank: 1000}
	w := AchievementWalker{
		Game: g, Layout: layout.Default().Achievements,
		Reader: AchievementReader{Layout: layout.Default().Achievements, OCR: &scriptedOCR{}},
	}
	if _, err := w.awaitResults(t.Context(), win); err != nil {
		t.Fatal(err)
	}
	if g.captures != resultPolls {
		t.Errorf("captures = %d, want %d", g.captures, resultPolls)
	}
}

func TestAwaitResultsStopsAtEmptySearchMessage(t *testing.T) {
	win, _ := layout.NewWindow(1280, 720)
	g := &slowResults{blank: 1000}
	w := AchievementWalker{
		Game: g, Layout: layout.Default().Achievements,
		Reader: AchievementReader{Layout: layout.Default().Achievements, OCR: &scriptedOCR{replies: []string{"", `No achievements related to "x"`}}},
	}
	if _, err := w.awaitResults(t.Context(), win); err != nil {
		t.Fatal(err)
	}
	if g.captures != 3 {
		t.Errorf("captures = %d, want 3: the empty message should end the wait early", g.captures)
	}
}

func TestNotFoundIsNotRetriedOrSaved(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "failures")
	g := &slowResults{blank: 1000}
	cat := catalog.New(catalog.Data{Achievements: []catalog.Achievement{{Name: "Hidden", IDs: []game.AchievementID{9}}}})
	message := `No achievements related to "Hidden"`
	records, err := AchievementWalker{
		Game: g, Layout: layout.Default().Achievements, Catalog: cat, SaveFailuresDir: dir,
		Reader: AchievementReader{Layout: layout.Default().Achievements, OCR: &scriptedOCR{replies: []string{message, message}}},
	}.Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || !records[0].NotFound || records[0].NeedsReview || records[0].Stars != 0 {
		t.Fatalf("records = %+v", records)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("saved %d frames for an expected result", len(entries))
	}
	if !keepResult(records[0]) {
		t.Error("not-found result would be searched again on resume")
	}
}

func TestUnconfirmedResultsAreRetriedAndSaved(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "failures")
	cat := catalog.New(catalog.Data{Achievements: []catalog.Achievement{
		{Name: "Alpha", IDs: []game.AchievementID{1}},
		{Name: "Alpha Beta", IDs: []game.AchievementID{2}},
	}})
	ocrReplies := &scriptedOCR{replies: []string{"Wrong", "Alpha", "Completed", "Wrong", "Wrong"}}
	records, err := AchievementWalker{
		Game: &recordingGame{}, Layout: layout.Default().Achievements, Catalog: cat, SaveFailuresDir: dir,
		Reader: AchievementReader{Layout: layout.Default().Achievements, OCR: ocrReplies},
	}.Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !records[0].Verified {
		t.Errorf("retry did not recover the first result: %+v", records[0])
	}
	if !records[1].NeedsReview || records[1].Source == "" {
		t.Fatalf("second result = %+v, want it flagged with a saved frame", records[1])
	}
	if _, err := imaging.Load(records[1].Source); err != nil {
		t.Errorf("saved frame unreadable: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("saved %d frames, want only the failure", len(entries))
	}
}

func TestResumeRescansUnverifiedResults(t *testing.T) {
	if keepResult(AchievementRecord{Verified: false}) {
		t.Error("unverified result was kept")
	}
	if keepResult(AchievementRecord{Verified: true, NeedsReview: true}) {
		t.Error("result needing review was kept")
	}
	if !keepResult(AchievementRecord{Verified: true}) {
		t.Error("confirmed result was discarded")
	}
}

func TestAchievementRecordCompleted(t *testing.T) {
	record := AchievementRecord{Category: 7, IDs: []game.AchievementID{10, 11, 12}, Stars: 2}
	want := []game.CompletedAchievement{{ID: 10, Category: 7}, {ID: 11, Category: 7}}
	if got := record.Completed(); !slices.Equal(got, want) {
		t.Errorf("Completed = %+v", got)
	}
	record.Stars = 9
	if got := record.Completed(); len(got) != 3 {
		t.Errorf("Completed exceeded the number of stages: %+v", got)
	}
}

// recordingGame logs input for the achievement walker tests.
type recordingGame struct {
	simulatedInventory
	log       []string
	failAfter int
}

func (g *recordingGame) Size() (int, int) { return 1280, 720 }

func (g *recordingGame) Capture() (*image.RGBA, error) { return achievementFrame(1), nil }

func (g *recordingGame) Click(int, int) error {
	g.log = append(g.log, "click")
	return nil
}

func (g *recordingGame) Type(text string) error {
	if g.failAfter > 0 && strings.Count(strings.Join(g.log, " "), "type:") >= g.failAfter {
		return control.ErrFocusLost
	}
	g.log = append(g.log, "type:"+text)
	return nil
}

func (g *recordingGame) PressEnter() error {
	g.log = append(g.log, "enter")
	return nil
}

func achievementCatalog() *catalog.Catalog {
	return catalog.New(catalog.Data{Achievements: []catalog.Achievement{
		{Name: "Alpha", IDs: []game.AchievementID{1}},
		{Name: "Beta", IDs: []game.AchievementID{2}},
		{Name: "Gamma", IDs: []game.AchievementID{3}},
	}})
}

func TestAchievementWalkerSequence(t *testing.T) {
	g := &recordingGame{}
	cat := achievementCatalog()
	records, err := AchievementWalker{
		Game: g, Layout: layout.Default().Achievements, Catalog: cat, Limit: 2,
		Reader: AchievementReader{Layout: layout.Default().Achievements, OCR: &scriptedOCR{replies: []string{"Alpha", "Completed", "Beta", "Completed"}}},
	}.Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || !records[0].Verified {
		t.Errorf("records = %+v", records)
	}
	want := []string{"click", "click", "type:Alpha", "enter", "click", "click", "type:Beta", "enter", "click", "click"}
	if !slices.Equal(g.log, want) {
		t.Errorf("input log = %v", g.log)
	}
}

func TestAchievementWalkerResumes(t *testing.T) {
	g := &recordingGame{failAfter: 1}
	cat := achievementCatalog()
	walker := AchievementWalker{
		Game: g, Layout: layout.Default().Achievements, Catalog: cat,
		Reader: AchievementReader{Layout: layout.Default().Achievements, OCR: &scriptedOCR{replies: []string{"Alpha", "Completed", "Beta", "Completed", "Gamma", "Completed"}}},
	}
	records, err := walker.Run(t.Context())
	if !errors.Is(err, control.ErrFocusLost) {
		t.Fatalf("err = %v, want ErrFocusLost", err)
	}
	if len(records) != 1 || records[0].Name != "Alpha" {
		t.Fatalf("records = %+v", records)
	}

	// Results needing review are searched again; confirmed ones are not.
	records = append(records, AchievementRecord{Name: "Gamma", IDs: []game.AchievementID{3}, NeedsReview: true})
	g.failAfter, g.log = 0, nil
	walker.Done = records
	final, err := walker.Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var typed []string
	for _, entry := range g.log {
		if name, ok := strings.CutPrefix(entry, "type:"); ok {
			typed = append(typed, name)
		}
	}
	if !slices.Equal(typed, []string{"Beta", "Gamma"}) {
		t.Errorf("typed = %v", typed)
	}
	if len(final) != 3 {
		t.Errorf("final = %+v", final)
	}
}

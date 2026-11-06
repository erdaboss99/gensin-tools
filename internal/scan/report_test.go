package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erdaboss99/gensin-scan/internal/game"
)

func TestReportRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "report.json")
	records := []ArtifactRecord{{Index: 1, Artifact: game.Artifact{Slot: game.SlotGoblet, Rarity: 5}, Issues: []string{"x"}}}
	if err := WriteJSON(path, NewReport(records, true)); err != nil {
		t.Fatal(err)
	}
	report, err := ReadReport[ArtifactRecord](path)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Complete || len(report.Items) != 1 || report.Items[0].Artifact.Slot != game.SlotGoblet {
		t.Errorf("report = %+v", report)
	}

	if err := os.WriteFile(path, []byte(`{"version":99}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadReport[ArtifactRecord](path); err == nil {
		t.Error("future report version accepted")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("temporary files left behind: %v", entries)
	}
}

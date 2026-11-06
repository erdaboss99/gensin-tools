package scan

import (
	"fmt"

	"github.com/erdaboss99/gensin-scan/internal/game"
)

// RawArtifactText preserves what OCR read, for diagnosing bad results.
type RawArtifactText struct {
	Name     string   `json:"name"`
	Slot     string   `json:"slot"`
	MainStat string   `json:"mainStat,omitempty"`
	Level    string   `json:"level"`
	Substats []string `json:"substats"`
	Equipped string   `json:"equipped,omitempty"`
}

// ArtifactRecord is the result of scanning one frame.
type ArtifactRecord struct {
	Index    int           `json:"index"`
	Source   string        `json:"source"`
	Artifact game.Artifact `json:"artifact"`
	// Issues are problems that may make the artifact wrong.
	Issues []string `json:"issues,omitempty"`
	// Corrections are OCR errors that were confidently repaired.
	Corrections []string        `json:"corrections,omitempty"`
	Raw         RawArtifactText `json:"raw"`
}

// Exportable reports whether the artifact is complete and nothing went wrong while
// reading it. An issue can mean a missing substat on an otherwise valid artifact.
func (r ArtifactRecord) Exportable() bool {
	return len(r.Issues) == 0 && r.Artifact.Validate() == nil
}

func (r *ArtifactRecord) issue(format string, args ...any) {
	r.Issues = append(r.Issues, fmt.Sprintf(format, args...))
}

func (r *ArtifactRecord) correction(format string, args ...any) {
	r.Corrections = append(r.Corrections, fmt.Sprintf(format, args...))
}

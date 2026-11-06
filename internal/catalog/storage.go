package catalog

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed data/*.json
var embedded embed.FS

const (
	piecesFile       = "artifacts.json"
	charactersFile   = "characters.json"
	achievementsFile = "achievements.json"
)

// DefaultDir returns where update-data stores fresh data by default.
func DefaultDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "gensin-scan"), nil
}

// Load reads each data file from dir when it exists there, falling back to the
// embedded snapshot. An empty dir uses only the snapshot.
func Load(dir string) (*Catalog, error) {
	var data Data
	if err := readData(dir, piecesFile, &data.Pieces); err != nil {
		return nil, err
	}
	if err := readData(dir, charactersFile, &data.Characters); err != nil {
		return nil, err
	}
	if err := readData(dir, achievementsFile, &data.Achievements); err != nil {
		return nil, err
	}
	return New(data), nil
}

func readData[T any](dir, name string, out *T) error {
	var raw []byte
	var err error
	if dir != "" {
		raw, err = os.ReadFile(filepath.Join(dir, name))
	}
	if dir == "" || errors.Is(err, fs.ErrNotExist) {
		raw, err = embedded.ReadFile("data/" + name)
	}
	if err != nil {
		return fmt.Errorf("read catalog %s: %w", name, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("parse catalog %s: %w", name, err)
	}
	return nil
}

// Save writes data to dir, creating it if needed.
func Save(dir string, data Data) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return errors.Join(
		writeData(dir, piecesFile, data.Pieces),
		writeData(dir, charactersFile, data.Characters),
		writeData(dir, achievementsFile, data.Achievements),
	)
}

func writeData[T any](dir, name string, value T) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	return os.WriteFile(filepath.Join(dir, name), append(raw, '\n'), 0o644)
}

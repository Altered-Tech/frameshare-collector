package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultDir returns the directory a confirmed Profile should be saved
// to by default: <home>/FrameShare/profiles. This differs from
// cmd/collector's -out flag (a CLI tool run from a terminal has a
// well-understood working directory); cmd/gui is typically launched by
// double-click or as a Steam game, where the working directory isn't
// predictable or user-facing, so it needs a fixed, discoverable location
// instead.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, "FrameShare", "profiles"), nil
}

// Save writes p as indented JSON to a timestamped file under dir
// (creating dir if it doesn't exist) and returns the path written to.
// The filename is derived from CollectedAt rather than the current
// time, matching cmd/collector's hardware-snapshot-<timestamp>.json
// convention, so re-saving the same collection (e.g. after further
// edits) overwrites the same file instead of accumulating duplicates.
func Save(p *Profile, dir string) (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal profile: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	filename := fmt.Sprintf("profile-%s.json", p.CollectedAt.Format("20060102-150405"))
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

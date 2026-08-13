package library

import (
	"os"
	"path/filepath"
)

// isSteamRoot reports whether path looks like a real Steam install
// directory rather than just an empty/nonexistent one, by checking for its
// steamapps subdirectory.
func isSteamRoot(path string) bool {
	info, err := os.Stat(filepath.Join(path, "steamapps"))
	return err == nil && info.IsDir()
}

package library

import (
	"context"
	"os"
	"path/filepath"
)

// steamRoot returns macOS's single default Steam install location, or ""
// if Steam isn't installed there.
func steamRoot(ctx context.Context) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	root := filepath.Join(home, "Library", "Application Support", "Steam")
	if !isSteamRoot(root) {
		return ""
	}
	return root
}

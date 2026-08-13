package library

import (
	"context"
	"os"
	"path/filepath"
)

// linuxSteamRootCandidates are default Steam install locations across the
// common Linux packaging methods, checked in order: native/.deb install
// (~/.local/share/Steam, or the older ~/.steam/steam layout), Flatpak, and
// Snap.
var linuxSteamRootCandidates = []string{
	".steam/steam",
	".local/share/Steam",
	".var/app/com.valvesoftware.Steam/.local/share/Steam",
	"snap/steam/common/.steam/steam",
}

func steamRoot(ctx context.Context) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, candidate := range linuxSteamRootCandidates {
		root := filepath.Join(home, candidate)
		if isSteamRoot(root) {
			return root
		}
	}
	return ""
}

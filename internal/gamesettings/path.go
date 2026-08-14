package gamesettings

import (
	"fmt"
	"path/filepath"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

// protonAppDataPath resolves the path a Windows-only Steam title's config
// would live at under its Proton prefix on Linux: Steam runs the title in
// a per-app "compatdata" prefix that mirrors a fresh Windows user profile,
// so anything the title would normally write to
// "%LOCALAPPDATA%\...\rel" ends up here instead. rel is that same
// LOCALAPPDATA-relative suffix.
//
// game.InstallPath is "<library>/steamapps/common/<installdir>" (see
// library.parseAppManifest); compatdata is a sibling of common under the
// same steamapps directory.
func protonAppDataPath(game library.Game, rel string) (string, error) {
	if game.AppID == "" {
		return "", fmt.Errorf("no Steam app id for %s, can't locate its Proton prefix", game.Name)
	}
	steamapps := filepath.Dir(filepath.Dir(game.InstallPath))
	if filepath.Base(steamapps) != "steamapps" {
		return "", fmt.Errorf("unexpected install path %q for %s", game.InstallPath, game.Name)
	}
	return filepath.Join(steamapps, "compatdata", game.AppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", rel), nil
}

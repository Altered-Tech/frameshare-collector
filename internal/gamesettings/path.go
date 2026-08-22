package gamesettings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

// ProtonAppDataPath resolves the path a Windows-only Steam title's config
// would live at under its Proton prefix on Linux: Steam runs the title in
// a per-app "compatdata" prefix that mirrors a fresh Windows user profile,
// so anything the title would normally write to
// "%LOCALAPPDATA%\...\rel" ends up here instead. rel is that same
// LOCALAPPDATA-relative suffix.
//
// game.InstallPath is "<library>/steamapps/common/<installdir>" (see
// library.parseAppManifest); compatdata is a sibling of common under the
// same steamapps directory. Engine subpackages (e.g. unreal) call this for
// titles that don't ship a native Linux build.
func ProtonAppDataPath(game library.Game, rel string) (string, error) {
	steamapps, err := steamappsDir(game)
	if err != nil {
		return "", err
	}
	return filepath.Join(steamapps, "compatdata", game.AppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", rel), nil
}

// steamappsDir resolves the steamapps directory game.InstallPath lives
// under, shared by ProtonAppDataPath and ProtonVersion.
func steamappsDir(game library.Game) (string, error) {
	if game.AppID == "" {
		return "", fmt.Errorf("no Steam app id for %s, can't locate its Proton prefix", game.Name)
	}
	steamapps := filepath.Dir(filepath.Dir(game.InstallPath))
	if filepath.Base(steamapps) != "steamapps" {
		return "", fmt.Errorf("unexpected install path %q for %s", game.InstallPath, game.Name)
	}
	return steamapps, nil
}

// ProtonVersion best-effort resolves the Proton compatibility tool Steam
// has mapped to game's app id, for titles whose settings were actually
// resolved from inside a Proton prefix. configPath is the config file
// path a Parser resolved for game (see gamesettings.Collect) -- it only
// counts as Proton-run if configPath falls under that prefix, i.e. was
// produced by ProtonAppDataPath. That check matters because a compatdata
// prefix, and a CompatToolMapping entry, can exist for a title even when
// it wasn't used: Steam lets a user force a compatibility tool override
// on a title that also has a native Linux build (e.g. Terraria), which
// would otherwise make this report a Proton version for a profile that
// was actually read from the title's native config path.
//
// It returns "" -- never an error -- when configPath isn't under a
// Proton prefix, or the mapping can't be read: a title's Proton version
// is an annotation on its GameProfile, not something settings collection
// should fail over.
//
// The version comes from Steam's own per-app compatibility tool
// selection: config/config.vdf's CompatToolMapping/<appid>/name (e.g.
// "proton_experimental", "proton_9") -- the same identifier Steam itself
// uses to pick which Proton build runs the title, so it's an unambiguous
// (if not especially pretty) answer to "which Proton produced this
// prefix". config.vdf lives at the Steam root, a sibling of steamapps.
func ProtonVersion(game library.Game, configPath string) string {
	steamapps, err := steamappsDir(game)
	if err != nil {
		return ""
	}
	prefix := filepath.Join(steamapps, "compatdata", game.AppID) + string(filepath.Separator)
	if !strings.HasPrefix(configPath, prefix) {
		return ""
	}

	root := filepath.Dir(steamapps)
	data, err := os.ReadFile(filepath.Join(root, "config", "config.vdf"))
	if err != nil {
		return ""
	}
	parsed, err := library.ParseVDF(data)
	if err != nil {
		return ""
	}
	return compatToolMappingName(parsed, game.AppID)
}

// compatToolMappingName walks a parsed config.vdf down to
// CompatToolMapping/<appID>/name, returning "" if any step along the way
// is missing or not the shape expected.
func compatToolMappingName(parsed map[string]any, appID string) string {
	node := parsed
	for _, key := range []string{"InstallConfigStore", "Software", "Valve", "Steam", "CompatToolMapping"} {
		next, ok := node[key].(map[string]any)
		if !ok {
			return ""
		}
		node = next
	}
	entry, ok := node[appID].(map[string]any)
	if !ok {
		return ""
	}
	name, _ := entry["name"].(string)
	return name
}

package library

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// detectSteamLibraries locates the Steam install (if any) and returns every
// library folder it knows about, each with its installed titles. A missing
// Steam install is not an error: it returns a nil slice.
func detectSteamLibraries(ctx context.Context) ([]Library, error) {
	root := steamRoot(ctx)
	if root == "" {
		return nil, nil
	}

	paths, err := steamLibraryPaths(root)
	if err != nil {
		return nil, err
	}

	var libs []Library
	for _, p := range paths {
		games, err := steamGamesIn(p)
		if err != nil {
			// An unreadable library folder (e.g. on a disconnected
			// external drive) shouldn't take down detection of the
			// libraries that are still reachable.
			continue
		}
		libs = append(libs, Library{Source: SourceSteam, Path: p, Games: games})
	}
	return libs, nil
}

// steamLibraryPaths reads steamapps/libraryfolders.vdf under the Steam root
// to find every library folder Steam knows about, including the root
// install itself (which libraryfolders.vdf always lists as one of its own
// entries). Falls back to just the root if the file doesn't exist yet,
// which happens on a fresh install with no games added.
func steamLibraryPaths(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{root}, nil
		}
		return nil, err
	}

	parsed, err := ParseVDF(data)
	if err != nil {
		return nil, err
	}
	folders, _ := parsed["libraryfolders"].(map[string]any)

	seen := map[string]bool{}
	var paths []string
	for _, v := range folders {
		entry, ok := v.(map[string]any)
		if !ok {
			continue
		}
		path, _ := entry["path"].(string)
		path = filepath.Clean(path)
		if path == "." || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return []string{root}, nil
	}
	return paths, nil
}

// steamGamesIn parses every appmanifest_*.acf in a library folder's
// steamapps directory into a Game. Manifests that fail to parse are
// skipped rather than failing the whole library.
func steamGamesIn(libraryPath string) ([]Game, error) {
	steamappsDir := filepath.Join(libraryPath, "steamapps")
	entries, err := os.ReadDir(steamappsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var games []Game
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "appmanifest_") || !strings.HasSuffix(name, ".acf") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(steamappsDir, name))
		if err != nil {
			continue
		}
		if game, ok := parseAppManifest(data, steamappsDir); ok {
			games = append(games, game)
		}
	}
	return games, nil
}

// parseAppManifest extracts the fields we care about from an
// appmanifest_<appid>.acf file's "AppState" object. installDir in the
// manifest is relative to <steamappsDir>/common; it's resolved to an
// absolute path here.
func parseAppManifest(data []byte, steamappsDir string) (Game, bool) {
	parsed, err := ParseVDF(data)
	if err != nil {
		return Game{}, false
	}
	state, ok := parsed["AppState"].(map[string]any)
	if !ok {
		return Game{}, false
	}

	name, _ := state["name"].(string)
	installDir, _ := state["installdir"].(string)
	if name == "" || installDir == "" {
		return Game{}, false
	}
	appID, _ := state["appid"].(string)
	sizeBytes, _ := strconv.ParseUint(fromVDFString(state["SizeOnDisk"]), 10, 64)

	return Game{
		AppID:       appID,
		Name:        name,
		InstallPath: filepath.Join(steamappsDir, "common", installDir),
		SizeBytes:   sizeBytes,
	}, true
}

func fromVDFString(v any) string {
	s, _ := v.(string)
	return s
}

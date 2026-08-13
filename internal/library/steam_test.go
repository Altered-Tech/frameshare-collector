package library

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates path's parent directories and writes contents to it.
func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestSteamLibraryPaths(t *testing.T) {
	root := t.TempDir()
	secondLibrary := t.TempDir()

	writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), `"libraryfolders"
{
	"0"
	{
		"path"		"`+escapeVDFPath(root)+`"
	}
	"1"
	{
		"path"		"`+escapeVDFPath(secondLibrary)+`"
	}
}
`)

	got, err := steamLibraryPaths(root)
	if err != nil {
		t.Fatalf("steamLibraryPaths() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("steamLibraryPaths() = %v, want 2 entries", got)
	}
	gotSet := map[string]bool{got[0]: true, got[1]: true}
	if !gotSet[filepath.Clean(root)] || !gotSet[filepath.Clean(secondLibrary)] {
		t.Errorf("steamLibraryPaths() = %v, want %v and %v", got, root, secondLibrary)
	}
}

func TestSteamLibraryPathsNoManifestFile(t *testing.T) {
	root := t.TempDir()
	got, err := steamLibraryPaths(root)
	if err != nil {
		t.Fatalf("steamLibraryPaths() error = %v", err)
	}
	if len(got) != 1 || got[0] != root {
		t.Errorf("steamLibraryPaths() = %v, want [%v]", got, root)
	}
}

func TestSteamGamesIn(t *testing.T) {
	libraryPath := t.TempDir()
	writeFile(t, filepath.Join(libraryPath, "steamapps", "appmanifest_228980.acf"), `"AppState"
{
	"appid"		"228980"
	"name"		"Steamworks Common Redistributables"
	"installdir"		"Steamworks Shared"
	"SizeOnDisk"		"7130886"
}
`)
	// A manifest missing required fields should be skipped, not fail the
	// whole library.
	writeFile(t, filepath.Join(libraryPath, "steamapps", "appmanifest_bad.acf"), `"AppState"
{
	"appid"		"1"
}
`)

	games, err := steamGamesIn(libraryPath)
	if err != nil {
		t.Fatalf("steamGamesIn() error = %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("steamGamesIn() = %v, want 1 game", games)
	}
	want := Game{
		AppID:       "228980",
		Name:        "Steamworks Common Redistributables",
		InstallPath: filepath.Join(libraryPath, "steamapps", "common", "Steamworks Shared"),
		SizeBytes:   7130886,
	}
	if games[0] != want {
		t.Errorf("steamGamesIn()[0] = %#v, want %#v", games[0], want)
	}
}

func TestSteamGamesInNoSteamapps(t *testing.T) {
	games, err := steamGamesIn(t.TempDir())
	if err != nil {
		t.Fatalf("steamGamesIn() error = %v", err)
	}
	if games != nil {
		t.Errorf("steamGamesIn() = %v, want nil", games)
	}
}

// escapeVDFPath mirrors how Steam's VDF writer escapes backslashes so a
// filesystem path can be embedded in a VDF fixture string.
func escapeVDFPath(path string) string {
	escaped := ""
	for _, r := range path {
		if r == '\\' {
			escaped += `\\`
		} else {
			escaped += string(r)
		}
	}
	return escaped
}

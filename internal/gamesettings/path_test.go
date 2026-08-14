package gamesettings

import (
	"path/filepath"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

func TestProtonAppDataPath(t *testing.T) {
	steamLibrary := t.TempDir()
	installPath := filepath.Join(steamLibrary, "steamapps", "common", "Stray")
	game := library.Game{Name: "Stray", InstallPath: installPath, AppID: "1332010"}

	got, err := ProtonAppDataPath(game, filepath.Join("Stray", "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini"))
	if err != nil {
		t.Fatalf("ProtonAppDataPath() error = %v", err)
	}
	want := filepath.Join(steamLibrary, "steamapps", "compatdata", "1332010", "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Stray", "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini")
	if got != want {
		t.Errorf("ProtonAppDataPath() = %q, want %q", got, want)
	}
}

func TestProtonAppDataPathNoAppID(t *testing.T) {
	game := library.Game{Name: "Stray", InstallPath: filepath.Join(t.TempDir(), "steamapps", "common", "Stray")}
	if _, err := ProtonAppDataPath(game, "irrelevant"); err == nil {
		t.Error("ProtonAppDataPath() error = nil, want error for missing app id")
	}
}

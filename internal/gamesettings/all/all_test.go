package all

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

// TestCollectEveryRegisteredTitle is an end-to-end check that every title
// package's init() actually registers with gamesettings via this
// package's blank imports, by running gamesettings.Collect against a
// title from each subpackage (Team Fortress 2's config path is
// install-relative, so it's the only one of the three we can fully
// exercise without mocking a per-OS home directory).
func TestCollectEveryRegisteredTitle(t *testing.T) {
	installPath := t.TempDir()
	cfgPath := filepath.Join(installPath, "tf", "cfg", "video.txt")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte(`"config" { "setting.defaultres" "1920" "setting.defaultresheight" "1080" }`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	game := library.Game{AppID: "440", Name: "Team Fortress 2", InstallPath: installPath}
	if !gamesettings.Supported(game, library.SourceSteam) {
		t.Fatal("Supported() = false, want true once internal/gamesettings/all is imported")
	}

	profile, err := gamesettings.Collect(game, library.SourceSteam)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if profile.Settings.Display.Resolution.WidthPx != 1920 {
		t.Errorf("Settings.Display.Resolution.WidthPx = %d, want 1920", profile.Settings.Display.Resolution.WidthPx)
	}

	for _, appID := range []string{"1332010" /* Stray */, "105600" /* Terraria */} {
		if !gamesettings.Supported(library.Game{AppID: appID}, library.SourceSteam) {
			t.Errorf("Supported() = false for app id %s, want true", appID)
		}
	}
}

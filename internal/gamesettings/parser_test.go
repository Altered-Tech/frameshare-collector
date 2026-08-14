package gamesettings

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

func TestCollectTeamFortress2(t *testing.T) {
	installPath := t.TempDir()
	cfgPath := filepath.Join(installPath, "tf", "cfg", "video.txt")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte(tf2VideoTxt), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	game := library.Game{AppID: tf2AppID, Name: "Team Fortress 2", InstallPath: installPath}
	if !Supported(game, library.SourceSteam) {
		t.Fatal("Supported() = false, want true for a registered app id")
	}

	profile, err := Collect(game, library.SourceSteam)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if profile.ConfigPath != cfgPath {
		t.Errorf("ConfigPath = %q, want %q", profile.ConfigPath, cfgPath)
	}
	if profile.Name != "Team Fortress 2" || profile.AppID != tf2AppID || profile.Source != string(library.SourceSteam) {
		t.Errorf("profile = %#v, want matching Name/AppID/Source", profile)
	}
	if profile.Settings.Display.Resolution.WidthPx != 1920 {
		t.Errorf("Settings.Display.Resolution.WidthPx = %d, want 1920", profile.Settings.Display.Resolution.WidthPx)
	}
	if profile.ParserVersion != ParserVersion {
		t.Errorf("ParserVersion = %q, want %q", profile.ParserVersion, ParserVersion)
	}
}

func TestCollectUnsupportedTitle(t *testing.T) {
	game := library.Game{AppID: "999999999", Name: "Some Unsupported Game"}
	if Supported(game, library.SourceSteam) {
		t.Fatal("Supported() = true, want false for an unregistered app id")
	}

	_, err := Collect(game, library.SourceSteam)
	if !errors.Is(err, ErrTitleUnsupported) {
		t.Errorf("Collect() error = %v, want ErrTitleUnsupported", err)
	}
}

func TestCollectMissingConfigFile(t *testing.T) {
	game := library.Game{AppID: tf2AppID, Name: "Team Fortress 2", InstallPath: t.TempDir()}
	if _, err := Collect(game, library.SourceSteam); err == nil {
		t.Error("Collect() error = nil, want error when the config file was never written")
	}
}

package gamesettings

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

// fakeParser is a minimal Parser used to test Register/Collect/Supported
// without depending on any real title package (those live in
// subpackages -- unreal, source, standalone -- that import this package,
// so this package can't import them back).
type fakeParser struct {
	appID string
}

func (p fakeParser) Matches(game library.Game, source library.Source) bool {
	return source == library.SourceSteam && game.AppID == p.appID
}

func (fakeParser) ConfigPath(game library.Game) (string, error) {
	return filepath.Join(game.InstallPath, "settings.json"), nil
}

func (fakeParser) Parse(data []byte) (TitleSettings, error) {
	return TitleSettings{GraphicsPreset: string(data)}, nil
}

func TestRegisterCollectSupported(t *testing.T) {
	const appID = "999999901"
	Register(fakeParser{appID: appID})

	installPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(installPath, "settings.json"), []byte("High"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	game := library.Game{AppID: appID, Name: "Fake Game", InstallPath: installPath}
	if !Supported(game, library.SourceSteam) {
		t.Fatal("Supported() = false, want true for a registered app id")
	}

	profile, err := Collect(game, library.SourceSteam)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if profile.ConfigPath != filepath.Join(installPath, "settings.json") {
		t.Errorf("ConfigPath = %q, want %q", profile.ConfigPath, filepath.Join(installPath, "settings.json"))
	}
	if profile.Name != "Fake Game" || profile.AppID != appID || profile.Source != string(library.SourceSteam) {
		t.Errorf("profile = %#v, want matching Name/AppID/Source", profile)
	}
	if profile.Settings.GraphicsPreset != "High" {
		t.Errorf("Settings.GraphicsPreset = %q, want %q", profile.Settings.GraphicsPreset, "High")
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
	const appID = "999999902"
	Register(fakeParser{appID: appID})

	game := library.Game{AppID: appID, Name: "Fake Game", InstallPath: t.TempDir()}
	if _, err := Collect(game, library.SourceSteam); err == nil {
		t.Error("Collect() error = nil, want error when the config file was never written")
	}
}

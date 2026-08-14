package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

func testLibraries() []library.Library {
	return []library.Library{
		{
			Source: library.SourceSteam,
			Path:   "/games/SteamLibrary",
			Games: []library.Game{
				{AppID: "1", Name: "Subnautica", InstallPath: "/games/SteamLibrary/steamapps/common/Subnautica"},
				{AppID: "2", Name: "Hades", InstallPath: "/games/SteamLibrary/steamapps/common/Hades"},
			},
		},
	}
}

func TestNumberedGamePicker(t *testing.T) {
	var out bytes.Buffer
	entries := gameEntries(testLibraries())

	idx, err := numberedGamePicker(entries, strings.NewReader("2\n"), &out)
	if err != nil {
		t.Fatalf("numberedGamePicker() error = %v", err)
	}
	if entries[idx].game.Name != "Hades" {
		t.Errorf("numberedGamePicker() game = %q, want Hades", entries[idx].game.Name)
	}
	if entries[idx].source != library.SourceSteam {
		t.Errorf("numberedGamePicker() source = %q, want %q", entries[idx].source, library.SourceSteam)
	}
	if !strings.Contains(out.String(), "1) Subnautica") || !strings.Contains(out.String(), "2) Hades") {
		t.Errorf("numberedGamePicker() output = %q, want both games listed", out.String())
	}
}

func TestNumberedGamePickerInvalidSelection(t *testing.T) {
	entries := gameEntries(testLibraries())
	cases := []string{"0\n", "3\n", "abc\n", "\n"}
	for _, input := range cases {
		var out bytes.Buffer
		if _, err := numberedGamePicker(entries, strings.NewReader(input), &out); err == nil {
			t.Errorf("numberedGamePicker(%q) error = nil, want error", input)
		}
	}
}

func TestGameEntriesEmpty(t *testing.T) {
	if got := gameEntries(nil); got != nil {
		t.Errorf("gameEntries(nil) = %v, want nil", got)
	}
}

func TestPickGameNoGames(t *testing.T) {
	if _, _, err := pickGame(nil, nil, nil); err == nil {
		t.Error("pickGame(nil libs) error = nil, want error for empty library list")
	}
}

func TestCollectGameSettingsSupportedTitle(t *testing.T) {
	installPath := t.TempDir()
	cfgPath := filepath.Join(installPath, "tf", "cfg", "video.txt")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	video := `"config" { "setting.defaultres" "1920" "setting.defaultresheight" "1080" "setting.fullscreen" "1" }`
	if err := os.WriteFile(cfgPath, []byte(video), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	game := library.Game{AppID: "440", Name: "Team Fortress 2", InstallPath: installPath}
	profile := collectGameSettings(game, library.SourceSteam)
	if profile == nil {
		t.Fatal("collectGameSettings() = nil, want a profile for a registered title with a config file")
	}
	if profile.Settings.Display.Resolution.WidthPx != 1920 {
		t.Errorf("Settings.Display.Resolution.WidthPx = %d, want 1920", profile.Settings.Display.Resolution.WidthPx)
	}
}

func TestCollectGameSettingsUnsupportedTitle(t *testing.T) {
	game := library.Game{AppID: "0", Name: "Some Unsupported Game"}
	if profile := collectGameSettings(game, library.SourceSteam); profile != nil {
		t.Errorf("collectGameSettings() = %#v, want nil for a title with no registered parser", profile)
	}
}

func TestCollectGameSettingsMissingConfigFile(t *testing.T) {
	game := library.Game{AppID: "440", Name: "Team Fortress 2", InstallPath: t.TempDir()}
	if profile := collectGameSettings(game, library.SourceSteam); profile != nil {
		t.Errorf("collectGameSettings() = %#v, want nil when the config file was never written", profile)
	}
}

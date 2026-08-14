package gamesettings

import (
	"path/filepath"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

// strayGameUserSettings is a trimmed fixture matching the keys confirmed
// against a real install in https://github.com/Altered-Tech/frameshare-collector/issues/3.
const strayGameUserSettings = `[ScalabilityGroups]
sg.EffectsQuality=1
sg.TextureQuality=1
sg.ShadowQuality=1
sg.ResolutionQuality=100.000000
sg.ViewDistanceQuality=3
sg.PostProcessQuality=3
sg.FoliageQuality=3
sg.ShadingQuality=3

[/Script/Hk_project.HKGameUserSettings]
bUseVSync=True
ResolutionSizeX=2560
ResolutionSizeY=1440
FullscreenMode=1
FrameRateLimit=60.000000
bUseHDRDisplayOutput=False
`

func TestParseUnrealGameUserSettings(t *testing.T) {
	got, err := parseUnrealGameUserSettings([]byte(strayGameUserSettings))
	if err != nil {
		t.Fatalf("parseUnrealGameUserSettings() error = %v", err)
	}

	want := TitleSettings{
		Display: DisplaySettings{
			Resolution:     Resolution{WidthPx: 2560, HeightPx: 1440},
			WindowMode:     WindowBorderless,
			VSync:          true,
			FrameRateLimit: 60,
			HDR:            false,
		},
		Detail: DetailSettings{
			TextureQuality: "Medium",
			ShadowQuality:  "Medium",
			ViewDistance:   "Epic",
			FoliageDensity: "Epic",
			EffectsQuality: "Medium",
		},
	}
	if got != want {
		t.Errorf("parseUnrealGameUserSettings() = %#v, want %#v", got, want)
	}
}

func TestUnrealScalabilityLabel(t *testing.T) {
	cases := map[string]string{"0": "Low", "1": "Medium", "2": "High", "3": "Epic", "": "", "garbage": ""}
	for raw, want := range cases {
		if got := unrealScalabilityLabel(raw); got != want {
			t.Errorf("unrealScalabilityLabel(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestUnrealWindowMode(t *testing.T) {
	cases := map[string]WindowMode{"0": WindowFullscreen, "1": WindowBorderless, "2": WindowWindowed, "garbage": ""}
	for raw, want := range cases {
		if got := unrealWindowMode(raw); got != want {
			t.Errorf("unrealWindowMode(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestProtonAppDataPath(t *testing.T) {
	steamLibrary := t.TempDir()
	installPath := filepath.Join(steamLibrary, "steamapps", "common", "Stray")
	game := library.Game{Name: "Stray", InstallPath: installPath, AppID: "1332010"}

	got, err := protonAppDataPath(game, filepath.Join("Stray", "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini"))
	if err != nil {
		t.Fatalf("protonAppDataPath() error = %v", err)
	}
	want := filepath.Join(steamLibrary, "steamapps", "compatdata", "1332010", "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Stray", "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini")
	if got != want {
		t.Errorf("protonAppDataPath() = %q, want %q", got, want)
	}
}

func TestProtonAppDataPathNoAppID(t *testing.T) {
	game := library.Game{Name: "Stray", InstallPath: filepath.Join(t.TempDir(), "steamapps", "common", "Stray")}
	if _, err := protonAppDataPath(game, "irrelevant"); err == nil {
		t.Error("protonAppDataPath() error = nil, want error for missing app id")
	}
}

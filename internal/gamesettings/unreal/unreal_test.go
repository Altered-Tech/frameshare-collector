package unreal

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

func TestParseINI(t *testing.T) {
	input := `; leading comment
GlobalKey=GlobalValue

[ScalabilityGroups]
sg.EffectsQuality=1
sg.TextureQuality=1
# another comment style
sg.ResolutionQuality=100.000000

[/Script/Hk_project.HKGameUserSettings]
ResolutionSizeX=1920
ResolutionSizeY=1080
`
	got := parseINI([]byte(input))
	want := map[string]map[string]string{
		"": {"GlobalKey": "GlobalValue"},
		"ScalabilityGroups": {
			"sg.EffectsQuality":    "1",
			"sg.TextureQuality":    "1",
			"sg.ResolutionQuality": "100.000000",
		},
		"/Script/Hk_project.HKGameUserSettings": {
			"ResolutionSizeX": "1920",
			"ResolutionSizeY": "1080",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseINI() = %#v, want %#v", got, want)
	}
}

func TestParseINIDuplicateKeyKeepsLast(t *testing.T) {
	got := parseINI([]byte("[S]\nk=first\nk=second\n"))
	if got["S"]["k"] != "second" {
		t.Errorf(`parseINI() section "S" key "k" = %q, want "second"`, got["S"]["k"])
	}
}

func TestSectionWithSuffix(t *testing.T) {
	sections := map[string]map[string]string{
		"ScalabilityGroups":                     {"a": "1"},
		"/Script/Hk_project.HKGameUserSettings": {"b": "2"},
	}

	got, ok := sectionWithSuffix(sections, "GameUserSettings")
	if !ok || got["b"] != "2" {
		t.Errorf("sectionWithSuffix() = %#v, %v, want the HKGameUserSettings section", got, ok)
	}

	if _, ok := sectionWithSuffix(sections, "NoSuchSuffix"); ok {
		t.Errorf("sectionWithSuffix() ok = true for a suffix with no match")
	}
}

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

func TestParseGameUserSettings(t *testing.T) {
	got, err := parseGameUserSettings([]byte(strayGameUserSettings))
	if err != nil {
		t.Fatalf("parseGameUserSettings() error = %v", err)
	}

	want := gamesettings.TitleSettings{
		Display: gamesettings.DisplaySettings{
			Resolution:     gamesettings.Resolution{WidthPx: 2560, HeightPx: 1440},
			WindowMode:     gamesettings.WindowBorderless,
			VSync:          true,
			FrameRateLimit: 60,
			HDR:            false,
		},
		Detail: gamesettings.DetailSettings{
			TextureQuality: "Medium",
			ShadowQuality:  "Medium",
			ViewDistance:   "Epic",
			FoliageDensity: "Epic",
			EffectsQuality: "Medium",
		},
	}
	if got != want {
		t.Errorf("parseGameUserSettings() = %#v, want %#v", got, want)
	}
}

func TestScalabilityLabel(t *testing.T) {
	cases := map[string]string{"0": "Low", "1": "Medium", "2": "High", "3": "Epic", "": "", "garbage": ""}
	for raw, want := range cases {
		if got := scalabilityLabel(raw); got != want {
			t.Errorf("scalabilityLabel(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestWindowMode(t *testing.T) {
	cases := map[string]gamesettings.WindowMode{
		"0": gamesettings.WindowFullscreen, "1": gamesettings.WindowBorderless, "2": gamesettings.WindowWindowed, "garbage": "",
	}
	for raw, want := range cases {
		if got := windowMode(raw); got != want {
			t.Errorf("windowMode(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestStrayMatches(t *testing.T) {
	p := strayParser{}
	if !p.Matches(library.Game{AppID: strayAppID}, library.SourceSteam) {
		t.Error("Matches() = false, want true for Stray's app id")
	}
	if p.Matches(library.Game{AppID: "0"}, library.SourceSteam) {
		t.Error("Matches() = true, want false for a different app id")
	}
}

func TestConfigPathLinuxUsesProtonPrefix(t *testing.T) {
	steamLibrary := t.TempDir()
	installPath := filepath.Join(steamLibrary, "steamapps", "common", "Stray")
	game := library.Game{Name: "Stray", InstallPath: installPath, AppID: strayAppID}

	got, err := gamesettings.ProtonAppDataPath(game, filepath.Join(strayConfigFolder, "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini"))
	if err != nil {
		t.Fatalf("ProtonAppDataPath() error = %v", err)
	}
	want := filepath.Join(steamLibrary, "steamapps", "compatdata", strayAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", strayConfigFolder, "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini")
	if got != want {
		t.Errorf("ProtonAppDataPath() = %q, want %q", got, want)
	}
}

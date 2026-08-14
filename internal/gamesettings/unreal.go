package gamesettings

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

// Unreal Engine writes every title's graphics settings to a
// GameUserSettings.ini at the same relative path under a per-platform
// config root, and uses the same key names across titles: a
// "[ScalabilityGroups]" section of shared sg.* quality tiers, plus a
// title-specific "[/Script/<Studio>.<Studio>GameUserSettings]" section
// (see iniSectionWithSuffix) for resolution/window/frame-rate. These
// helpers implement that shared layout once; a title's Parser (e.g.
// stray.go) only needs to supply its own display name.

// unrealConfigPath resolves a title's GameUserSettings.ini path.
// folderName is the name Unreal saves the title's config under, which is
// usually -- but not guaranteed to be -- the title's Steam display name
// (it comes from the project's own configured name).
func unrealConfigPath(folderName string, game library.Game) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Preferences", folderName, "MacNoEditor", "GameUserSettings.ini"), nil
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", fmt.Errorf("%%LOCALAPPDATA%% is not set")
		}
		return filepath.Join(localAppData, folderName, "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini"), nil
	case "linux":
		// Most Unreal Engine titles on Steam ship Windows-only builds
		// run through Proton, which land config under the title's
		// compat prefix rather than a native ~/.config path.
		rel := filepath.Join(folderName, "Saved", "Config", "WindowsNoEditor", "GameUserSettings.ini")
		return protonAppDataPath(game, rel)
	default:
		return "", fmt.Errorf("unsupported platform %q for Unreal Engine config lookup", runtime.GOOS)
	}
}

// parseUnrealGameUserSettings extracts TitleSettings from a
// GameUserSettings.ini file's contents, using the key names Unreal Engine
// itself defines (so this applies to any UE title, not just the one that
// registered it).
func parseUnrealGameUserSettings(data []byte) (TitleSettings, error) {
	sections := parseINI(data)
	scalability := sections["ScalabilityGroups"]
	main, _ := iniSectionWithSuffix(sections, "GameUserSettings")

	var settings TitleSettings
	settings.Display.Resolution = Resolution{
		WidthPx:  iniInt(main["ResolutionSizeX"]),
		HeightPx: iniInt(main["ResolutionSizeY"]),
	}
	settings.Display.WindowMode = unrealWindowMode(main["FullscreenMode"])
	settings.Display.VSync = iniBool(main["bUseVSync"])
	settings.Display.FrameRateLimit = int(iniFloat(main["FrameRateLimit"]))
	settings.Display.HDR = iniBool(main["bUseHDRDisplayOutput"])

	settings.Detail.TextureQuality = unrealScalabilityLabel(scalability["sg.TextureQuality"])
	settings.Detail.ShadowQuality = unrealScalabilityLabel(scalability["sg.ShadowQuality"])
	settings.Detail.ViewDistance = unrealScalabilityLabel(scalability["sg.ViewDistanceQuality"])
	settings.Detail.FoliageDensity = unrealScalabilityLabel(scalability["sg.FoliageQuality"])
	settings.Detail.EffectsQuality = unrealScalabilityLabel(scalability["sg.EffectsQuality"])

	return settings, nil
}

// unrealScalabilityLabel maps a "sg.*" scalability group's 0-3 tier to
// Unreal's own names for those tiers (Engine/Config/BaseScalability.ini),
// which is what the title's own settings menu shows the player.
func unrealScalabilityLabel(raw string) string {
	switch raw {
	case "0":
		return "Low"
	case "1":
		return "Medium"
	case "2":
		return "High"
	case "3":
		return "Epic"
	default:
		return ""
	}
}

// unrealWindowMode maps Unreal's EWindowMode enum, as stored in
// FullscreenMode, to WindowMode.
func unrealWindowMode(raw string) WindowMode {
	switch raw {
	case "0":
		return WindowFullscreen
	case "1":
		return WindowBorderless // "WindowedFullscreen" in EWindowMode
	case "2":
		return WindowWindowed
	default:
		return ""
	}
}

func iniInt(raw string) int {
	v, _ := strconv.Atoi(raw)
	return v
}

func iniFloat(raw string) float64 {
	v, _ := strconv.ParseFloat(raw, 64)
	return v
}

func iniBool(raw string) bool {
	v, _ := strconv.ParseBool(raw)
	return v
}

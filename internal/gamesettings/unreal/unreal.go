// Package unreal implements gamesettings.Parser for titles built on
// Unreal Engine. Every UE title writes its graphics settings to a
// GameUserSettings.ini at the same relative path under a per-platform
// config root, using the same key names: a "[ScalabilityGroups]" section
// of shared sg.* quality tiers, plus a title-specific
// "[/Script/<Studio>.<Studio>GameUserSettings]" section for
// resolution/window/frame-rate. This file implements that shared layout
// once; a title's Parser (see stray.go) only supplies its own app id and
// the folder name Unreal saves its config under.
package unreal

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

// configPath resolves a title's GameUserSettings.ini path. folderName is
// the name Unreal saves the title's config under, which is usually -- but
// not guaranteed to be -- the title's Steam display name (it comes from
// the project's own configured name).
func configPath(folderName string, game library.Game) (string, error) {
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
		return gamesettings.ProtonAppDataPath(game, rel)
	default:
		return "", fmt.Errorf("unsupported platform %q for Unreal Engine config lookup", runtime.GOOS)
	}
}

// parseGameUserSettings extracts TitleSettings from a
// GameUserSettings.ini file's contents, using the key names Unreal Engine
// itself defines (so this applies to any UE title, not just the one that
// calls it).
func parseGameUserSettings(data []byte) (gamesettings.TitleSettings, error) {
	sections := parseINI(data)
	scalability := sections["ScalabilityGroups"]
	main, _ := sectionWithSuffix(sections, "GameUserSettings")

	var settings gamesettings.TitleSettings
	settings.Display.Resolution = gamesettings.Resolution{
		WidthPx:  iniInt(main["ResolutionSizeX"]),
		HeightPx: iniInt(main["ResolutionSizeY"]),
	}
	settings.Display.WindowMode = windowMode(main["FullscreenMode"])
	settings.Display.VSync = iniBool(main["bUseVSync"])
	settings.Display.FrameRateLimit = int(iniFloat(main["FrameRateLimit"]))
	settings.Display.HDR = iniBool(main["bUseHDRDisplayOutput"])

	settings.Detail.TextureQuality = scalabilityLabel(scalability["sg.TextureQuality"])
	settings.Detail.ShadowQuality = scalabilityLabel(scalability["sg.ShadowQuality"])
	settings.Detail.ViewDistance = scalabilityLabel(scalability["sg.ViewDistanceQuality"])
	settings.Detail.FoliageDensity = scalabilityLabel(scalability["sg.FoliageQuality"])
	settings.Detail.EffectsQuality = scalabilityLabel(scalability["sg.EffectsQuality"])

	return settings, nil
}

// scalabilityLabel maps a "sg.*" scalability group's 0-3 tier to Unreal's
// own names for those tiers (Engine/Config/BaseScalability.ini), which is
// what the title's own settings menu shows the player.
func scalabilityLabel(raw string) string {
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

// windowMode maps Unreal's EWindowMode enum, as stored in FullscreenMode,
// to gamesettings.WindowMode.
func windowMode(raw string) gamesettings.WindowMode {
	switch raw {
	case "0":
		return gamesettings.WindowFullscreen
	case "1":
		return gamesettings.WindowBorderless // "WindowedFullscreen" in EWindowMode
	case "2":
		return gamesettings.WindowWindowed
	default:
		return ""
	}
}

// parseINI parses a minimal INI file: `[Section]` headers followed by
// `key=value` lines, as used by GameUserSettings.ini. Comment lines
// starting with `;` or `#` and blank lines are skipped. Keys that appear
// before the first section header are stored under the "" section.
// Duplicate keys within a section keep the last value seen, matching how
// Unreal itself reads its own INI files.
func parseINI(data []byte) map[string]map[string]string {
	sections := map[string]map[string]string{"": {}}
	section := ""

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if _, ok := sections[section]; !ok {
				sections[section] = map[string]string{}
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		sections[section][strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return sections
}

// sectionWithSuffix returns the first section whose name ends with
// suffix, along with whether one was found. Unreal names a title's main
// settings section after its studio-specific GameUserSettings subclass
// (e.g. "/Script/Hk_project.HKGameUserSettings"), which varies per title,
// but it always ends in "GameUserSettings" -- matching on that suffix
// avoids hardcoding the title-specific prefix.
func sectionWithSuffix(sections map[string]map[string]string, suffix string) (map[string]string, bool) {
	for name, kv := range sections {
		if strings.HasSuffix(name, suffix) {
			return kv, true
		}
	}
	return nil, false
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

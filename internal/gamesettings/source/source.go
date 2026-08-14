// Package source implements gamesettings.Parser for titles built on
// Valve's Source engine. Every Source title writes its video settings to
// a cfg/video.txt under its own install directory, in Valve's KeyValues
// ("VDF") format, using the same setting.* key names (confirmed against
// real Team Fortress 2 and CS:GO files -- see teamfortress2.go). This
// file implements that shared layout once; a title's Parser only supplies
// its own app id and mod directory name (e.g. "tf" for Team Fortress 2).
package source

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

// configPath returns the install-relative path to modDir's video.txt.
// Unlike Unreal Engine titles, Source stores this under the game's own
// install directory rather than a per-OS user-data path, so no
// darwin/windows/linux branching (or Proton prefix lookup) is needed.
func configPath(game library.Game, modDir string) (string, error) {
	return filepath.Join(game.InstallPath, modDir, "cfg", "video.txt"), nil
}

// parseVideoTxt extracts TitleSettings from a video.txt file's contents.
func parseVideoTxt(data []byte) (gamesettings.TitleSettings, error) {
	parsed, err := library.ParseVDF(data)
	if err != nil {
		return gamesettings.TitleSettings{}, fmt.Errorf("decoding video.txt: %w", err)
	}
	cfg, ok := firstMapValue(parsed)
	if !ok {
		return gamesettings.TitleSettings{}, fmt.Errorf("video.txt has no settings object")
	}

	var settings gamesettings.TitleSettings
	settings.Display.Resolution = gamesettings.Resolution{
		WidthPx:  atoi(str(cfg, "setting.defaultres")),
		HeightPx: atoi(str(cfg, "setting.defaultresheight")),
	}
	switch {
	case str(cfg, "setting.nowindowborder") == "1":
		settings.Display.WindowMode = gamesettings.WindowBorderless
	case str(cfg, "setting.fullscreen") == "1":
		settings.Display.WindowMode = gamesettings.WindowFullscreen
	default:
		settings.Display.WindowMode = gamesettings.WindowWindowed
	}
	settings.Display.VSync = str(cfg, "setting.mat_vsync") == "1"
	settings.Detail.AntiAliasing = antiAliasingLabel(str(cfg, "setting.mat_antialias"))
	settings.GraphicsPreset = qualityLabel(str(cfg, "setting.gpu_level"))

	return settings, nil
}

// antiAliasingLabel maps mat_antialias, which Source stores as the MSAA
// sample count directly (0, 2, 4, 6, 8...), to a label.
func antiAliasingLabel(raw string) string {
	if raw == "" {
		return ""
	}
	if raw == "0" {
		return "Off"
	}
	return fmt.Sprintf("%sx MSAA", raw)
}

// qualityLabel maps gpu_level's 0-3 tiers to the labels Source's "Overall
// Video Quality" preset uses for them.
func qualityLabel(raw string) string {
	switch raw {
	case "0":
		return "Low"
	case "1":
		return "Medium"
	case "2":
		return "High"
	case "3":
		return "Very High"
	default:
		return ""
	}
}

// firstMapValue returns the single nested object a VDF file's root key
// holds (its name varies -- "VideoConfig", "config" -- so callers match on
// shape, not name).
func firstMapValue(m map[string]any) (map[string]any, bool) {
	for _, v := range m {
		if child, ok := v.(map[string]any); ok {
			return child, true
		}
	}
	return nil, false
}

// str reads a string value from a parsed VDF object, returning "" for a
// missing or non-string key.
func str(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func atoi(raw string) int {
	v, _ := strconv.Atoi(raw)
	return v
}

package gamesettings

import (
	"fmt"
	"path/filepath"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

// tf2AppID is Team Fortress 2's Steam app id (store.steampowered.com/app/440).
const tf2AppID = "440"

func init() {
	register(teamFortress2Parser{})
}

// teamFortress2Parser reads Team Fortress 2's cfg/video.txt, a Valve
// KeyValues file the Source engine writes under the title's own install
// directory (unlike Stray/Terraria, no per-OS user-data path is needed).
type teamFortress2Parser struct{}

func (teamFortress2Parser) Matches(game library.Game, source library.Source) bool {
	return source == library.SourceSteam && game.AppID == tf2AppID
}

func (teamFortress2Parser) ConfigPath(game library.Game) (string, error) {
	return filepath.Join(game.InstallPath, "tf", "cfg", "video.txt"), nil
}

func (teamFortress2Parser) Parse(data []byte) (TitleSettings, error) {
	parsed, err := library.ParseVDF(data)
	if err != nil {
		return TitleSettings{}, fmt.Errorf("decoding video.txt: %w", err)
	}
	cfg, ok := firstMapValue(parsed)
	if !ok {
		return TitleSettings{}, fmt.Errorf("video.txt has no settings object")
	}

	var settings TitleSettings
	settings.Display.Resolution = Resolution{
		WidthPx:  iniInt(vdfString(cfg, "setting.defaultres")),
		HeightPx: iniInt(vdfString(cfg, "setting.defaultresheight")),
	}
	switch {
	case vdfString(cfg, "setting.nowindowborder") == "1":
		settings.Display.WindowMode = WindowBorderless
	case vdfString(cfg, "setting.fullscreen") == "1":
		settings.Display.WindowMode = WindowFullscreen
	default:
		settings.Display.WindowMode = WindowWindowed
	}
	settings.Display.VSync = vdfString(cfg, "setting.mat_vsync") == "1"
	settings.Detail.AntiAliasing = sourceAntiAliasingLabel(vdfString(cfg, "setting.mat_antialias"))
	settings.GraphicsPreset = sourceQualityLabel(vdfString(cfg, "setting.gpu_level"))

	return settings, nil
}

// sourceAntiAliasingLabel maps mat_antialias, which the Source engine
// stores as the MSAA sample count directly (0, 2, 4, 6, 8...), to a label.
func sourceAntiAliasingLabel(raw string) string {
	if raw == "" {
		return ""
	}
	if raw == "0" {
		return "Off"
	}
	return fmt.Sprintf("%sx MSAA", raw)
}

// sourceQualityLabel maps gpu_level's 0-3 tiers to the labels the Source
// engine's "Overall Video Quality" preset uses for them.
func sourceQualityLabel(raw string) string {
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

// vdfString reads a string value from a parsed VDF object, returning ""
// for a missing or non-string key.
func vdfString(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

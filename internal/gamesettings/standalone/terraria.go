// Package standalone holds gamesettings.Parser implementations for titles
// that don't share a config format/engine with any other supported title
// (yet) -- so, unlike unreal or source, there's no shared per-engine file
// here. If a second title ever turns up using the same format as one of
// these, split it into its own package the way unreal/source did.
package standalone

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

// terrariaAppID is Terraria's Steam app id (store.steampowered.com/app/105600).
const terrariaAppID = "105600"

func init() {
	gamesettings.Register(terrariaParser{})
}

// terrariaParser reads Terraria's config.json, a plain JSON file FNA
// writes to a per-platform user data directory (not under the Steam
// install path).
type terrariaParser struct{}

func (terrariaParser) Matches(game library.Game, source library.Source) bool {
	return source == library.SourceSteam && game.AppID == terrariaAppID
}

func (terrariaParser) ConfigPath(game library.Game) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "Documents", "My Games", "Terraria", "config.json"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Terraria", "config.json"), nil
	case "linux":
		return filepath.Join(home, ".local", "share", "Terraria", "config.json"), nil
	default:
		return "", fmt.Errorf("unsupported platform %q for Terraria config lookup", runtime.GOOS)
	}
}

// terrariaConfig covers the config.json fields this parser maps to
// TitleSettings; see https://terraria.wiki.gg/wiki/Config.json_settings
// for the full (larger) key set.
type terrariaConfig struct {
	DisplayWidth     int  `json:"DisplayWidth"`
	DisplayHeight    int  `json:"DisplayHeight"`
	Fullscreen       bool `json:"Fullscreen"`
	WindowBorderless bool `json:"WindowBorderless"`
	GraphicsQuality  int  `json:"GraphicsQuality"`
}

func (terrariaParser) Parse(data []byte) (gamesettings.TitleSettings, error) {
	var cfg terrariaConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return gamesettings.TitleSettings{}, fmt.Errorf("decoding Terraria config.json: %w", err)
	}

	var settings gamesettings.TitleSettings
	settings.Display.Resolution = gamesettings.Resolution{WidthPx: cfg.DisplayWidth, HeightPx: cfg.DisplayHeight}
	switch {
	case cfg.WindowBorderless:
		settings.Display.WindowMode = gamesettings.WindowBorderless
	case cfg.Fullscreen:
		settings.Display.WindowMode = gamesettings.WindowFullscreen
	default:
		settings.Display.WindowMode = gamesettings.WindowWindowed
	}
	settings.GraphicsPreset = terrariaQualityLabel(cfg.GraphicsQuality)

	return settings, nil
}

// terrariaQualityLabel maps GraphicsQuality's 0-3 tiers to the labels
// Terraria's own video settings menu uses for them (Off/Low/Medium/High).
func terrariaQualityLabel(v int) string {
	switch v {
	case 0:
		return "Off"
	case 1:
		return "Low"
	case 2:
		return "Medium"
	case 3:
		return "High"
	default:
		return ""
	}
}

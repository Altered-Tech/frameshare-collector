package standalone

import (
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

func TestTerrariaMatches(t *testing.T) {
	p := terrariaParser{}
	if !p.Matches(library.Game{AppID: terrariaAppID}, library.SourceSteam) {
		t.Error("Matches() = false, want true for Terraria's app id")
	}
	if p.Matches(library.Game{AppID: "0"}, library.SourceSteam) {
		t.Error("Matches() = true, want false for a different app id")
	}
}

func TestTerrariaParserParse(t *testing.T) {
	input := `{
  "DisplayWidth": 1920,
  "DisplayHeight": 1080,
  "Fullscreen": true,
  "WindowBorderless": false,
  "GraphicsQuality": 3
}`
	got, err := terrariaParser{}.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	want := gamesettings.TitleSettings{
		Display: gamesettings.DisplaySettings{
			Resolution: gamesettings.Resolution{WidthPx: 1920, HeightPx: 1080},
			WindowMode: gamesettings.WindowFullscreen,
		},
		GraphicsPreset: "High",
	}
	if got != want {
		t.Errorf("Parse() = %#v, want %#v", got, want)
	}
}

func TestTerrariaParserParseBorderless(t *testing.T) {
	input := `{"DisplayWidth":1280,"DisplayHeight":720,"Fullscreen":false,"WindowBorderless":true,"GraphicsQuality":0}`
	got, err := terrariaParser{}.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Display.WindowMode != gamesettings.WindowBorderless {
		t.Errorf("Display.WindowMode = %q, want %q", got.Display.WindowMode, gamesettings.WindowBorderless)
	}
	if got.GraphicsPreset != "Off" {
		t.Errorf("GraphicsPreset = %q, want %q", got.GraphicsPreset, "Off")
	}
}

func TestTerrariaParserParseInvalidJSON(t *testing.T) {
	if _, err := (terrariaParser{}).Parse([]byte("not json")); err == nil {
		t.Error("Parse() error = nil, want error for invalid JSON")
	}
}

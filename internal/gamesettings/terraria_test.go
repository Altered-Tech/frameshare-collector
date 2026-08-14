package gamesettings

import "testing"

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
	want := TitleSettings{
		Display: DisplaySettings{
			Resolution: Resolution{WidthPx: 1920, HeightPx: 1080},
			WindowMode: WindowFullscreen,
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
	if got.Display.WindowMode != WindowBorderless {
		t.Errorf("Display.WindowMode = %q, want %q", got.Display.WindowMode, WindowBorderless)
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

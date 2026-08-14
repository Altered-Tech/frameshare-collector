package source

import (
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
)

// videoTxtFixture mirrors the real KeyValues shape Source-engine games
// write to cfg/video.txt (confirmed against a public CS:GO example; Team
// Fortress 2 uses the same engine and key names).
const videoTxtFixture = `"config"
{
	"setting.defaultres"		"1920"
	"setting.defaultresheight"		"1080"
	"setting.fullscreen"		"1"
	"setting.nowindowborder"		"0"
	"setting.mat_vsync"		"0"
	"setting.mat_antialias"		"4"
	"setting.gpu_level"		"2"
}
`

func TestParseVideoTxt(t *testing.T) {
	got, err := parseVideoTxt([]byte(videoTxtFixture))
	if err != nil {
		t.Fatalf("parseVideoTxt() error = %v", err)
	}
	want := gamesettings.TitleSettings{
		Display: gamesettings.DisplaySettings{
			Resolution: gamesettings.Resolution{WidthPx: 1920, HeightPx: 1080},
			WindowMode: gamesettings.WindowFullscreen,
			VSync:      false,
		},
		Detail:         gamesettings.DetailSettings{AntiAliasing: "4x MSAA"},
		GraphicsPreset: "High",
	}
	if got != want {
		t.Errorf("parseVideoTxt() = %#v, want %#v", got, want)
	}
}

func TestParseVideoTxtBorderless(t *testing.T) {
	input := `"config" { "setting.nowindowborder" "1" "setting.fullscreen" "0" "setting.mat_antialias" "0" }`
	got, err := parseVideoTxt([]byte(input))
	if err != nil {
		t.Fatalf("parseVideoTxt() error = %v", err)
	}
	if got.Display.WindowMode != gamesettings.WindowBorderless {
		t.Errorf("Display.WindowMode = %q, want %q", got.Display.WindowMode, gamesettings.WindowBorderless)
	}
	if got.Detail.AntiAliasing != "Off" {
		t.Errorf("Detail.AntiAliasing = %q, want %q", got.Detail.AntiAliasing, "Off")
	}
}

func TestParseVideoTxtInvalid(t *testing.T) {
	if _, err := parseVideoTxt([]byte("not vdf")); err == nil {
		t.Error("parseVideoTxt() error = nil, want error for malformed video.txt")
	}
}

func TestFirstMapValue(t *testing.T) {
	m := map[string]any{"config": map[string]any{"a": "1"}}
	got, ok := firstMapValue(m)
	if !ok || got["a"] != "1" {
		t.Errorf("firstMapValue() = %#v, %v, want the nested map", got, ok)
	}

	if _, ok := firstMapValue(map[string]any{"a": "not a map"}); ok {
		t.Error("firstMapValue() ok = true for a root with no nested object")
	}
}

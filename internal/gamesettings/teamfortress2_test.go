package gamesettings

import "testing"

// tf2VideoTxt mirrors the real KeyValues shape Source-engine games write
// to cfg/video.txt (confirmed against a public CS:GO example; TF2 uses the
// same engine and key names).
const tf2VideoTxt = `"config"
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

func TestTeamFortress2ParserParse(t *testing.T) {
	got, err := teamFortress2Parser{}.Parse([]byte(tf2VideoTxt))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	want := TitleSettings{
		Display: DisplaySettings{
			Resolution: Resolution{WidthPx: 1920, HeightPx: 1080},
			WindowMode: WindowFullscreen,
			VSync:      false,
		},
		Detail:         DetailSettings{AntiAliasing: "4x MSAA"},
		GraphicsPreset: "High",
	}
	if got != want {
		t.Errorf("Parse() = %#v, want %#v", got, want)
	}
}

func TestTeamFortress2ParserParseBorderless(t *testing.T) {
	input := `"config" { "setting.nowindowborder" "1" "setting.fullscreen" "0" "setting.mat_antialias" "0" }`
	got, err := teamFortress2Parser{}.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Display.WindowMode != WindowBorderless {
		t.Errorf("Display.WindowMode = %q, want %q", got.Display.WindowMode, WindowBorderless)
	}
	if got.Detail.AntiAliasing != "Off" {
		t.Errorf("Detail.AntiAliasing = %q, want %q", got.Detail.AntiAliasing, "Off")
	}
}

func TestTeamFortress2ParserParseInvalid(t *testing.T) {
	if _, err := (teamFortress2Parser{}).Parse([]byte("not vdf")); err == nil {
		t.Error("Parse() error = nil, want error for malformed video.txt")
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

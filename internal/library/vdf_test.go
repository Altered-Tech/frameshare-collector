package library

import (
	"reflect"
	"testing"
)

func TestParseVDF(t *testing.T) {
	input := `"AppState"
{
	"appid"		"228980"
	"name"		"Steamworks Common Redistributables"
	"nested"
	{
		"a"	"1"
	}
}
`
	got, err := parseVDF([]byte(input))
	if err != nil {
		t.Fatalf("parseVDF() error = %v", err)
	}
	want := map[string]any{
		"AppState": map[string]any{
			"appid": "228980",
			"name":  "Steamworks Common Redistributables",
			"nested": map[string]any{
				"a": "1",
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseVDF() = %#v, want %#v", got, want)
	}
}

func TestParseVDFEscapes(t *testing.T) {
	input := `"path"		"C:\\Program Files (x86)\\Steam"` + "\n"
	got, err := parseVDF([]byte(input))
	if err != nil {
		t.Fatalf("parseVDF() error = %v", err)
	}
	want := map[string]any{"path": `C:\Program Files (x86)\Steam`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseVDF() = %#v, want %#v", got, want)
	}
}

func TestParseVDFErrors(t *testing.T) {
	cases := map[string]string{
		"unterminated string": `"key"		"value`,
		"unclosed object":     `"key" { "a" "b"`,
		"stray close brace":   `"key" "value" }`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseVDF([]byte(input)); err == nil {
				t.Errorf("parseVDF(%q) error = nil, want error", input)
			}
		})
	}
}

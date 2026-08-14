package gamesettings

import (
	"reflect"
	"testing"
)

func TestParseINI(t *testing.T) {
	input := `; leading comment
GlobalKey=GlobalValue

[ScalabilityGroups]
sg.EffectsQuality=1
sg.TextureQuality=1
# another comment style
sg.ResolutionQuality=100.000000

[/Script/Hk_project.HKGameUserSettings]
ResolutionSizeX=1920
ResolutionSizeY=1080
`
	got := parseINI([]byte(input))
	want := map[string]map[string]string{
		"": {"GlobalKey": "GlobalValue"},
		"ScalabilityGroups": {
			"sg.EffectsQuality":    "1",
			"sg.TextureQuality":    "1",
			"sg.ResolutionQuality": "100.000000",
		},
		"/Script/Hk_project.HKGameUserSettings": {
			"ResolutionSizeX": "1920",
			"ResolutionSizeY": "1080",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseINI() = %#v, want %#v", got, want)
	}
}

func TestParseINIDuplicateKeyKeepsLast(t *testing.T) {
	got := parseINI([]byte("[S]\nk=first\nk=second\n"))
	if got["S"]["k"] != "second" {
		t.Errorf(`parseINI() section "S" key "k" = %q, want "second"`, got["S"]["k"])
	}
}

func TestIniSectionWithSuffix(t *testing.T) {
	sections := map[string]map[string]string{
		"ScalabilityGroups":                     {"a": "1"},
		"/Script/Hk_project.HKGameUserSettings": {"b": "2"},
	}

	got, ok := iniSectionWithSuffix(sections, "GameUserSettings")
	if !ok || got["b"] != "2" {
		t.Errorf("iniSectionWithSuffix() = %#v, %v, want the HKGameUserSettings section", got, ok)
	}

	if _, ok := iniSectionWithSuffix(sections, "NoSuchSuffix"); ok {
		t.Errorf("iniSectionWithSuffix() ok = true for a suffix with no match")
	}
}

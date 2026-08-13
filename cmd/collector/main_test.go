package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

func testLibraries() []library.Library {
	return []library.Library{
		{
			Source: library.SourceSteam,
			Path:   "/games/SteamLibrary",
			Games: []library.Game{
				{AppID: "1", Name: "Subnautica", InstallPath: "/games/SteamLibrary/steamapps/common/Subnautica"},
				{AppID: "2", Name: "Hades", InstallPath: "/games/SteamLibrary/steamapps/common/Hades"},
			},
		},
	}
}

func TestNumberedGamePicker(t *testing.T) {
	var out bytes.Buffer
	entries := gameEntries(testLibraries())

	idx, err := numberedGamePicker(entries, strings.NewReader("2\n"), &out)
	if err != nil {
		t.Fatalf("numberedGamePicker() error = %v", err)
	}
	if entries[idx].game.Name != "Hades" {
		t.Errorf("numberedGamePicker() game = %q, want Hades", entries[idx].game.Name)
	}
	if entries[idx].source != library.SourceSteam {
		t.Errorf("numberedGamePicker() source = %q, want %q", entries[idx].source, library.SourceSteam)
	}
	if !strings.Contains(out.String(), "1) Subnautica") || !strings.Contains(out.String(), "2) Hades") {
		t.Errorf("numberedGamePicker() output = %q, want both games listed", out.String())
	}
}

func TestNumberedGamePickerInvalidSelection(t *testing.T) {
	entries := gameEntries(testLibraries())
	cases := []string{"0\n", "3\n", "abc\n", "\n"}
	for _, input := range cases {
		var out bytes.Buffer
		if _, err := numberedGamePicker(entries, strings.NewReader(input), &out); err == nil {
			t.Errorf("numberedGamePicker(%q) error = nil, want error", input)
		}
	}
}

func TestGameEntriesEmpty(t *testing.T) {
	if got := gameEntries(nil); got != nil {
		t.Errorf("gameEntries(nil) = %v, want nil", got)
	}
}

func TestPickGameNoGames(t *testing.T) {
	if _, _, err := pickGame(nil, nil, nil); err == nil {
		t.Error("pickGame(nil libs) error = nil, want error for empty library list")
	}
}

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

func TestChooseGame(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("2\n")

	game, source, err := chooseGame(testLibraries(), in, &out)
	if err != nil {
		t.Fatalf("chooseGame() error = %v", err)
	}
	if game.Name != "Hades" {
		t.Errorf("chooseGame() game = %q, want Hades", game.Name)
	}
	if source != library.SourceSteam {
		t.Errorf("chooseGame() source = %q, want %q", source, library.SourceSteam)
	}
	if !strings.Contains(out.String(), "1) Subnautica") || !strings.Contains(out.String(), "2) Hades") {
		t.Errorf("chooseGame() output = %q, want both games listed", out.String())
	}
}

func TestChooseGameNoGames(t *testing.T) {
	var out bytes.Buffer
	if _, _, err := chooseGame(nil, strings.NewReader("1\n"), &out); err == nil {
		t.Error("chooseGame() error = nil, want error for empty library list")
	}
}

func TestChooseGameInvalidSelection(t *testing.T) {
	cases := []string{"0\n", "3\n", "abc\n", "\n"}
	for _, input := range cases {
		var out bytes.Buffer
		if _, _, err := chooseGame(testLibraries(), strings.NewReader(input), &out); err == nil {
			t.Errorf("chooseGame(%q) error = nil, want error", input)
		}
	}
}

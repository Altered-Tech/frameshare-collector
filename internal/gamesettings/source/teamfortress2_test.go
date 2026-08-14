package source

import (
	"path/filepath"
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/library"
)

func TestTeamFortress2Matches(t *testing.T) {
	p := teamFortress2Parser{}
	if !p.Matches(library.Game{AppID: tf2AppID}, library.SourceSteam) {
		t.Error("Matches() = false, want true for Team Fortress 2's app id")
	}
	if p.Matches(library.Game{AppID: "0"}, library.SourceSteam) {
		t.Error("Matches() = true, want false for a different app id")
	}
}

func TestTeamFortress2ConfigPath(t *testing.T) {
	installPath := filepath.Join(t.TempDir(), "steamapps", "common", "Team Fortress 2")
	got, err := (teamFortress2Parser{}).ConfigPath(library.Game{InstallPath: installPath})
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}
	want := filepath.Join(installPath, "tf", "cfg", "video.txt")
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

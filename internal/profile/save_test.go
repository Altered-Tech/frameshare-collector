package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/hardware"
)

func TestSave_RoundTrips(t *testing.T) {
	dir := t.TempDir()

	when := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	gs := gamesettings.GameProfile{
		Name:     "Team Fortress 2",
		Settings: gamesettings.TitleSettings{Display: gamesettings.DisplaySettings{VSync: true}},
	}
	p := Merge(hardware.Snapshot{CollectedAt: when, Device: hardware.DeviceInfo{Vendor: "Valve"}}, &gs)
	if err := p.SetValue("hardware.Device.Vendor", "Corrected Vendor"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}

	path, err := Save(p, filepath.Join(dir, "profiles"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}

	var loaded Profile
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal saved file: %v", err)
	}

	if loaded.Hardware.Device.Vendor != "Corrected Vendor" {
		t.Errorf("loaded Device.Vendor = %q, want %q", loaded.Hardware.Device.Vendor, "Corrected Vendor")
	}
	if !loaded.Overrides["hardware.Device.Vendor"] {
		t.Errorf("loaded profile lost the override marker for hardware.Device.Vendor")
	}
	if loaded.GameSettings == nil || !loaded.GameSettings.Settings.Display.VSync {
		t.Errorf("loaded GameSettings.Settings.Display.VSync = %+v, want true", loaded.GameSettings)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does", "not", "exist", "yet")
	p := Merge(hardware.Snapshot{CollectedAt: time.Now()}, nil)

	path, err := Save(p, dir)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected saved file to exist at %s: %v", path, err)
	}
}

func TestSave_SameCollectedAtOverwrites(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

	p1 := Merge(hardware.Snapshot{CollectedAt: when, Device: hardware.DeviceInfo{Vendor: "First"}}, nil)
	path1, err := Save(p1, dir)
	if err != nil {
		t.Fatalf("Save (first): %v", err)
	}

	p2 := Merge(hardware.Snapshot{CollectedAt: when, Device: hardware.DeviceInfo{Vendor: "Second"}}, nil)
	path2, err := Save(p2, dir)
	if err != nil {
		t.Fatalf("Save (second): %v", err)
	}

	if path1 != path2 {
		t.Fatalf("expected re-saving the same CollectedAt to overwrite the same file, got %q then %q", path1, path2)
	}

	data, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	var loaded Profile
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded.Hardware.Device.Vendor != "Second" {
		t.Errorf("expected the second save to overwrite the first; got Device.Vendor = %q", loaded.Hardware.Device.Vendor)
	}
}

func TestDefaultDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory available in this environment: %v", err)
	}

	dir, err := DefaultDir()
	if err != nil {
		t.Fatalf("DefaultDir: %v", err)
	}
	want := filepath.Join(home, "FrameShare", "profiles")
	if dir != want {
		t.Errorf("DefaultDir() = %q, want %q", dir, want)
	}
}

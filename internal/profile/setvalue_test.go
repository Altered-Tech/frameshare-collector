package profile

import (
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/hardware"
)

func TestSetValue_String(t *testing.T) {
	gs := gamesettings.GameProfile{Settings: gamesettings.TitleSettings{GraphicsPreset: "Low"}}
	p := Merge(hardware.Snapshot{}, &gs)
	if err := p.SetValue("game_settings.Settings.GraphicsPreset", "Ultra"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if p.GameSettings.Settings.GraphicsPreset != "Ultra" {
		t.Errorf("GraphicsPreset = %q, want %q", p.GameSettings.Settings.GraphicsPreset, "Ultra")
	}
	if !p.Overrides["game_settings.Settings.GraphicsPreset"] {
		t.Errorf("expected game_settings.Settings.GraphicsPreset to be marked overridden")
	}
}

// TestSetValue_RejectsHardwareFields is the core assertion behind
// "lock down hardware fields": hardware detection is reliable enough,
// and a shared profile database trusting enough, that letting a user
// freely retype it would let false information (e.g. claiming a
// different GPU than the machine actually has) look identical to a
// genuine correction. Game settings stay editable since those come from
// a config file a parser may only partially understand (see
// internal/gamesettings), not a spec someone could misrepresent as
// easily. Covers a plain, a nested, and an indexed-slice path shape, so
// the lock can't be bypassed via a path resolvePath handles differently
// internally.
func TestSetValue_RejectsHardwareFields(t *testing.T) {
	p := Merge(hardware.Snapshot{
		Device: hardware.DeviceInfo{Vendor: "Valve"},
		CPU:    hardware.CPUInfo{PhysicalCores: 4},
		GPUs:   []hardware.GPUInfo{{Name: "GPU A"}},
	}, nil)

	paths := []string{
		"hardware.Device.Vendor",
		"hardware.CPU.PhysicalCores",
		"hardware.GPUs[1].Name",
	}
	for _, path := range paths {
		if p.Editable(path) {
			t.Errorf("Editable(%q) = true, want false (hardware fields are locked)", path)
		}
		if err := p.SetValue(path, "tampered"); err == nil {
			t.Errorf("SetValue(%q, ...) succeeded, want an error (hardware fields are locked)", path)
		}
	}

	if p.Hardware.Device.Vendor != "Valve" {
		t.Errorf("Device.Vendor changed despite being locked: %q", p.Hardware.Device.Vendor)
	}
	if p.Hardware.CPU.PhysicalCores != 4 {
		t.Errorf("CPU.PhysicalCores changed despite being locked: %d", p.Hardware.CPU.PhysicalCores)
	}
	if p.Hardware.GPUs[0].Name != "GPU A" {
		t.Errorf("GPUs[0].Name changed despite being locked: %q", p.Hardware.GPUs[0].Name)
	}
	if len(p.Overrides) != 0 {
		t.Errorf("expected no overrides to be recorded, got %v", p.Overrides)
	}
}

func TestSetValue_UnchangedResubmitDoesNotMarkOverridden(t *testing.T) {
	gs := gamesettings.GameProfile{Settings: gamesettings.TitleSettings{GraphicsPreset: "Low"}}
	p := Merge(hardware.Snapshot{}, &gs)

	if err := p.SetValue("game_settings.Settings.GraphicsPreset", "Low"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if p.Overrides["game_settings.Settings.GraphicsPreset"] {
		t.Errorf("re-submitting the same value marked the field overridden")
	}

	if err := p.SetValue("game_settings.Settings.GraphicsPreset", "Ultra"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if !p.Overrides["game_settings.Settings.GraphicsPreset"] {
		t.Errorf("expected a genuine change to mark the field overridden")
	}
}

func TestSetValue_RoundTripsDisplayedBool(t *testing.T) {
	gs := gamesettings.GameProfile{Settings: gamesettings.TitleSettings{Display: gamesettings.DisplaySettings{VSync: true}}}
	gp := Merge(hardware.Snapshot{}, &gs)

	displayed, ok := fieldByPath(gp.Fields(), "game_settings.Settings.Display.VSync")
	if !ok {
		t.Fatalf("expected game_settings.Settings.Display.VSync field")
	}
	if displayed.Value != "Yes" {
		t.Fatalf("precondition: displayed VSync = %q, want %q", displayed.Value, "Yes")
	}

	// Re-submitting the pre-filled display text unchanged must round-trip,
	// not fail to parse -- this is exactly what happens if a user opens
	// the edit widget and hits Save without changing anything.
	if err := gp.SetValue("game_settings.Settings.Display.VSync", displayed.Value); err != nil {
		t.Fatalf("SetValue with unchanged displayed value: %v", err)
	}
	if !gp.GameSettings.Settings.Display.VSync {
		t.Errorf("VSync = false after round-tripping %q", displayed.Value)
	}
}

func TestSetValue_IntAndFloat(t *testing.T) {
	gs := gamesettings.GameProfile{Settings: gamesettings.TitleSettings{
		Detail:  gamesettings.DetailSettings{AnisotropicFiltering: 4},
		Display: gamesettings.DisplaySettings{RefreshRateHz: 60},
	}}
	p := Merge(hardware.Snapshot{}, &gs)

	if err := p.SetValue("game_settings.Settings.Detail.AnisotropicFiltering", "16"); err != nil {
		t.Fatalf("SetValue int: %v", err)
	}
	if p.GameSettings.Settings.Detail.AnisotropicFiltering != 16 {
		t.Errorf("AnisotropicFiltering = %d, want 16", p.GameSettings.Settings.Detail.AnisotropicFiltering)
	}

	if err := p.SetValue("game_settings.Settings.Display.RefreshRateHz", "120.5"); err != nil {
		t.Fatalf("SetValue float: %v", err)
	}
	if p.GameSettings.Settings.Display.RefreshRateHz != 120.5 {
		t.Errorf("RefreshRateHz = %v, want 120.5", p.GameSettings.Settings.Display.RefreshRateHz)
	}
}

func TestSetValue_RejectsBadInput(t *testing.T) {
	gs := gamesettings.GameProfile{Settings: gamesettings.TitleSettings{Detail: gamesettings.DetailSettings{AnisotropicFiltering: 4}}}
	p := Merge(hardware.Snapshot{}, &gs)

	if err := p.SetValue("game_settings.Settings.Detail.AnisotropicFiltering", "not a number"); err == nil {
		t.Fatal("expected an error for non-numeric input to an int field")
	}
	if p.GameSettings.Settings.Detail.AnisotropicFiltering != 4 {
		t.Errorf("AnisotropicFiltering changed despite rejected input: %d", p.GameSettings.Settings.Detail.AnisotropicFiltering)
	}
	if p.Overrides["game_settings.Settings.Detail.AnisotropicFiltering"] {
		t.Errorf("field marked overridden despite rejected input")
	}
}

func TestEditable(t *testing.T) {
	gs := gamesettings.GameProfile{Settings: gamesettings.TitleSettings{GraphicsPreset: "Low"}}
	p := Merge(hardware.Snapshot{Device: hardware.DeviceInfo{Vendor: "Valve"}}, &gs)

	if !p.Editable("game_settings.Settings.GraphicsPreset") {
		t.Error("expected game_settings.Settings.GraphicsPreset to be editable")
	}
	if p.Editable("hardware.Device.Vendor") {
		t.Error("expected hardware.Device.Vendor to be locked, not editable")
	}
	if p.Editable("game_settings.ParsedAt") {
		t.Error("expected game_settings.ParsedAt (a timestamp) to be not editable")
	}
	if p.Editable("hardware.NotAField") {
		t.Error("expected an unknown path to be not editable")
	}
}

func TestSetValue_RejectsUnknownAndTimestampPaths(t *testing.T) {
	gs := gamesettings.GameProfile{}
	p := Merge(hardware.Snapshot{}, &gs)

	if err := p.SetValue("game_settings.NotAField", "x"); err == nil {
		t.Error("expected an error for an unknown field path")
	}
	if err := p.SetValue("game_settings.ParsedAt", "2026-01-01"); err == nil {
		t.Error("expected an error editing a timestamp field")
	}

	noGameSettings := Merge(hardware.Snapshot{}, nil)
	if err := noGameSettings.SetValue("game_settings.Settings.Display.VSync", "yes"); err == nil {
		t.Error("expected an error setting a game_settings field with no GameSettings on the profile")
	}
}

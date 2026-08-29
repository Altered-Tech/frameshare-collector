package profile

import (
	"testing"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/hardware"
)

func TestSetValue_String(t *testing.T) {
	p := Merge(hardware.Snapshot{Device: hardware.DeviceInfo{Vendor: "Valve"}}, nil)
	if err := p.SetValue("hardware.Device.Vendor", "Not Valve"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if p.Hardware.Device.Vendor != "Not Valve" {
		t.Errorf("Device.Vendor = %q, want %q", p.Hardware.Device.Vendor, "Not Valve")
	}
	if !p.Overrides["hardware.Device.Vendor"] {
		t.Errorf("expected hardware.Device.Vendor to be marked overridden")
	}
}

func TestSetValue_UnchangedResubmitDoesNotMarkOverridden(t *testing.T) {
	p := Merge(hardware.Snapshot{Device: hardware.DeviceInfo{Vendor: "Valve"}}, nil)
	if err := p.SetValue("hardware.Device.Vendor", "Valve"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if p.Overrides["hardware.Device.Vendor"] {
		t.Errorf("re-submitting the same value marked the field overridden")
	}

	if err := p.SetValue("hardware.Device.Vendor", "Steam Machine"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if !p.Overrides["hardware.Device.Vendor"] {
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
	p := Merge(hardware.Snapshot{CPU: hardware.CPUInfo{PhysicalCores: 4}}, nil)
	if err := p.SetValue("hardware.CPU.PhysicalCores", "8"); err != nil {
		t.Fatalf("SetValue int: %v", err)
	}
	if p.Hardware.CPU.PhysicalCores != 8 {
		t.Errorf("PhysicalCores = %d, want 8", p.Hardware.CPU.PhysicalCores)
	}

	if err := p.SetValue("hardware.CPU.MaxMHz", "4200.5"); err != nil {
		t.Fatalf("SetValue float: %v", err)
	}
	if p.Hardware.CPU.MaxMHz != 4200.5 {
		t.Errorf("MaxMHz = %v, want 4200.5", p.Hardware.CPU.MaxMHz)
	}
}

func TestSetValue_SliceIndex(t *testing.T) {
	p := Merge(hardware.Snapshot{GPUs: []hardware.GPUInfo{{Name: "GPU A"}, {Name: "GPU B"}}}, nil)
	if err := p.SetValue("hardware.GPUs[2].Name", "Renamed GPU"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if p.Hardware.GPUs[1].Name != "Renamed GPU" {
		t.Errorf("GPUs[1].Name = %q, want %q", p.Hardware.GPUs[1].Name, "Renamed GPU")
	}
	if p.Hardware.GPUs[0].Name != "GPU A" {
		t.Errorf("GPUs[0].Name changed unexpectedly: %q", p.Hardware.GPUs[0].Name)
	}
}

func TestSetValue_RejectsBadInput(t *testing.T) {
	p := Merge(hardware.Snapshot{CPU: hardware.CPUInfo{PhysicalCores: 4}}, nil)
	if err := p.SetValue("hardware.CPU.PhysicalCores", "not a number"); err == nil {
		t.Fatal("expected an error for non-numeric input to an int field")
	}
	if p.Hardware.CPU.PhysicalCores != 4 {
		t.Errorf("PhysicalCores changed despite rejected input: %d", p.Hardware.CPU.PhysicalCores)
	}
	if p.Overrides["hardware.CPU.PhysicalCores"] {
		t.Errorf("field marked overridden despite rejected input")
	}
}

func TestEditable(t *testing.T) {
	p := Merge(hardware.Snapshot{Device: hardware.DeviceInfo{Vendor: "Valve"}}, nil)
	if !p.Editable("hardware.Device.Vendor") {
		t.Error("expected hardware.Device.Vendor to be editable")
	}
	if p.Editable("hardware.CollectedAt") {
		t.Error("expected hardware.CollectedAt (a timestamp) to be not editable")
	}
	if p.Editable("hardware.NotAField") {
		t.Error("expected an unknown path to be not editable")
	}
}

func TestSetValue_RejectsUnknownAndTimestampPaths(t *testing.T) {
	p := Merge(hardware.Snapshot{}, nil)
	if err := p.SetValue("hardware.NotAField", "x"); err == nil {
		t.Error("expected an error for an unknown field path")
	}
	if err := p.SetValue("hardware.CollectedAt", "2026-01-01"); err == nil {
		t.Error("expected an error editing a timestamp field")
	}
	if err := p.SetValue("game_settings.Settings.Display.VSync", "yes"); err == nil {
		t.Error("expected an error setting a game_settings field with no GameSettings on the profile")
	}
}

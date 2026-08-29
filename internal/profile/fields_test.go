package profile

import (
	"strings"
	"testing"
	"time"

	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	"github.com/alteredtech/frameshare-collector/internal/hardware"
)

func fieldByPath(fields []Field, path string) (Field, bool) {
	for _, f := range fields {
		if f.Path == path {
			return f, true
		}
	}
	return Field{}, false
}

func TestFields_HardwareOnly(t *testing.T) {
	hw := hardware.Snapshot{
		CollectorVersion: "v1.2.3",
		Device:           hardware.DeviceInfo{Vendor: "Valve", Model: "Jupiter"},
		CPU:              hardware.CPUInfo{Model: "AMD Custom APU 0405", PhysicalCores: 4},
		GPUs:             []hardware.GPUInfo{{Name: "AMD Radeon Graphics"}},
	}
	p := Merge(hw, nil)
	fields := p.Fields()

	f, ok := fieldByPath(fields, "hardware.Device.Vendor")
	if !ok {
		t.Fatalf("expected hardware.Device.Vendor field, got %+v", fields)
	}
	if f.Value != "Valve" {
		t.Errorf("Device.Vendor = %q, want %q", f.Value, "Valve")
	}
	if f.Label != "Device Vendor" {
		t.Errorf("Device.Vendor label = %q, want %q", f.Label, "Device Vendor")
	}

	f, ok = fieldByPath(fields, "hardware.CPU.PhysicalCores")
	if !ok || f.Value != "4" {
		t.Errorf("CPU.PhysicalCores = %+v, ok=%v, want value 4", f, ok)
	}
	if f.Label != "CPU Physical Cores" {
		t.Errorf("CPU.PhysicalCores label = %q, want %q", f.Label, "CPU Physical Cores")
	}

	if _, ok := fieldByPath(fields, "hardware.GPUs[1].Name"); !ok {
		t.Errorf("expected an indexed field for GPUs[1].Name, got %+v", fields)
	}

	for _, f := range fields {
		if strings.HasPrefix(f.Path, "game_settings") {
			t.Errorf("did not expect a game_settings field with nil GameSettings: %+v", f)
		}
	}
}

func TestFields_GameSettingsExcludesConfigPath(t *testing.T) {
	gs := gamesettings.GameProfile{
		Name:       "Team Fortress 2",
		ConfigPath: "/home/deck/.steam/steam/userdata/12345/config.cfg",
		Settings: gamesettings.TitleSettings{
			Display: gamesettings.DisplaySettings{VSync: true},
		},
	}
	p := Merge(hardware.Snapshot{}, &gs)
	fields := p.Fields()

	for _, f := range fields {
		if f.Value == gs.ConfigPath {
			t.Errorf("ConfigPath leaked into review fields: %+v", f)
		}
	}

	f, ok := fieldByPath(fields, "game_settings.Settings.Display.VSync")
	if !ok || f.Value != "Yes" {
		t.Errorf("Settings.Display.VSync = %+v, ok=%v, want value Yes", f, ok)
	}
}

func TestFields_Overrides(t *testing.T) {
	p := Merge(hardware.Snapshot{Device: hardware.DeviceInfo{Vendor: "Valve"}}, nil)
	p.Overrides["hardware.Device.Vendor"] = true

	f, ok := fieldByPath(p.Fields(), "hardware.Device.Vendor")
	if !ok || !f.Overridden {
		t.Errorf("Device.Vendor = %+v, ok=%v, want Overridden=true", f, ok)
	}
}

func TestFields_TimeFormatting(t *testing.T) {
	when := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	p := Merge(hardware.Snapshot{CollectedAt: when}, nil)

	f, ok := fieldByPath(p.Fields(), "hardware.CollectedAt")
	if !ok {
		t.Fatalf("expected hardware.CollectedAt field, got %+v", p.Fields())
	}
	if f.Value == "" {
		t.Errorf("expected non-empty formatted time, got empty string")
	}

	zero := Merge(hardware.Snapshot{}, nil)
	f, ok = fieldByPath(zero.Fields(), "hardware.CollectedAt")
	if !ok || f.Value != "" {
		t.Errorf("zero time: CollectedAt = %+v, ok=%v, want empty value", f, ok)
	}
}

func TestHumanize(t *testing.T) {
	cases := map[string]string{
		"Vendor":               "Vendor",
		"PhysicalCores":        "Physical Cores",
		"MaxMHz":               "Max MHz",
		"VRAMBytes":            "VRAM Bytes",
		"CPU":                  "CPU",
		"AnisotropicFiltering": "Anisotropic Filtering",
	}
	for in, want := range cases {
		if got := humanize(in); got != want {
			t.Errorf("humanize(%q) = %q, want %q", in, got, want)
		}
	}
}

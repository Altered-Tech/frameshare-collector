// Package hardware detects local hardware and OS information for a Phase 1
// FrameShare snapshot: no network calls, no UI.
package hardware

import "time"

// Snapshot is the full local hardware report written to disk.
type Snapshot struct {
	CollectorVersion string     `json:"collector_version"`
	CollectedAt      time.Time  `json:"collected_at"`
	Device           DeviceInfo `json:"device"`
	OS               OSInfo     `json:"os"`
	CPU              CPUInfo    `json:"cpu"`
	Memory           MemInfo    `json:"memory"`
	GPUs             []GPUInfo  `json:"gpus"`
	Displays         []Display  `json:"displays"`
	Storage          []Disk     `json:"storage"`
}

// DeviceInfo identifies the physical machine model via DMI/SMBIOS strings.
// KnownHandheld is set when Vendor/Model matches a recognized gaming
// handheld (Steam Deck, ROG Ally, etc); it's empty for unrecognized or
// non-handheld machines.
type DeviceInfo struct {
	Vendor        string `json:"vendor,omitempty"`
	Model         string `json:"model,omitempty"`
	KnownHandheld string `json:"known_handheld,omitempty"`
}

type OSInfo struct {
	Platform      string `json:"platform"` // e.g. "darwin", "windows", "linux"
	Family        string `json:"family"`   // e.g. "Standalone Workstation", "Windows", "debian"
	Name          string `json:"name"`     // human-readable name, e.g. "macOS", "Windows 11"
	Version       string `json:"version"`
	KernelVersion string `json:"kernel_version"`
	Arch          string `json:"arch"` // e.g. "arm64", "amd64"
}

type CPUInfo struct {
	Model         string  `json:"model"`
	Vendor        string  `json:"vendor"`
	PhysicalCores int     `json:"physical_cores"`
	LogicalCores  int     `json:"logical_cores"`
	MaxMHz        float64 `json:"max_mhz,omitempty"`
}

type MemInfo struct {
	TotalBytes uint64  `json:"total_bytes"`
	TotalGB    float64 `json:"total_gb"`
}

type GPUInfo struct {
	Name      string  `json:"name"`
	Vendor    string  `json:"vendor,omitempty"`
	VRAMBytes uint64  `json:"vram_bytes,omitempty"`
	VRAMGB    float64 `json:"vram_gb,omitempty"`
}

type Display struct {
	// Name is the connector/adapter identifier (e.g. "DP-1" on Linux, the
	// GPU adapter name on Windows) — not a reliable monitor identity. Kept
	// for backward compatibility and as a fallback when EDID is unreadable.
	Name          string  `json:"name,omitempty"`
	WidthPx       int     `json:"width_px"`
	HeightPx      int     `json:"height_px"`
	RefreshHz     float64 `json:"refresh_hz,omitempty"`
	IsPrimary     bool    `json:"is_primary,omitempty"`
	MonitorVendor string  `json:"monitor_vendor,omitempty"` // decoded PNP ID -> name, or raw PNP ID if unmapped
	MonitorModel  string  `json:"monitor_model,omitempty"`  // EDID product name descriptor, e.g. "AW3225QF"
}

type Disk struct {
	Device     string  `json:"device"`
	Mountpoint string  `json:"mountpoint"`
	Fstype     string  `json:"fstype,omitempty"`
	TotalBytes uint64  `json:"total_bytes"`
	TotalGB    float64 `json:"total_gb"`
}

// Package hardware detects local hardware and OS information for a Phase 1
// FrameShare snapshot: no network calls, no UI.
package hardware

import "time"

// Snapshot is the full local hardware report written to disk.
type Snapshot struct {
	CollectedAt time.Time `json:"collected_at"`
	OS          OSInfo    `json:"os"`
	CPU         CPUInfo   `json:"cpu"`
	Memory      MemInfo   `json:"memory"`
	GPUs        []GPUInfo `json:"gpus"`
	Displays    []Display `json:"displays"`
	Storage     []Disk    `json:"storage"`
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
	Name      string  `json:"name,omitempty"`
	WidthPx   int     `json:"width_px"`
	HeightPx  int     `json:"height_px"`
	RefreshHz float64 `json:"refresh_hz,omitempty"`
	IsPrimary bool    `json:"is_primary,omitempty"`
}

type Disk struct {
	Device     string  `json:"device"`
	Mountpoint string  `json:"mountpoint"`
	Fstype     string  `json:"fstype,omitempty"`
	TotalBytes uint64  `json:"total_bytes"`
	TotalGB    float64 `json:"total_gb"`
}

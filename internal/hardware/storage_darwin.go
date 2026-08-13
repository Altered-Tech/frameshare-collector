package hardware

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
)

// macInternalMountPrefixes are mountpoints for macOS-internal volumes that
// share a physical disk with "/" (Data, Preboot, VM, Update, Recovery,
// etc) or are ephemeral disk images (simulator/cryptex mounts under
// /private/var/run), not separate physical drives.
var macInternalMountPrefixes = []string{"/System/Volumes/", "/private/var/run/"}

// macInternalMounts are exact-match mountpoints for other non-physical
// macOS volumes that live outside the prefixes above.
var macInternalMounts = map[string]bool{"/Volumes/Recovery": true}

// macPhysicalDiskPattern extracts the top-level disk identifier (e.g.
// "disk3") from a BSD device name like "disk3s1s1", so multiple APFS
// volumes/containers on the same physical disk can be deduplicated.
var macPhysicalDiskPattern = regexp.MustCompile(`^disk\d+`)

// spStorageOutput mirrors the JSON shape of
// `system_profiler SPStorageDataType -json`.
type spStorageOutput struct {
	SPStorageDataType []spVolume `json:"SPStorageDataType"`
}

type spVolume struct {
	BSDName    string          `json:"bsd_name"`
	MountPoint string          `json:"mount_point"`
	SizeBytes  uint64          `json:"size_in_bytes"`
	Physical   spPhysicalDrive `json:"physical_drive"`
}

type spPhysicalDrive struct {
	DeviceName string `json:"device_name"`
	MediumType string `json:"medium_type"` // "ssd" or "rotational"
}

func collectStorage(ctx context.Context, installPath string) ([]Disk, error) {
	cmd := exec.CommandContext(ctx, "system_profiler", "SPStorageDataType", "-json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var parsed spStorageOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, err
	}

	installDiskID := ""
	if installPath != "" {
		installDiskID = macDiskIDForPath(ctx, installPath)
	}

	seen := map[string]bool{}
	var disks []Disk
	for _, v := range parsed.SPStorageDataType {
		if isMacInternalMount(v.MountPoint) {
			continue
		}
		id := macPhysicalDiskPattern.FindString(v.BSDName)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		isOSDrive := v.MountPoint == "/"
		isInstallDrive := installDiskID != "" && id == installDiskID
		disks = append(disks, Disk{
			Model:          v.Physical.DeviceName,
			Type:           macDiskType(v.Physical.MediumType),
			TotalBytes:     v.SizeBytes,
			TotalGB:        round2(float64(v.SizeBytes) / bytesPerGB),
			IsOSDrive:      isOSDrive,
			IsInstallDrive: isInstallDrive,
			Role:           diskRole(isOSDrive, isInstallDrive),
		})
	}
	return disks, nil
}

// macDiskIDForPath resolves an arbitrary filesystem path to the top-level
// physical disk identifier (e.g. "disk3") backing it, via `df`. Returns ""
// if the path doesn't exist or df fails.
func macDiskIDForPath(ctx context.Context, path string) string {
	cmd := exec.CommandContext(ctx, "df", "-P", path)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return ""
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) == 0 {
		return ""
	}
	device := strings.TrimPrefix(fields[0], "/dev/")
	return macPhysicalDiskPattern.FindString(device)
}

func macDiskType(medium string) string {
	switch strings.ToLower(medium) {
	case "ssd":
		return "SSD"
	case "rotational":
		return "HDD"
	default:
		return ""
	}
}

func isMacInternalMount(mountpoint string) bool {
	if macInternalMounts[mountpoint] {
		return true
	}
	for _, prefix := range macInternalMountPrefixes {
		if strings.HasPrefix(mountpoint, prefix) {
			return true
		}
	}
	return false
}

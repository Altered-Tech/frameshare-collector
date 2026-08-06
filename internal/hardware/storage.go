package hardware

import (
	"context"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

// pseudoFstypes are filesystem types that don't represent real local
// physical storage (virtual, network, or OS-internal mounts) and are
// excluded from the report.
var pseudoFstypes = map[string]bool{
	"devfs": true, "tmpfs": true, "proc": true, "sysfs": true,
	"cgroup": true, "cgroup2": true, "overlay": true, "squashfs": true,
	"autofs": true, "debugfs": true, "mqueue": true, "hugetlbfs": true,
	"tracefs": true, "configfs": true, "securityfs": true, "pstore": true,
	"binfmt_misc": true, "efivarfs": true, "rpc_pipefs": true,
	// network filesystems: not local hardware
	"smbfs": true, "nfs": true, "nfs4": true, "cifs": true,
	"afpfs": true, "webdav": true, "ftp": true,
}

// macOS mounts everything under /System/Volumes/* from the same physical
// APFS container as "/" (Data, Preboot, VM, Update, Recovery, etc). They're
// not separate disks, so only "/" is kept to avoid reporting one physical
// drive N times. Simulator/cryptex mounts under /private/var/run are
// likewise not real storage.
var macInternalMountPrefixes = []string{"/System/Volumes/", "/private/var/run/"}

// macInternalMounts are exact-match mountpoints for other non-physical
// macOS volumes that live outside the prefixes above.
var macInternalMounts = map[string]bool{"/Volumes/Recovery": true}

func collectStorage(ctx context.Context) ([]Disk, error) {
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, err
	}

	var disks []Disk
	for _, p := range partitions {
		if pseudoFstypes[strings.ToLower(p.Fstype)] {
			continue
		}
		if isMacInternalMount(p.Mountpoint) {
			continue
		}
		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || usage.Total == 0 {
			continue
		}
		disks = append(disks, Disk{
			Device:     p.Device,
			Mountpoint: p.Mountpoint,
			Fstype:     p.Fstype,
			TotalBytes: usage.Total,
			TotalGB:    round2(float64(usage.Total) / bytesPerGB),
		})
	}
	return disks, nil
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

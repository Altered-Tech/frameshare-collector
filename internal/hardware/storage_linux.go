package hardware

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// linuxVirtualBlockPrefixes are /sys/block entries that don't represent a
// distinct physical disk: loopback devices, RAM disks, optical drives,
// zram, device-mapper layers (LUKS/LVM sit on top of a real disk that's
// already counted), and software RAID arrays (built from disks that are
// already counted).
var linuxVirtualBlockPrefixes = []string{"loop", "ram", "sr", "zram", "dm-", "md", "fd"}

const sectorSize = 512

func collectStorage(ctx context.Context, installPath string) ([]Disk, error) {
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil, err
	}

	osDisks := diskNameSet(resolvePhysicalDisks(deviceForPath("/")))
	var installDisks map[string]bool
	if installPath != "" {
		installDisks = diskNameSet(resolvePhysicalDisks(deviceForPath(installPath)))
	}

	var disks []Disk
	for _, e := range entries {
		name := e.Name()
		if isVirtualBlockDevice(name) {
			continue
		}
		sizeSectors := readSysfsUint("/sys/block/" + name + "/size")
		if sizeSectors == 0 {
			continue
		}
		totalBytes := sizeSectors * sectorSize
		isOSDrive := osDisks[name]
		isInstallDrive := installDisks[name]
		disks = append(disks, Disk{
			Model:          linuxDiskModel(name),
			Type:           linuxDiskType(name),
			TotalBytes:     totalBytes,
			TotalGB:        round2(float64(totalBytes) / bytesPerGB),
			IsOSDrive:      isOSDrive,
			IsInstallDrive: isInstallDrive,
			Role:           diskRole(isOSDrive, isInstallDrive),
		})
	}
	return disks, nil
}

func isVirtualBlockDevice(name string) bool {
	for _, prefix := range linuxVirtualBlockPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// linuxDiskModel reads the device model string. ATA/SCSI/NVMe devices
// expose it as "model"; MMC/SD devices (mmcblk*) expose it as "name".
func linuxDiskModel(name string) string {
	for _, field := range []string{"model", "name"} {
		if v := readSysfsField("/sys/block/" + name + "/device/" + field); v != "" {
			return v
		}
	}
	return ""
}

func linuxDiskType(name string) string {
	if strings.HasPrefix(name, "nvme") {
		return "NVMe SSD"
	}
	switch readSysfsField("/sys/block/" + name + "/queue/rotational") {
	case "1":
		return "HDD"
	case "0":
		return "SSD"
	default:
		return ""
	}
}

func readSysfsField(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readSysfsUint(path string) uint64 {
	v, err := strconv.ParseUint(readSysfsField(path), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func diskNameSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
}

// deviceForPath returns the device backing path (e.g. "/dev/nvme0n1p2" or
// "/dev/mapper/vg-root"), found by matching path's stat device ID against
// the longest-matching mountpoint in /proc/mounts (the standard "find the
// containing mount" technique). Returns "" if path doesn't exist or
// /proc/mounts can't be read.
func deviceForPath(path string) string {
	var pst syscall.Stat_t
	if err := syscall.Stat(path, &pst); err != nil {
		return ""
	}

	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return ""
	}

	bestDevice, bestLen := "", -1
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		device := fields[0]
		mountpoint := strings.ReplaceAll(fields[1], `\040`, " ")

		var mst syscall.Stat_t
		if err := syscall.Stat(mountpoint, &mst); err != nil {
			continue
		}
		if mst.Dev != pst.Dev {
			continue
		}
		if len(mountpoint) > bestLen {
			bestLen = len(mountpoint)
			bestDevice = device
		}
	}
	return bestDevice
}

// resolvePhysicalDisks walks from a device (a partition, an LVM/dm-crypt
// mapper device, or a whole disk) down to the underlying whole disk(s) in
// /sys/block, by following /sys/class/block/*/slaves. Device-mapper and
// software RAID devices can be built from more than one physical disk, so
// this can return multiple names.
func resolvePhysicalDisks(device string) []string {
	if device == "" {
		return nil
	}
	name := canonicalBlockName(device)
	if name == "" {
		return nil
	}
	return resolveDiskNames(name, map[string]bool{})
}

func canonicalBlockName(device string) string {
	real, err := filepath.EvalSymlinks(device)
	if err != nil {
		real = device
	}
	return filepath.Base(real)
}

func resolveDiskNames(name string, visited map[string]bool) []string {
	if visited[name] {
		return nil
	}
	visited[name] = true

	if entries, err := os.ReadDir("/sys/class/block/" + name + "/slaves"); err == nil && len(entries) > 0 {
		var result []string
		for _, e := range entries {
			result = append(result, resolveDiskNames(e.Name(), visited)...)
		}
		return result
	}

	if disk := wholeDiskFor(name); disk != "" {
		return []string{disk}
	}
	return nil
}

// wholeDiskFor returns name itself if it's already a whole disk (present
// directly under /sys/block), or its parent whole disk if name is a
// partition of one.
func wholeDiskFor(name string) string {
	if _, err := os.Stat("/sys/block/" + name); err == nil {
		return name
	}
	real, err := filepath.EvalSymlinks("/sys/class/block/" + name)
	if err != nil {
		return ""
	}
	parent := filepath.Base(filepath.Dir(real))
	if _, err := os.Stat("/sys/block/" + parent); err == nil {
		return parent
	}
	return ""
}

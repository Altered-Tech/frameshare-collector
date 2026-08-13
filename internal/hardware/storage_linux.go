package hardware

import (
	"context"
	"os"
	"strconv"
	"strings"
)

// linuxVirtualBlockPrefixes are /sys/block entries that don't represent a
// distinct physical disk: loopback devices, RAM disks, optical drives,
// zram, device-mapper layers (LUKS/LVM sit on top of a real disk that's
// already counted), and software RAID arrays (built from disks that are
// already counted).
var linuxVirtualBlockPrefixes = []string{"loop", "ram", "sr", "zram", "dm-", "md", "fd"}

const sectorSize = 512

func collectStorage(ctx context.Context) ([]Disk, error) {
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil, err
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
		disks = append(disks, Disk{
			Model:      linuxDiskModel(name),
			Type:       linuxDiskType(name),
			TotalBytes: totalBytes,
			TotalGB:    round2(float64(totalBytes) / bytesPerGB),
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

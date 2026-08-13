package hardware

import (
	"context"
	"encoding/json"
)

// win32PhysicalDisk mirrors the fields of Get-PhysicalDisk, which (unlike
// Win32_DiskDrive) already classifies each disk's MediaType as "SSD",
// "HDD", "SCM", or "Unspecified".
type win32PhysicalDisk struct {
	FriendlyName string `json:"FriendlyName"`
	MediaType    string `json:"MediaType"`
	Size         uint64 `json:"Size"`
}

const physicalDiskScript = `Get-PhysicalDisk | ` +
	`Select-Object FriendlyName,MediaType,Size | ConvertTo-Json`

func collectStorage(ctx context.Context) ([]Disk, error) {
	raw, err := runPowerShellJSON(ctx, physicalDiskScript)
	if err != nil {
		return nil, err
	}

	var physicalDisks []win32PhysicalDisk
	if err := json.Unmarshal(normalizeJSONArray(raw), &physicalDisks); err != nil {
		return nil, err
	}

	var disks []Disk
	for _, p := range physicalDisks {
		disks = append(disks, Disk{
			Model:      p.FriendlyName,
			Type:       p.MediaType,
			TotalBytes: p.Size,
			TotalGB:    round2(float64(p.Size) / bytesPerGB),
		})
	}
	return disks, nil
}

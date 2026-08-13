package hardware

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// win32PhysicalDisk mirrors the joined output of Get-PhysicalDisk (which,
// unlike Win32_DiskDrive, already classifies each disk's MediaType as
// "SSD", "HDD", "SCM", or "Unspecified") and Get-Disk (which knows whether
// a disk is the boot/system disk, and is matched to the install path's
// disk number when one is given).
type win32PhysicalDisk struct {
	FriendlyName string `json:"FriendlyName"`
	MediaType    string `json:"MediaType"`
	Size         uint64 `json:"Size"`
	IsBoot       bool   `json:"IsBoot"`
	IsSystem     bool   `json:"IsSystem"`
	IsInstall    bool   `json:"IsInstall"`
}

const physicalDiskScriptTemplate = `
$installPath = %s
$installDiskNumber = $null
if ($installPath) {
	try {
		$vol = Get-Volume -FilePath $installPath -ErrorAction Stop
		$part = $vol | Get-Partition -ErrorAction Stop | Select-Object -First 1
		if ($part) { $installDiskNumber = $part.DiskNumber }
	} catch {}
}
Get-PhysicalDisk | ForEach-Object {
	$pd = $_
	$d = Get-Disk -Number $pd.DeviceId -ErrorAction SilentlyContinue
	[PSCustomObject]@{
		FriendlyName = $pd.FriendlyName
		MediaType    = $pd.MediaType.ToString()
		Size         = $pd.Size
		IsBoot       = if ($d) { [bool]$d.IsBoot } else { $false }
		IsSystem     = if ($d) { [bool]$d.IsSystem } else { $false }
		IsInstall    = [bool]($d -and $installDiskNumber -ne $null -and $d.Number -eq $installDiskNumber)
	}
} | ConvertTo-Json
`

func collectStorage(ctx context.Context, installPath string) ([]Disk, error) {
	raw, err := runPowerShellJSON(ctx, buildPhysicalDiskScript(installPath))
	if err != nil {
		return nil, err
	}

	var physicalDisks []win32PhysicalDisk
	if err := json.Unmarshal(normalizeJSONArray(raw), &physicalDisks); err != nil {
		return nil, err
	}

	var disks []Disk
	for _, p := range physicalDisks {
		isOSDrive := p.IsBoot || p.IsSystem
		disks = append(disks, Disk{
			Model:          p.FriendlyName,
			Type:           p.MediaType,
			TotalBytes:     p.Size,
			TotalGB:        round2(float64(p.Size) / bytesPerGB),
			IsOSDrive:      isOSDrive,
			IsInstallDrive: p.IsInstall,
			Role:           diskRole(isOSDrive, p.IsInstall),
		})
	}
	return disks, nil
}

func buildPhysicalDiskScript(installPath string) string {
	installExpr := "$null"
	if installPath != "" {
		installExpr = psQuote(installPath)
	}
	return fmt.Sprintf(physicalDiskScriptTemplate, installExpr)
}

// psQuote wraps a value in single quotes for safe interpolation into a
// PowerShell script, escaping any embedded single quotes by doubling them
// (the standard PowerShell single-quoted-string escape).
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

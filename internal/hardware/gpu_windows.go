package hardware

import (
	"context"
	"encoding/json"
)

type win32VideoController struct {
	Name                 string `json:"Name"`
	AdapterCompatibility string `json:"AdapterCompatibility"`
	AdapterRAM           uint64 `json:"AdapterRAM"`
}

const videoControllerScript = `Get-CimInstance Win32_VideoController | ` +
	`Select-Object Name,AdapterCompatibility,AdapterRAM,CurrentHorizontalResolution,CurrentVerticalResolution,CurrentRefreshRate | ` +
	`ConvertTo-Json`

func collectGPUs(ctx context.Context) ([]GPUInfo, error) {
	raw, err := runPowerShellJSON(ctx, videoControllerScript)
	if err != nil {
		return nil, err
	}

	var controllers []win32VideoController
	if err := json.Unmarshal(normalizeJSONArray(raw), &controllers); err != nil {
		return nil, err
	}

	var gpus []GPUInfo
	for _, c := range controllers {
		gpu := GPUInfo{
			Name:   c.Name,
			Vendor: c.AdapterCompatibility,
		}
		if c.AdapterRAM > 0 {
			gpu.VRAMBytes = c.AdapterRAM
			gpu.VRAMGB = round2(float64(c.AdapterRAM) / bytesPerGB)
		}
		gpus = append(gpus, gpu)
	}
	return gpus, nil
}

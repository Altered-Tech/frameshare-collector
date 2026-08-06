package hardware

import (
	"context"
	"encoding/json"
	"os/exec"
)

// spDisplaysOutput mirrors the JSON shape of
// `system_profiler SPDisplaysDataType -json`.
type spDisplaysOutput struct {
	SPDisplaysDataType []spGPU `json:"SPDisplaysDataType"`
}

type spGPU struct {
	Name   string   `json:"_name"`
	Model  string   `json:"sppci_model"`
	VRAM   string   `json:"spdisplays_vram,omitempty"`
	Vendor string   `json:"spdisplays_vendor,omitempty"`
	Ndrvs  []spNdrv `json:"spdisplays_ndrvs,omitempty"`
}

type spNdrv struct {
	Name       string `json:"_name"`
	Resolution string `json:"_spdisplays_resolution,omitempty"`
	// Pixels is the true native panel resolution. On HiDPI/Retina displays,
	// Resolution instead reports the scaled logical (points) size used for
	// UI layout, which is smaller than the physical pixel count.
	Pixels string `json:"_spdisplays_pixels,omitempty"`
	Main   string `json:"spdisplays_main,omitempty"`
}

func fetchDisplaysData(ctx context.Context) (spDisplaysOutput, error) {
	cmd := exec.CommandContext(ctx, "system_profiler", "SPDisplaysDataType", "-json")
	out, err := cmd.Output()
	if err != nil {
		return spDisplaysOutput{}, err
	}
	var parsed spDisplaysOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return spDisplaysOutput{}, err
	}
	return parsed, nil
}

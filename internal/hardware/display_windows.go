package hardware

import (
	"context"
	"encoding/json"
)

type win32DisplayController struct {
	Name                        string  `json:"Name"`
	CurrentHorizontalResolution int     `json:"CurrentHorizontalResolution"`
	CurrentVerticalResolution   int     `json:"CurrentVerticalResolution"`
	CurrentRefreshRate          float64 `json:"CurrentRefreshRate"`
}

func collectDisplays(ctx context.Context) ([]Display, error) {
	raw, err := runPowerShellJSON(ctx, videoControllerScript)
	if err != nil {
		return nil, err
	}

	var controllers []win32DisplayController
	if err := json.Unmarshal(normalizeJSONArray(raw), &controllers); err != nil {
		return nil, err
	}

	var displays []Display
	for _, c := range controllers {
		if c.CurrentHorizontalResolution == 0 || c.CurrentVerticalResolution == 0 {
			continue // adapter has no active display attached
		}
		displays = append(displays, Display{
			Name:      c.Name,
			WidthPx:   c.CurrentHorizontalResolution,
			HeightPx:  c.CurrentVerticalResolution,
			RefreshHz: c.CurrentRefreshRate,
		})
	}
	return displays, nil
}

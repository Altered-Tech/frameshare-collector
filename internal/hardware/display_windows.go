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

	// WmiMonitorID (root\wmi) carries monitor identity but no resolution,
	// and has no key shared with Win32_VideoController above; pair entries
	// positionally when counts match (the common case: one active monitor
	// per adapter row), otherwise leave monitor identity empty rather than
	// risk mislabeling a display.
	monitors := fetchMonitorIDs(ctx)
	if len(monitors) == len(displays) {
		for i, m := range monitors {
			vendor := decodeWMICharCodes(m.ManufacturerName)
			if vendor != "" {
				displays[i].MonitorVendor = resolvePNPVendor(vendor)
			}
			displays[i].MonitorModel = decodeWMICharCodes(m.ProductCodeID)
		}
	}

	return displays, nil
}

const monitorIDScript = `Get-CimInstance -Namespace root\wmi -ClassName WmiMonitorID | ` +
	`Select-Object ManufacturerName,ProductCodeID | ConvertTo-Json`

type win32MonitorID struct {
	ManufacturerName []uint16 `json:"ManufacturerName"`
	ProductCodeID    []uint16 `json:"ProductCodeID"`
}

func fetchMonitorIDs(ctx context.Context) []win32MonitorID {
	raw, err := runPowerShellJSON(ctx, monitorIDScript)
	if err != nil {
		return nil
	}
	var monitors []win32MonitorID
	if err := json.Unmarshal(normalizeJSONArray(raw), &monitors); err != nil {
		return nil
	}
	return monitors
}

// decodeWMICharCodes converts a WMI UInt16 code-point array (as returned
// for WmiMonitorID's ManufacturerName/ProductCodeID/SerialNumberID) into a
// string, stopping at the first null terminator.
func decodeWMICharCodes(codes []uint16) string {
	b := make([]byte, 0, len(codes))
	for _, c := range codes {
		if c == 0 {
			break
		}
		b = append(b, byte(c))
	}
	return string(b)
}

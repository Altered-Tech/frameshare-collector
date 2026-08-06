package hardware

import (
	"context"
	"encoding/json"
)

type win32ComputerSystemProduct struct {
	Vendor string `json:"Vendor"`
	Name   string `json:"Name"`
}

const computerSystemProductScript = `Get-CimInstance Win32_ComputerSystemProduct | ` +
	`Select-Object Vendor,Name | ConvertTo-Json`

func collectDevice(ctx context.Context) (DeviceInfo, error) {
	raw, err := runPowerShellJSON(ctx, computerSystemProductScript)
	if err != nil {
		return DeviceInfo{}, err
	}

	var products []win32ComputerSystemProduct
	if err := json.Unmarshal(normalizeJSONArray(raw), &products); err != nil {
		return DeviceInfo{}, err
	}
	if len(products) == 0 {
		return DeviceInfo{}, nil
	}

	vendor := products[0].Vendor
	model := products[0].Name
	return DeviceInfo{
		Vendor:        vendor,
		Model:         model,
		KnownHandheld: matchHandheld(vendor, model),
	}, nil
}

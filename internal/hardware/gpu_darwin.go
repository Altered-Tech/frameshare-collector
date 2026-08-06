package hardware

import "context"

func collectGPUs(ctx context.Context) ([]GPUInfo, error) {
	data, err := fetchDisplaysData(ctx)
	if err != nil {
		return nil, err
	}

	var gpus []GPUInfo
	for _, g := range data.SPDisplaysDataType {
		name := g.Model
		if name == "" {
			name = g.Name
		}
		gpus = append(gpus, GPUInfo{
			Name:   name,
			Vendor: g.Vendor,
		})
	}
	return gpus, nil
}

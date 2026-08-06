package hardware

import (
	"context"

	"github.com/shirou/gopsutil/v4/cpu"
)

func collectCPU(ctx context.Context) (CPUInfo, error) {
	infos, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return CPUInfo{}, err
	}

	physical, err := cpu.CountsWithContext(ctx, false)
	if err != nil {
		return CPUInfo{}, err
	}
	logical, err := cpu.CountsWithContext(ctx, true)
	if err != nil {
		return CPUInfo{}, err
	}

	result := CPUInfo{
		PhysicalCores: physical,
		LogicalCores:  logical,
	}
	if len(infos) > 0 {
		result.Model = infos[0].ModelName
		result.Vendor = infos[0].VendorID
		result.MaxMHz = infos[0].Mhz
	}
	return result, nil
}

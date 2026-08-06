package hardware

import (
	"context"

	"github.com/shirou/gopsutil/v4/mem"
)

const bytesPerGB = 1024 * 1024 * 1024

func collectMemory(ctx context.Context) (MemInfo, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return MemInfo{}, err
	}
	return MemInfo{
		TotalBytes: vm.Total,
		TotalGB:    round2(float64(vm.Total) / bytesPerGB),
	}, nil
}

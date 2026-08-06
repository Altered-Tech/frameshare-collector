package hardware

import (
	"context"
	"fmt"
	"time"
)

// Collect gathers a full local hardware snapshot. It does not make any
// network calls.
func Collect(ctx context.Context) (Snapshot, error) {
	snap := Snapshot{CollectedAt: time.Now().UTC()}

	osInfo, err := collectOS(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect os: %w", err)
	}
	snap.OS = osInfo

	cpuInfo, err := collectCPU(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect cpu: %w", err)
	}
	snap.CPU = cpuInfo

	memInfo, err := collectMemory(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect memory: %w", err)
	}
	snap.Memory = memInfo

	storage, err := collectStorage(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect storage: %w", err)
	}
	snap.Storage = storage

	// GPU and display detection shell out to OS-specific tools that may be
	// missing or fail on unusual configurations; treat failures as
	// non-fatal so the rest of the snapshot still gets written.
	if gpus, err := collectGPUs(ctx); err == nil {
		snap.GPUs = gpus
	}
	if displays, err := collectDisplays(ctx); err == nil {
		snap.Displays = displays
	}

	return snap, nil
}

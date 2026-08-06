package hardware

import (
	"context"
	"runtime"

	"github.com/shirou/gopsutil/v4/host"
)

func collectOS(ctx context.Context) (OSInfo, error) {
	info, err := host.InfoWithContext(ctx)
	if err != nil {
		return OSInfo{}, err
	}
	name := info.Platform
	if info.OS == "darwin" {
		name = "macOS" // gopsutil reports the raw platform string "darwin" here
	}
	return OSInfo{
		Platform:      info.OS,
		Family:        info.PlatformFamily,
		Name:          name,
		Version:       info.PlatformVersion,
		KernelVersion: info.KernelVersion,
		Arch:          runtime.GOARCH,
	}, nil
}

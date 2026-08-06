package hardware

import (
	"bytes"
	"context"
	"os/exec"
)

// collectDevice reports the Mac model identifier (e.g. "MacBookPro18,1").
// Macs aren't gaming handhelds, so KnownHandheld is always empty here.
func collectDevice(ctx context.Context) (DeviceInfo, error) {
	cmd := exec.CommandContext(ctx, "sysctl", "-n", "hw.model")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return DeviceInfo{}, err
	}
	model := bytes.TrimSpace(out.Bytes())
	return DeviceInfo{
		Vendor: "Apple",
		Model:  string(model),
	}, nil
}

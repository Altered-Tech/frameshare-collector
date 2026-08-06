package hardware

import (
	"context"
	"os"
	"strings"
)

// collectDevice reads DMI/SMBIOS strings exposed by the kernel under
// /sys/class/dmi/id. These are the same fields `dmidecode` reads and are
// normally world-readable (unlike product_serial/board_serial, which
// require root).
func collectDevice(ctx context.Context) (DeviceInfo, error) {
	vendor := readDMIField("sys_vendor")
	model := readDMIField("product_name")

	return DeviceInfo{
		Vendor:        vendor,
		Model:         model,
		KnownHandheld: matchHandheld(vendor, model),
	}, nil
}

func readDMIField(field string) string {
	data, err := os.ReadFile("/sys/class/dmi/id/" + field)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

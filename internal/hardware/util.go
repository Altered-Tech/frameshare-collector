package hardware

import "math"

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// diskRole summarizes a disk's IsOSDrive/IsInstallDrive flags into a
// human-readable label.
func diskRole(isOSDrive, isInstallDrive bool) string {
	switch {
	case isOSDrive && isInstallDrive:
		return "OS+Install Drive"
	case isOSDrive:
		return "OS Drive"
	case isInstallDrive:
		return "Install Drive"
	default:
		return ""
	}
}

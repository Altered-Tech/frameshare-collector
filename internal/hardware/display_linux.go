package hardware

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	connectedRe = regexp.MustCompile(`^(\S+) connected( primary)?`)
	modeRe      = regexp.MustCompile(`(\d+)x(\d+)\s+([\d.]+)\*`)
)

// collectDisplays parses `xrandr --current` output. Steam Deck and most
// desktop Linux setups run X11 or XWayland with xrandr available; if it's
// missing (pure Wayland compositor with no XWayland), displays are omitted.
func collectDisplays(ctx context.Context) ([]Display, error) {
	cmd := exec.CommandContext(ctx, "xrandr", "--current")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var displays []Display
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var current *Display
	for scanner.Scan() {
		line := scanner.Text()
		if m := connectedRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				displays = append(displays, *current)
			}
			current = &Display{
				Name:      m[1],
				IsPrimary: m[2] != "",
			}
			continue
		}
		if current == nil || !strings.HasPrefix(line, " ") {
			continue
		}
		if m := modeRe.FindStringSubmatch(line); m != nil {
			width, _ := strconv.Atoi(m[1])
			height, _ := strconv.Atoi(m[2])
			refresh, _ := strconv.ParseFloat(m[3], 64)
			current.WidthPx = width
			current.HeightPx = height
			current.RefreshHz = refresh
		}
	}
	if current != nil {
		displays = append(displays, *current)
	}

	for i := range displays {
		pnpID, model := readConnectorEDID(displays[i].Name)
		if pnpID != "" {
			displays[i].MonitorVendor = resolvePNPVendor(pnpID)
		}
		displays[i].MonitorModel = model
	}

	return displays, scanner.Err()
}

// readConnectorEDID reads and decodes the EDID for the DRM connector
// matching an xrandr connector name (e.g. "DP-1"). Sysfs exposes each
// connector as /sys/class/drm/cardN-<connector>/edid; xrandr's connector
// name is that path with the "cardN-" prefix stripped, so match on suffix.
func readConnectorEDID(connector string) (pnpID, model string) {
	entries, err := os.ReadDir("/sys/class/drm")
	if err != nil {
		return "", ""
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), "-"+connector) {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/sys/class/drm", e.Name(), "edid"))
		if err != nil {
			return "", ""
		}
		return decodeEDID(data)
	}
	return "", ""
}

package hardware

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
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
	return displays, scanner.Err()
}

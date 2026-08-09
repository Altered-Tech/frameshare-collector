package hardware

import (
	"context"
	"regexp"
	"strconv"
)

// matches strings like "1512 x 982 @ 120.00Hz", "1920 x 1080", or "3024 x 1964"
var resolutionRe = regexp.MustCompile(`(\d+)\s*x\s*(\d+)(?:\s*@\s*([\d.]+)\s*Hz)?`)

func collectDisplays(ctx context.Context) ([]Display, error) {
	data, err := fetchDisplaysData(ctx)
	if err != nil {
		return nil, err
	}

	var displays []Display
	for _, g := range data.SPDisplaysDataType {
		for _, n := range g.Ndrvs {
			// Resolution carries the refresh rate; on HiDPI displays its
			// width/height are scaled logical points, not real pixels, so
			// prefer Pixels (native panel size) when present.
			resMatch := resolutionRe.FindStringSubmatch(n.Resolution)
			pxMatch := resolutionRe.FindStringSubmatch(n.Pixels)
			sizeMatch := pxMatch
			if sizeMatch == nil {
				sizeMatch = resMatch
			}
			if sizeMatch == nil {
				continue
			}

			width, _ := strconv.Atoi(sizeMatch[1])
			height, _ := strconv.Atoi(sizeMatch[2])
			var refresh float64
			if resMatch != nil && resMatch[3] != "" {
				refresh, _ = strconv.ParseFloat(resMatch[3], 64)
			}
			displays = append(displays, Display{
				Name:      n.Name,
				WidthPx:   width,
				HeightPx:  height,
				RefreshHz: refresh,
				IsPrimary: n.Main == "spdisplays_yes",
			})
		}
	}

	// ioreg enumerates IODisplay services independently of
	// system_profiler's per-GPU Ndrvs grouping, and neither exposes a key
	// to correlate entries precisely. When the counts match (the common
	// case: one row per active monitor) pair them up positionally;
	// otherwise leave monitor identity empty rather than risk mislabeling
	// a display.
	edids := fetchDisplayEDIDs(ctx)
	if len(edids) == len(displays) {
		for i, raw := range edids {
			pnpID, model := decodeEDID(raw)
			if pnpID != "" {
				displays[i].MonitorVendor = resolvePNPVendor(pnpID)
			}
			displays[i].MonitorModel = model
		}
	}

	return displays, nil
}

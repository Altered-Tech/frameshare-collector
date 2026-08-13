package hardware

import (
	"context"
	"encoding/hex"
	"os/exec"
	"regexp"
)

var edidPropertyRe = regexp.MustCompile(`"IODisplayEDID"\s*=\s*<([0-9a-fA-F]+)>`)

// fetchDisplayEDIDs returns the raw EDID block for each active IODisplay
// service, in ioreg's enumeration order.
func fetchDisplayEDIDs(ctx context.Context) [][]byte {
	cmd := exec.CommandContext(ctx, "ioreg", "-c", "IODisplay", "-r", "-l")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var edids [][]byte
	for _, m := range edidPropertyRe.FindAllSubmatch(out, -1) {
		if raw, err := hex.DecodeString(string(m[1])); err == nil {
			edids = append(edids, raw)
		}
	}
	return edids
}

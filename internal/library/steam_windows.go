package library

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

// windowsSteamRootFallbacks are checked if the registry lookup fails or the
// key is missing (e.g. a very old install that predates SteamPath being
// written).
var windowsSteamRootFallbacks = []string{
	`C:\Program Files (x86)\Steam`,
	`C:\Program Files\Steam`,
}

func steamRoot(ctx context.Context) string {
	if root := steamRootFromRegistry(ctx); root != "" && isSteamRoot(root) {
		return root
	}
	for _, candidate := range windowsSteamRootFallbacks {
		if isSteamRoot(candidate) {
			return candidate
		}
	}
	return ""
}

// steamRootFromRegistry reads the SteamPath value the Steam installer
// writes to HKCU\Software\Valve\Steam on every install.
func steamRootFromRegistry(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "reg", "query", `HKCU\Software\Valve\Steam`, "/v", "SteamPath")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "SteamPath" {
			return filepath.FromSlash(strings.Join(fields[2:], " "))
		}
	}
	return ""
}

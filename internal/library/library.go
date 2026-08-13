// Package library detects installed game libraries on the local machine.
// Steam is the only supported launcher today; other launchers (Epic, GOG,
// etc) can be added later by following the same per-OS detection pattern
// used in internal/hardware/device_*.go.
package library

import "context"

// Source identifies which game launcher/store a Library belongs to.
type Source string

const SourceSteam Source = "steam"

// Library is one game library root (e.g. a Steam library folder) and the
// titles installed under it. A single launcher can have multiple
// libraries, e.g. Steam libraries spread across several disks.
type Library struct {
	Source Source `json:"source"`
	Path   string `json:"path"`
	Games  []Game `json:"games"`
}

// Game is a single installed title within a Library.
type Game struct {
	AppID       string `json:"app_id,omitempty"`
	Name        string `json:"name"`
	InstallPath string `json:"install_path"`
	SizeBytes   uint64 `json:"size_bytes,omitempty"`
}

// Detect finds installed game libraries across supported launchers. A
// launcher that isn't installed is not an error; its libraries are simply
// omitted from the result.
func Detect(ctx context.Context) ([]Library, error) {
	return detectSteamLibraries(ctx)
}

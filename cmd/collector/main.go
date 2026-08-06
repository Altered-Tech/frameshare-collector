// Command collector detects the local machine's hardware (CPU, GPU, RAM,
// display, OS, storage) and writes it to a local JSON file. Phase 1 is
// local-only: no network calls, no UI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alteredtech/frameshare-collector/internal/hardware"
)

func main() {
	outDir := flag.String("out", ".", "directory to write the snapshot JSON file to")
	flag.Parse()

	ctx := context.Background()
	snap, err := hardware.Collect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling snapshot: %v\n", err)
		os.Exit(1)
	}

	filename := fmt.Sprintf("hardware-snapshot-%s.json", snap.CollectedAt.Format("20060102-150405"))
	outPath := filepath.Join(*outDir, filename)
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing snapshot: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Hardware snapshot written to %s\n\n", outPath)
	printSummary(snap)
}

func printSummary(snap hardware.Snapshot) {
	fmt.Printf("OS:      %s %s (%s, %s)\n", snap.OS.Name, snap.OS.Version, snap.OS.Platform, snap.OS.Arch)
	fmt.Printf("CPU:     %s (%d cores / %d threads)\n", snap.CPU.Model, snap.CPU.PhysicalCores, snap.CPU.LogicalCores)
	fmt.Printf("Memory:  %.1f GB\n", snap.Memory.TotalGB)
	for _, g := range snap.GPUs {
		fmt.Printf("GPU:     %s\n", g.Name)
	}
	for _, d := range snap.Displays {
		fmt.Printf("Display: %dx%d @ %.0fHz%s\n", d.WidthPx, d.HeightPx, d.RefreshHz, primaryLabel(d.IsPrimary))
	}
	for _, s := range snap.Storage {
		fmt.Printf("Storage: %s (%s) %.1f GB\n", s.Device, s.Mountpoint, s.TotalGB)
	}
}

func primaryLabel(isPrimary bool) string {
	if isPrimary {
		return " [primary]"
	}
	return ""
}

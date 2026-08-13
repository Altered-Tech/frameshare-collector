// Command collector detects the local machine's hardware (CPU, GPU, RAM,
// display, OS, storage) and writes it to a local JSON file. Phase 1 is
// local-only: no network calls, no UI.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alteredtech/frameshare-collector/internal/hardware"
	"github.com/alteredtech/frameshare-collector/internal/library"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3"; the
// release workflow injects the pushed tag. Local `go build`/`go run` leave
// it at "dev".
var version = "dev"

// snapshotOutput is the JSON shape written to disk: the hardware snapshot,
// plus the game chosen via -select-game (if any). SelectedGame is what a
// future game-settings collector reads to know which title to inspect and
// where it's installed.
type snapshotOutput struct {
	hardware.Snapshot
	SelectedGame *selectedGame `json:"selected_game,omitempty"`
}

type selectedGame struct {
	library.Game
	Source library.Source `json:"source,omitempty"`
}

func main() {
	outDir := flag.String("out", ".", "directory to write the snapshot JSON file to")
	installPath := flag.String("install-path", "", "game install directory; the physical disk containing it is reported as the install drive. Ignored if -select-game is set")
	selectGameFlag := flag.Bool("select-game", false, "list installed games and choose one; sets -install-path to its install directory and records the choice in the snapshot")
	showVersion := flag.Bool("version", false, "print the collector version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	ctx := context.Background()

	var selected *selectedGame
	if *selectGameFlag {
		libs, err := library.Detect(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		game, source, err := chooseGame(libs, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		selected = &selectedGame{Game: game, Source: source}
		*installPath = game.InstallPath
	}

	snap, err := hardware.Collect(ctx, *installPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	snap.CollectorVersion = version

	out := snapshotOutput{Snapshot: snap, SelectedGame: selected}
	data, err := json.MarshalIndent(out, "", "  ")
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
	printSummary(snap, selected)
}

// chooseGame lists every installed game across all detected libraries and
// prompts the user (via in/out) to pick one by number.
func chooseGame(libs []library.Library, in io.Reader, out io.Writer) (library.Game, library.Source, error) {
	type entry struct {
		game   library.Game
		source library.Source
	}
	var entries []entry
	for _, lib := range libs {
		for _, g := range lib.Games {
			entries = append(entries, entry{game: g, source: lib.Source})
		}
	}
	if len(entries) == 0 {
		return library.Game{}, "", fmt.Errorf("no installed games found")
	}

	fmt.Fprintln(out, "Installed games:")
	for i, e := range entries {
		fmt.Fprintf(out, "  %d) %s [%s]\n", i+1, e.game.Name, e.source)
	}
	fmt.Fprint(out, "Select a game by number: ")

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return library.Game{}, "", fmt.Errorf("read selection: %w", err)
	}
	choice, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || choice < 1 || choice > len(entries) {
		return library.Game{}, "", fmt.Errorf("invalid selection %q", strings.TrimSpace(line))
	}

	chosen := entries[choice-1]
	return chosen.game, chosen.source, nil
}

func printSummary(snap hardware.Snapshot, selected *selectedGame) {
	fmt.Printf("Version: %s\n", snap.CollectorVersion)
	fmt.Printf("Device:  %s %s%s\n", snap.Device.Vendor, snap.Device.Model, handheldLabel(snap.Device.KnownHandheld))
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
		fmt.Printf("Storage: %s (%s) %.1f GB%s\n", s.Model, s.Type, s.TotalGB, roleLabel(s.Role))
	}
	if selected != nil {
		fmt.Printf("Game:    %s [%s] -> %s\n", selected.Name, selected.Source, selected.InstallPath)
	}
}

func primaryLabel(isPrimary bool) string {
	if isPrimary {
		return " [primary]"
	}
	return ""
}

func handheldLabel(knownHandheld string) string {
	if knownHandheld == "" {
		return ""
	}
	return fmt.Sprintf(" [%s]", knownHandheld)
}

func roleLabel(role string) string {
	if role == "" {
		return ""
	}
	return fmt.Sprintf(" [%s]", role)
}

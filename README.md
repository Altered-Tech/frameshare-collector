# frameshare-collector

## Requirements

- Go 1.26+
- Linux only: `lspci` and `xrandr` on `PATH` for GPU/display detection
  (usually already present on desktop distros; may be missing on a bare
  Wayland-only setup)

## Build

```sh
go build -o collector ./cmd/collector
```

## Run

```sh
go run ./cmd/collector
```

or, using the built binary:

```sh
./collector
```

This writes `hardware-snapshot-<timestamp>.json` to the current directory
and prints a short summary to stdout.

Use `-out` to choose a different output directory:

```sh
./collector -out ~/Desktop
```

## Releases

Versioning and releases are automated by [semantic-release](https://github.com/semantic-release/semantic-release):
every push to `main` is analyzed for [Conventional Commits](https://www.conventionalcommits.org/)
(`fix:`, `feat:`, `BREAKING CHANGE:`/`type!:`) since the last release, and if
any are found, it tags the next semantic version, builds binaries for every
platform, and publishes a GitHub Release with them attached. Commits that
don't follow the convention don't trigger a release. See `.releaserc.json`
and `.github/workflows/release.yml`.

## Notes

- GPU and display detection shell out to OS-specific tools (`system_profiler`
  on macOS, PowerShell/CIM on Windows, `lspci`/`xrandr` on Linux). If those
  fail or aren't available, the rest of the snapshot is still written with
  those fields left empty.
- Storage entries are filtered to real local physical volumes — network
  shares and OS-internal sub-volumes are excluded.
- Device identification reads DMI/SMBIOS strings (`/sys/class/dmi/id` on
  Linux, `Win32_ComputerSystemProduct` on Windows, `hw.model` on macOS) and
  matches them against a manually maintained list of known gaming handhelds
  (Steam Deck, ROG Ally, Legion Go, etc). Unrecognized devices still report
  their raw vendor/model; `known_handheld` is only set on a match.

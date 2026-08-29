# frameshare-collector

## Requirements

- Go 1.26+
- Linux only: `lspci` and `xrandr` on `PATH` for GPU/display detection
  (usually already present on desktop distros; may be missing on a bare
  Wayland-only setup)
- Only for `cmd/gui-spike` (see below): a C compiler, SDL2 development
  headers, and Fyne's own native build dependencies. `cmd/collector` has
  no such requirement.
  - Linux: `gcc pkg-config libsdl2-dev libgl1-mesa-dev xorg-dev libxkbcommon-dev`
    (Debian/Ubuntu package names; `libxkbcommon-dev` is needed by GLFW's
    keyboard handling even on X11, not just Wayland)
  - macOS: `brew install pkg-config sdl2` (Xcode Command Line Tools cover
    the rest)

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

## GUI navigation spike

`cmd/gui-spike` is a throwaway Fyne app (see #7's framework decision and
#36) that proves out gamepad-driven navigation before more UI work builds
on it: a `List` standing in for the game library, plus an `Entry` and
`Button` standing in for the auth flow, all navigable with a controller
via `internal/controllerinput`'s SDL2-based input translation, alongside
unmodified mouse/keyboard support.

```sh
go run ./cmd/gui-spike
```

Build a standalone binary the same way (`go build`, not `make` -- the
Makefile only targets `cmd/collector`):

```sh
go build -o gui-spike ./cmd/gui-spike
```

**On Steam Deck, launch it as a Steam game (a non-Steam game entry),
not directly from a Desktop Mode terminal.** While the desktop session
itself is focused, Steam's own controller configuration translates
gamepad input (e.g. D-pad) into real, OS-level synthetic keyboard events
system-wide, for general desktop navigation. Those arrive at the app
independently of, and simultaneously with, this app's own SDL-based
polling -- both end up calling the same widget's key handling, so a
single physical press can visibly double-fire (e.g. the game list
advancing by two entries per D-pad tap) even though this app's own
input pipeline only sees and processes it once. Launching it as an
actual (non-Steam) game switches the controller out of that desktop
translation mode, leaving only this app's own SDL polling active.

It launches full-screen automatically under a gamescope session (Gaming
Mode and similar SteamOS/gamescope-session-based handhelds) -- detected
via the `XDG_CURRENT_DESKTOP`/`XDG_SESSION_DESKTOP=gamescope` env vars
gamescope-session sets, not the `SteamDeck=1` var SteamOS also exports
(which is present in Desktop Mode too, where a windowed dev session
should stay windowed). No Launch Options entry needed for this. Confirmed
on hardware: without it, the app's small default window gets pillarboxed
by gamescope rather than stretched or auto-expanded, so an explicit
fullscreen request is required, not optional. Pass `-fullscreen`/
`-fullscreen=false` manually only to override the auto-detected default,
e.g. for local testing.

### Gaming Mode verification checklist (#34)

Full-screen launch is confirmed working on hardware. Still to verify:

- [ ] Text size, contrast, and widget scaling are legible at the Deck's
      native 1280x800 resolution from typical Gaming Mode (couch/handheld)
      viewing distance.
- [ ] Fyne's default theme/scale renders sensibly with no manual DPI
      tuning; if not, `fyne.Settings.SetScale` (or `FYNE_SCALE`) is the
      knob to reach for.
- [x] The window is full-screen/borderless in Gaming Mode -- confirmed,
      via the auto-detected `-fullscreen` default (see above).
- [ ] Focus highlighting -- the only visual indicator of which widget a
      gamepad-only user is on -- is clearly visible from Gaming Mode
      viewing distance.

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

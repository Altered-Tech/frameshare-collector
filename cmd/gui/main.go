// Command gui is FrameShare's controller-driven GUI: pick an installed
// game (or skip to review hardware alone), then browse the merged
// hardware + game-settings profile detected for it.
//
// It's the production build-out of the navigation approach cmd/gui-spike
// proved out for issue #33: a background goroutine polls SDL2's
// GameController API via internal/controllerinput and replays each
// translated Action as a synthetic key event against whatever widget
// currently has focus, using Fyne's own focus system rather than
// building gamepad-aware navigation from scratch. Mouse and keyboard are
// untouched: Fyne handles those natively.
//
// This first pass (issue #9) is view-only. Editing fields (#10) and
// confirming/saving the reviewed profile (#11) build on top of it.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/alteredtech/frameshare-collector/internal/controllerinput"
	"github.com/alteredtech/frameshare-collector/internal/gamesettings"
	_ "github.com/alteredtech/frameshare-collector/internal/gamesettings/all" // register every supported title's Parser
	"github.com/alteredtech/frameshare-collector/internal/hardware"
	"github.com/alteredtech/frameshare-collector/internal/library"
	"github.com/alteredtech/frameshare-collector/internal/profile"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3", the
// same way cmd/collector's is; local `go run`/`go build` leave it "dev".
var version = "dev"

// inGamescopeSession reports whether the process is running inside a
// gamescope compositor session -- what Steam Deck's Gaming Mode (and
// other SteamOS/gamescope-session-based handhelds) actually runs under.
// Ported from cmd/gui-spike, where this distinction (over the SteamDeck=1
// env var, also set in Desktop Mode) was validated on hardware for #34.
func inGamescopeSession() bool {
	return os.Getenv("XDG_CURRENT_DESKTOP") == "gamescope" || os.Getenv("XDG_SESSION_DESKTOP") == "gamescope"
}

// hardwareOnlyLabel is the synthetic first entry in the game picker,
// letting a user review hardware alone without any title selected.
const hardwareOnlyLabel = "(hardware only, no game)"

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	fullscreen := flag.Bool("fullscreen", inGamescopeSession(), "launch full-screen instead of the default small window; defaults to true automatically under a gamescope session (Steam Deck Gaming Mode and similar), false elsewhere, so end users never need to set this themselves")
	flag.Parse()

	a := app.NewWithID("com.alteredtech.frameshare-collector.gui")
	w := a.NewWindow("FrameShare Collector")

	poller, err := controllerinput.NewPoller(log.Printf)
	if err != nil {
		log.Fatalf("controller input unavailable: %v", err)
	}
	defer poller.Close()

	nav := newNavigator(w)
	nav.showGamePicker()

	go func() {
		for action := range poller.Actions() {
			fyne.Do(func() {
				nav.dispatch(action)
			})
		}
	}()

	if *fullscreen {
		w.SetFullScreen(true)
	} else {
		w.Resize(fyne.NewSize(480, 360))
	}
	w.ShowAndRun()
}

// navigator owns the window's current screen (its content and the set of
// widgets a gamepad can cycle focus across) and dispatches translated
// controller Actions against whichever screen is showing. Only one
// screen is ever live at a time, so this -- rather than each screen
// managing its own focus ring -- is the natural place for the
// focus-cycling logic gui-spike validated on hardware for #33/#37.
type navigator struct {
	win        fyne.Window
	focusables []fyne.Focusable
	focusIndex int
}

func newNavigator(w fyne.Window) *navigator {
	return &navigator{win: w}
}

// setScreen replaces the window's content and focus ring, and focuses
// the first focusable widget so a controller-only user always lands
// somewhere navigable without needing a mouse click first.
func (n *navigator) setScreen(content fyne.CanvasObject, focusables []fyne.Focusable) {
	n.win.SetContent(content)
	n.focusables = focusables
	n.focusIndex = 0
	if len(focusables) > 0 {
		n.win.Canvas().Focus(focusables[0])
	}
}

func (n *navigator) cycleFocus(delta int) {
	if len(n.focusables) == 0 {
		return
	}
	n.focusIndex = (n.focusIndex + delta + len(n.focusables)) % len(n.focusables)
	n.win.Canvas().Focus(n.focusables[n.focusIndex])
}

// dispatch handles FocusNext/FocusPrevious via cycleFocus and replays
// every other action as a synthetic key event against whatever widget
// currently has focus, the same pattern gui-spike validated on hardware:
// List moves its highlight on Up/Down and selects on Space.
func (n *navigator) dispatch(action controllerinput.Action) {
	switch action {
	case controllerinput.ActionFocusNext:
		n.cycleFocus(1)
		return
	case controllerinput.ActionFocusPrevious:
		n.cycleFocus(-1)
		return
	}

	key := actionKey(action)
	if key == "" {
		return
	}
	if focused := n.win.Canvas().Focused(); focused != nil {
		focused.TypedKey(&fyne.KeyEvent{Name: key})
	}
}

func actionKey(action controllerinput.Action) fyne.KeyName {
	switch action {
	case controllerinput.ActionUp:
		return fyne.KeyUp
	case controllerinput.ActionDown:
		return fyne.KeyDown
	case controllerinput.ActionLeft:
		return fyne.KeyLeft
	case controllerinput.ActionRight:
		return fyne.KeyRight
	case controllerinput.ActionActivate:
		return fyne.KeySpace
	default:
		return ""
	}
}

// gameEntries flattens every installed game across all detected
// libraries into a single pickable list, mirroring cmd/collector's own
// gameEntries/pickGame helpers.
type gameEntry struct {
	game   library.Game
	source library.Source
}

func (e gameEntry) label() string {
	return fmt.Sprintf("%s [%s]", e.game.Name, e.source)
}

func detectGameEntries(ctx context.Context) ([]gameEntry, error) {
	libs, err := library.Detect(ctx)
	if err != nil {
		return nil, err
	}
	var entries []gameEntry
	for _, lib := range libs {
		for _, g := range lib.Games {
			entries = append(entries, gameEntry{game: g, source: lib.Source})
		}
	}
	return entries, nil
}

// showGamePicker renders the first screen: a List of installed games
// (plus hardwareOnlyLabel) that starts hardware/game-settings collection
// on Activate and hands the resulting Profile to showReview.
func (n *navigator) showGamePicker() {
	status := widget.NewLabel("Detecting installed games...")
	hint := widget.NewLabel("D-pad/stick: navigate  |  A: select")
	content := container.NewBorder(nil, container.NewVBox(status, hint), nil, nil, widget.NewLabel(""))
	n.setScreen(content, nil)

	go func() {
		entries, err := detectGameEntries(context.Background())
		fyne.Do(func() {
			if err != nil {
				status.SetText(fmt.Sprintf("Could not detect game libraries: %v", err))
			}
			n.renderGamePicker(entries, status, hint)
		})
	}()
}

func (n *navigator) renderGamePicker(entries []gameEntry, status, hint *widget.Label) {
	labels := make([]string, len(entries)+1)
	labels[0] = hardwareOnlyLabel
	for i, e := range entries {
		labels[i+1] = e.label()
	}

	list := widget.NewList(
		func() int { return len(labels) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(labels[id])
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id == 0 {
			n.collectAndReview(nil, status)
			return
		}
		entry := entries[id-1]
		n.collectAndReview(&entry, status)
	}

	content := container.NewBorder(nil, container.NewVBox(status, hint), nil, nil, list)
	n.setScreen(content, []fyne.Focusable{list})
}

// collectAndReview runs hardware and (if entry is non-nil) game-settings
// detection, then switches to the review screen. entry is nil when the
// user picked hardwareOnlyLabel.
func (n *navigator) collectAndReview(entry *gameEntry, status *widget.Label) {
	status.SetText("Collecting profile...")

	go func() {
		ctx := context.Background()
		installPath := ""
		if entry != nil {
			installPath = entry.game.InstallPath
		}

		snap, err := hardware.Collect(ctx, installPath)
		if err != nil {
			fyne.Do(func() { status.SetText(fmt.Sprintf("Hardware detection failed: %v", err)) })
			return
		}
		snap.CollectorVersion = version

		var gameSettings *gamesettings.GameProfile
		if entry != nil {
			gameSettings = profile.CollectGameSettings(entry.game, entry.source, func(msg string) {
				fyne.Do(func() { status.SetText(msg) })
			})
		}

		p := profile.Merge(snap, gameSettings)
		fyne.Do(func() { n.showReview(p) })
	}()
}

// showReview renders every field of p, flattened via Profile.Fields, as
// a read-only, controller-navigable List. Editing (#10) and confirm/save
// (#11) are not yet wired in -- this issue (#9) is view-only.
func (n *navigator) showReview(p *profile.Profile) {
	fields := p.Fields()
	hint := widget.NewLabel("D-pad/stick: browse")

	list := widget.NewList(
		func() int { return len(fields) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			f := fields[id]
			text := fmt.Sprintf("%s: %s", f.Label, f.Value)
			if f.Overridden {
				text += " *"
			}
			obj.(*widget.Label).SetText(text)
		},
	)

	content := container.NewBorder(nil, hint, nil, nil, list)
	n.setScreen(content, []fyne.Focusable{list})
}

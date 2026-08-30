// Command gui is FrameShare's controller-driven GUI: pick an installed
// game (or skip to review hardware alone), browse the merged hardware +
// game-settings profile detected for it (#9), correct any field that
// came back wrong (#10), and confirm & save the result to a local file
// (#11, see internal/profile.Save) -- no network calls; backend
// submission is Phase 4.
//
// It's the production build-out of the navigation approach cmd/gui-spike
// proved out for issue #33: a background goroutine polls SDL2's
// GameController API via internal/controllerinput and replays each
// translated Action as a synthetic key event against whatever widget
// currently has focus, using Fyne's own focus system rather than
// building gamepad-aware navigation from scratch. Mouse and keyboard are
// untouched: Fyne handles those natively.
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
	// busy guards collectAndReview against a second Activate press
	// re-entering it while a collection is already in flight -- without
	// it, a controller-only user seeing no immediate feedback (hardware
	// detection can take real time, e.g. Windows' PowerShell-based GPU
	// detection) could plausibly press Activate again on a different row,
	// racing two collections against each other with whichever finishes
	// last silently winning and no indication either happened.
	busy bool
	// onBack is invoked on ActionBack, if set. What "back" means changes
	// with context -- return to the previous screen, cancel an
	// in-progress field edit -- so each screen/mode sets it to whatever
	// is correct for the state the user is currently in, rather than
	// this being a single fixed action. nil means Back does nothing,
	// which is correct for the first screen shown (there's nothing
	// before it to go back to).
	onBack func()
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
	case controllerinput.ActionBack:
		if n.onBack != nil {
			n.onBack()
		}
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
	n.onBack = nil // first screen -- nothing before it to go back to

	status := widget.NewLabel("Detecting installed games...")
	status.Wrapping = fyne.TextWrapWord
	hint := widget.NewLabel("D-pad/stick: navigate  |  A: select")
	content := container.NewBorder(nil, container.NewVBox(status, hint), nil, nil, widget.NewLabel(""))
	n.setScreen(content, nil)

	go func() {
		entries, err := detectGameEntries(context.Background())
		fyne.Do(func() {
			if err != nil {
				log.Printf("gui: detecting game libraries: %v", err)
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
// user picked hardwareOnlyLabel. A second call while one is already in
// flight (see busy's doc comment) is ignored rather than racing a second
// collection against the first.
func (n *navigator) collectAndReview(entry *gameEntry, status *widget.Label) {
	if n.busy {
		return
	}
	n.busy = true
	status.SetText("Collecting profile...")

	go func() {
		ctx := context.Background()
		installPath := ""
		if entry != nil {
			installPath = entry.game.InstallPath
		}

		snap, err := hardware.Collect(ctx, installPath)
		if err != nil {
			log.Printf("gui: hardware detection failed: %v", err)
			fyne.Do(func() {
				n.busy = false
				status.SetText(fmt.Sprintf("Hardware detection failed: %v", err))
			})
			return
		}
		snap.CollectorVersion = version

		// gameSettingsWarning carries CollectGameSettings' reason for a
		// nil GameProfile (e.g. no parser registered for this title)
		// across the screen switch below. It can't just be shown via
		// status here: status belongs to the game-picker screen, and
		// showReview's own fyne.Do call -- queued immediately after,
		// executing right behind this one -- replaces the whole window
		// content before a human could ever see it flash by. Passing it
		// into showReview so it can display it in the review screen's
		// own status label is the fix -- this is what a game with no
		// settings parser actually looks like today, not a silent gap:
		// only 3 titles (Team Fortress 2, Terraria, Stray) have one
		// registered, so it's the common case for any other game.
		var gameSettingsWarning string
		var gameSettings *gamesettings.GameProfile
		if entry != nil {
			gameSettings = profile.CollectGameSettings(entry.game, entry.source, func(msg string) {
				log.Printf("gui: %s", msg)
				gameSettingsWarning = msg
			})
		}

		p := profile.Merge(snap, gameSettings)
		fyne.Do(func() {
			n.busy = false
			n.showReview(p, gameSettingsWarning)
		})
	}()
}

// showReview renders every field of p, flattened via Profile.Fields, as
// a controller-navigable List; Activate on a row edits that field (#10)
// via a shared Entry + Save button below the list, rather than swapping
// widgets in place within the row -- Fyne's List recycles its item
// CanvasObjects as it scrolls, so a per-row Entry could be silently
// reassigned to a different field mid-edit. warning, if non-empty, is
// shown up front -- it's why GameSettings came back nil (e.g. no parser
// registered for the selected title), which would otherwise be
// invisible: p simply has fewer fields than the user might expect, with
// nothing on this screen explaining why. A Confirm & Save button (#11)
// persists p, edits included, to a local file; there's no network call
// and no further screen after that -- Phase 4 owns backend submission.
func (n *navigator) showReview(p *profile.Profile, warning string) {
	n.onBack = func() { n.showGamePicker() }

	fields := p.Fields()
	var editingField profile.Field

	const (
		browseHint      = "D-pad/stick: browse  |  A: edit a field  |  LB/RB: switch List/Confirm  |  B: back to game list"
		editHint        = "LB/RB: switch Entry/Save  |  Steam+X: on-screen keyboard  |  A on Save: apply  |  B: cancel edit"
		browseBackLabel = "Back to Game List"
		cancelLabel     = "Cancel"
	)

	status := widget.NewLabel(warning)
	// Wrapping (off by default) matters here specifically: a game
	// settings collection error wraps a full config file path plus the
	// underlying OS error, easily 100+ characters -- left unwrapped, a
	// Label's width is exactly its text's width, so Fyne's layout grows
	// the whole window to fit one long line instead of the window
	// staying put and the text wrapping within it.
	status.Wrapping = fyne.TextWrapWord
	if warning == "" {
		status.Hide()
	}
	editEntry := widget.NewEntry()
	editEntry.Hide()
	saveButton := widget.NewButton("Save", nil)
	saveButton.Hide()
	confirmButton := widget.NewButton("Confirm & Save", nil)
	hint := widget.NewLabel(browseHint)
	// backButton exists for mouse/keyboard: those bypass n.dispatch
	// entirely (Fyne handles them natively, see the package doc comment),
	// so ActionBack's gamepad mapping alone leaves them with no way to
	// leave this screen or cancel an edit. Calling n.onBack rather than
	// e.g. n.showGamePicker directly keeps this button doing exactly what
	// the B button currently does, including its label/target changing
	// with context (browse vs. edit) -- the two can't drift apart.
	backButton := widget.NewButton(browseBackLabel, func() {
		if n.onBack != nil {
			n.onBack()
		}
	})

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

	backToList := func() {
		editEntry.Hide()
		saveButton.Hide()
		status.Hide()
		hint.SetText(browseHint)
		backButton.SetText(browseBackLabel)
		n.focusables = []fyne.Focusable{list, confirmButton}
		n.focusIndex = 0
		n.win.Canvas().Focus(list)
		n.onBack = func() { n.showGamePicker() }
	}

	saveButton.OnTapped = func() {
		if err := p.SetValue(editingField.Path, editEntry.Text); err != nil {
			log.Printf("gui: %v", err)
			status.SetText(err.Error())
			n.focusIndex = 0
			n.win.Canvas().Focus(editEntry)
			return
		}
		fields = p.Fields()
		list.Refresh()
		backToList()
	}

	confirmButton.OnTapped = func() {
		dir, err := profile.DefaultDir()
		if err != nil {
			status.SetText(fmt.Sprintf("Save failed: %v", err))
			status.Show()
			return
		}
		path, err := profile.Save(p, dir)
		if err != nil {
			status.SetText(fmt.Sprintf("Save failed: %v", err))
		} else {
			status.SetText(fmt.Sprintf("Saved to %s", path))
		}
		status.Show()
	}

	list.OnSelected = func(id widget.ListItemID) {
		f := fields[id]
		if err := p.EditError(f.Path); err != nil {
			status.SetText(err.Error())
			status.Show()
			return
		}
		editingField = f
		status.SetText(fmt.Sprintf("Editing: %s", f.Label))
		status.Show()
		editEntry.SetText(f.Value)
		editEntry.Show()
		saveButton.Show()
		hint.SetText(editHint)
		backButton.SetText(cancelLabel)
		n.focusables = []fyne.Focusable{editEntry, saveButton}
		n.focusIndex = 0
		n.win.Canvas().Focus(editEntry)
		// Back (both the gamepad B button and backButton, which calls
		// n.onBack too) cancels: discard whatever's in editEntry and
		// return to browsing without calling SetValue, rather than
		// treating an in-progress, unsaved edit as if it had been
		// confirmed.
		n.onBack = backToList
	}

	content := container.NewBorder(nil, container.NewVBox(status, editEntry, saveButton, confirmButton, backButton, hint), nil, nil, list)
	n.setScreen(content, []fyne.Focusable{list, confirmButton})
}

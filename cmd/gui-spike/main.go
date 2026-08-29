// Command gui-spike is the navigation spike for issue #33 (a sub-issue of
// #36): a minimal Fyne window -- a List standing in for the game library,
// plus an Entry and a Button standing in for the auth flow -- driven
// entirely by a gamepad, to prove out the input-translation approach
// decided in #7 and #32 before more UI work builds on top of it.
//
// A background goroutine polls SDL2's GameController API via
// internal/controllerinput and replays each translated Action as a
// synthetic key event against whatever widget currently has focus, using
// Fyne's own focus system (List/Entry/Button all already handle arrow
// keys and Space) instead of building gamepad-aware navigation from
// scratch. Since List/Entry consume Up/Down/Left/Right themselves rather
// than yielding focus to a sibling widget, the shoulder buttons are wired
// to Fyne's Canvas.FocusNext/FocusPrevious directly to move between the
// List, Entry, and Button. Mouse and keyboard are untouched: Fyne handles
// those natively.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/alteredtech/frameshare-collector/internal/controllerinput"
)

// fakeGames stands in for a real detected game library (see
// internal/library) for the purposes of this navigation spike.
var fakeGames = []string{
	"Team Fortress 2",
	"Terraria",
	"Stray",
	"Portal 2",
}

// inGamescopeSession reports whether the process is running inside a
// gamescope compositor session -- what Steam Deck's Gaming Mode (and
// other SteamOS/gamescope-session-based handhelds) actually runs under.
// gamescope-session sets these two XDG variables; checking them, rather
// than the SteamDeck=1 env var SteamOS also exports, distinguishes actual
// Gaming Mode from Desktop Mode on the same hardware, where a windowed
// dev/test session should stay windowed.
func inGamescopeSession() bool {
	return os.Getenv("XDG_CURRENT_DESKTOP") == "gamescope" || os.Getenv("XDG_SESSION_DESKTOP") == "gamescope"
}

// touchDebugLogPath returns where -debug-touch's log file is written, so it
// can be retrieved after a Gaming Mode run (whose stdout isn't easily read
// back) by switching to Desktop Mode and reading it directly.
func touchDebugLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, "gui-spike-touch-debug.log")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	fullscreen := flag.Bool("fullscreen", inGamescopeSession(), "launch full-screen instead of the default small window; defaults to true automatically under a gamescope session (Steam Deck Gaming Mode and similar), false elsewhere, so end users never need to set this themselves")
	debugTouchFlag := flag.Bool("debug-touch", false, "log raw hover/tap events with position data, for diagnosing issue #41 (touchscreen double-tap lag). Also mirrors log output to a file since Gaming Mode's stdout isn't easily read back")
	flag.Parse()
	touchDebug = *debugTouchFlag

	if touchDebug {
		logPath := touchDebugLogPath()
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
			log.Printf("touch-debug: could not open log file %s: %v (logging to stderr only)", logPath, err)
		} else {
			defer f.Close()
			log.SetOutput(io.MultiWriter(os.Stderr, f))
			log.Printf("touch-debug: also logging to %s", logPath)
		}
	}

	a := app.NewWithID("com.alteredtech.frameshare-collector.gui-spike")
	w := a.NewWindow("Controller Navigation Spike")

	gameList := widget.NewList(
		func() int { return len(fakeGames) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(fakeGames[id])
		},
	)

	selected := -1
	gameList.OnSelected = func(id widget.ListItemID) {
		selected = id
		logSelection(id, fakeGames[id])
	}

	// Plain Entry, not NewPasswordEntry: this field only stands in for
	// Entry-focus/typing testing, not real secret handling (that's out of
	// scope here). NewPasswordEntry's reveal icon is mouse-only and has no
	// gamepad path wired to it, and Fyne re-adds that same icon any time
	// Password is set to true even on a manually built Entry, so there's
	// no way to keep masking without it.
	authEntry := widget.NewEntry()
	authEntry.SetPlaceHolder("Auth token (Steam+X for on-screen keyboard)")
	hint := widget.NewLabel("D-pad/stick: navigate  |  A: activate  |  LB/RB: switch field")

	status := widget.NewLabel("Select a game and enter a token, then confirm.")
	confirm := newDebugTouchButton("Confirm", func() {
		if selected < 0 {
			status.SetText("No game selected.")
			return
		}
		status.SetText(fmt.Sprintf("Confirmed: %s", fakeGames[selected]))
	})

	w.SetContent(container.NewBorder(nil, container.NewVBox(authEntry, confirm, status, hint), nil, nil, gameList))
	if *fullscreen {
		w.SetFullScreen(true)
	} else {
		w.Resize(fyne.NewSize(480, 360))
	}

	// Cycle focus among these three widgets ourselves rather than relying
	// on Fyne's built-in Canvas.FocusNext/FocusPrevious: in testing, that
	// built-in Tab-order walk stopped returning to the List after the
	// Button was activated. An explicit, fixed-order ring sidesteps
	// whatever that internal behavior is -- reasonable for this spike's
	// small, unchanging set of widgets.
	focusables := []fyne.Focusable{gameList, authEntry, confirm}
	focusIndex := 0
	w.Canvas().Focus(focusables[focusIndex])
	cycleFocus := func(delta int) {
		old := focusIndex
		focusIndex = (focusIndex + delta + len(focusables)) % len(focusables)
		w.Canvas().Focus(focusables[focusIndex])
		// If canvas.Focused()'s type doesn't match focusables[focusIndex],
		// something outside cycleFocus (e.g. a real Tab/Shift+Tab keypress
		// driving Fyne's own internal FocusNext/FocusPrevious -- see
		// window.capturesTab in Fyne's glfw driver) is also moving focus,
		// desyncing our tracked index from Fyne's actual state.
		log.Printf("cycleFocus(%+d): index %d -> %d (expect focus type %T); canvas.Focused() actually reports %T",
			delta, old, focusIndex, focusables[focusIndex], w.Canvas().Focused())
	}

	poller, err := controllerinput.NewPoller(log.Printf)
	if err != nil {
		log.Fatalf("controller input unavailable: %v", err)
	}
	defer poller.Close()

	go func() {
		for action := range poller.Actions() {
			log.Printf("dispatching action: %s", action)
			fyne.Do(func() {
				dispatchAction(w.Canvas(), cycleFocus, action)
			})
		}
	}()

	w.ShowAndRun()
}

// dispatchAction handles FocusNext/FocusPrevious via cycleFocus (see the
// comment where it's built, above), and replays every other action as a
// synthetic key event against whatever widget currently has focus,
// relying on Fyne's existing widgets to know how to handle it (List moves
// its highlight on Up/Down and selects on Space; Button activates on
// Space).
func dispatchAction(canvas fyne.Canvas, cycleFocus func(delta int), action controllerinput.Action) {
	switch action {
	case controllerinput.ActionFocusNext:
		cycleFocus(1)
		return
	case controllerinput.ActionFocusPrevious:
		cycleFocus(-1)
		return
	}

	key := actionKey(action)
	if key == "" {
		return
	}
	if focused := canvas.Focused(); focused != nil {
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

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
	"fmt"
	"log"

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

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

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
	gameList.OnSelected = func(id widget.ListItemID) { selected = id }

	authEntry := widget.NewPasswordEntry()
	authEntry.SetPlaceHolder("Auth token (Steam+X for on-screen keyboard)")
	hint := widget.NewLabel("D-pad/stick: navigate  |  A: activate  |  LB/RB: switch field")

	status := widget.NewLabel("Select a game and enter a token, then confirm.")
	confirm := widget.NewButton("Confirm", func() {
		if selected < 0 {
			status.SetText("No game selected.")
			return
		}
		status.SetText(fmt.Sprintf("Confirmed: %s", fakeGames[selected]))
	})

	w.SetContent(container.NewBorder(nil, container.NewVBox(authEntry, confirm, status, hint), nil, nil, gameList))
	w.Resize(fyne.NewSize(480, 360))

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

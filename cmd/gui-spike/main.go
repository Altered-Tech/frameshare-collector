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
// scratch. Mouse and keyboard are untouched: Fyne handles those natively.
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

	status := widget.NewLabel("Select a game and enter a token, then confirm.")
	confirm := widget.NewButton("Confirm", func() {
		if selected < 0 {
			status.SetText("No game selected.")
			return
		}
		status.SetText(fmt.Sprintf("Confirmed: %s", fakeGames[selected]))
	})

	w.SetContent(container.NewBorder(nil, container.NewVBox(authEntry, confirm, status), nil, nil, gameList))
	w.Resize(fyne.NewSize(480, 360))
	w.Canvas().Focus(gameList)

	poller, err := controllerinput.NewPoller()
	if err != nil {
		log.Fatalf("controller input unavailable: %v", err)
	}
	defer poller.Close()

	go func() {
		for action := range poller.Actions() {
			fyne.Do(func() {
				dispatchAction(w.Canvas(), action)
			})
		}
	}()

	w.ShowAndRun()
}

// dispatchAction replays action as a synthetic key event against whatever
// widget currently has focus. It relies entirely on Fyne's existing
// widgets already knowing how to handle these keys (List moves its
// highlight on Up/Down and selects on Space; Button activates on Space) --
// see #33's scope note about relying on defaults vs. explicit
// FocusNext/FocusPrevious wiring.
func dispatchAction(canvas fyne.Canvas, action controllerinput.Action) {
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

package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// touchDebug gates all logging in this file behind the -debug-touch flag,
// so newDebugTouchButton/logSelection can be wired in unconditionally
// without an extra branch at every call site.
var touchDebug bool

// debugTouchButton wraps a *widget.Button to log every raw hover/tap event
// Fyne's GLFW driver delivers to it, with position data, for diagnosing
// issue #41 (touchscreen taps under gamescope needing a double-press, with
// state lagging one step behind). It's temporary instrumentation, not
// meant to stick around once #41 is root-caused -- Button only exposes
// Hoverable (MouseIn/MouseMoved/MouseOut) and Tapped, not raw MouseDown/Up,
// since Fyne's driver computes Tapped itself from a full press+release
// cycle; MouseMoved firing is not logged itself since it fires continuously
// while hovering; only the coordinate at MouseIn/Tapped matters here.
type debugTouchButton struct {
	widget.Button
}

func newDebugTouchButton(label string, tapped func()) *debugTouchButton {
	b := &debugTouchButton{}
	b.Text = label
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

func (b *debugTouchButton) MouseIn(ev *desktop.MouseEvent) {
	if touchDebug {
		log.Printf("[touch-debug] Confirm.MouseIn  pos=%v abs=%v", ev.Position, ev.AbsolutePosition)
	}
	b.Button.MouseIn(ev)
}

func (b *debugTouchButton) MouseMoved(ev *desktop.MouseEvent) {
	b.Button.MouseMoved(ev)
}

func (b *debugTouchButton) MouseOut() {
	if touchDebug {
		log.Printf("[touch-debug] Confirm.MouseOut")
	}
	b.Button.MouseOut()
}

func (b *debugTouchButton) Tapped(ev *fyne.PointEvent) {
	if touchDebug {
		log.Printf("[touch-debug] Confirm.Tapped  pos=%v abs=%v", ev.Position, ev.AbsolutePosition)
	}
	b.Button.Tapped(ev)
}

// logSelection logs when the game List's own Tapped->OnSelected chain
// actually fires, so its timing/ordering can be compared against
// Confirm's raw events above.
func logSelection(id widget.ListItemID, name string) {
	if touchDebug {
		log.Printf("[touch-debug] List.OnSelected  id=%d name=%s", id, name)
	}
}

// Package controllerinput translates gamepad input, read via SDL2's
// GameController API, into a small set of UI-navigation actions. Fyne (the
// GUI framework this app uses, see issue #7) has no native gamepad support
// but does have a built-in keyboard-driven focus system, so the intended
// use is: read raw controller state here, then have the UI layer replay
// each Action as a synthetic key event against whatever widget currently
// has focus.
package controllerinput

import (
	"fmt"
	"runtime"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// Action is a UI-navigation intent derived from raw gamepad input. It says
// nothing about which key event a caller should synthesize for it -- that
// mapping belongs to whatever UI toolkit is consuming the action.
type Action int

const (
	ActionUp Action = iota
	ActionDown
	ActionLeft
	ActionRight
	ActionActivate
	// ActionFocusNext and ActionFocusPrevious move focus between sibling
	// widgets (e.g. from the game List to the auth Entry to the Confirm
	// Button). They exist separately from the directional actions because
	// widgets like List and Entry consume Up/Down/Left/Right themselves
	// (moving a highlight or a text cursor) rather than yielding focus --
	// Fyne's own Tab-based focus-cycling lives one level above a widget's
	// TypedKey handling, so a caller has to invoke it explicitly (see
	// fyne.Canvas.FocusNext/FocusPrevious) rather than by replaying a key.
	ActionFocusNext
	ActionFocusPrevious
	// ActionBack signals "leave the current screen/mode without
	// committing anything" (return to a previous screen, cancel an
	// in-progress edit). It's kept separate from Activate rather than
	// overloading it, and separate from the directional/focus actions,
	// since it's a UI-level navigation command a caller must handle
	// itself -- there's no Fyne key it can be generically replayed as
	// the way Up/Down/Left/Right/Activate are.
	ActionBack
)

func (a Action) String() string {
	switch a {
	case ActionUp:
		return "up"
	case ActionDown:
		return "down"
	case ActionLeft:
		return "left"
	case ActionRight:
		return "right"
	case ActionActivate:
		return "activate"
	case ActionFocusNext:
		return "focus-next"
	case ActionFocusPrevious:
		return "focus-previous"
	case ActionBack:
		return "back"
	default:
		return fmt.Sprintf("unknown action %d", int(a))
	}
}

// axisThreshold is how far an axis (SDL reports -32768..32767) must move
// from center before it counts as a directional press. High enough to
// ignore stick drift and noise near center on worn or cheap pads.
const axisThreshold = 16000

// debounceWindow suppresses a repeat press of the same button (or the
// same axis crossing back over its threshold) if it follows the previous
// one too quickly. This isn't about our poll rate or event handling --
// PollEvent drains genuinely distinct SDL events -- it's compensating for
// switch bounce at the hardware/driver level: confirmed via raw event
// logging during hands-on testing, a single physical D-pad tap on one
// controller produced two or three complete CONTROLLERBUTTONDOWN/UP
// pairs from SDL. Tune if it ever clips a deliberately fast double-tap.
const debounceWindow = 150 * time.Millisecond

// ButtonAction maps an SDL GameController button to the Action it
// represents, if any. D-pad directions map to their matching directional
// Action; the south face button (A on an Xbox-layout pad, Cross on
// PlayStation, and what Steam Input reports for Steam Deck's south face
// button) maps to Activate; the east face button (B on an Xbox-layout
// pad, Circle on PlayStation, Steam Deck's east face button) maps to
// Back, matching the back/cancel convention nearly every game and game
// console UI already uses that button for; the shoulder buttons cycle
// focus between widgets, since D-pad/stick directions alone can't
// escape a widget that consumes them internally (see ActionFocusNext).
// Every other button is unmapped for now.
func ButtonAction(button sdl.GameControllerButton) (Action, bool) {
	switch button {
	case sdl.CONTROLLER_BUTTON_DPAD_UP:
		return ActionUp, true
	case sdl.CONTROLLER_BUTTON_DPAD_DOWN:
		return ActionDown, true
	case sdl.CONTROLLER_BUTTON_DPAD_LEFT:
		return ActionLeft, true
	case sdl.CONTROLLER_BUTTON_DPAD_RIGHT:
		return ActionRight, true
	case sdl.CONTROLLER_BUTTON_A:
		return ActionActivate, true
	case sdl.CONTROLLER_BUTTON_B:
		return ActionBack, true
	case sdl.CONTROLLER_BUTTON_RIGHTSHOULDER:
		return ActionFocusNext, true
	case sdl.CONTROLLER_BUTTON_LEFTSHOULDER:
		return ActionFocusPrevious, true
	default:
		return 0, false
	}
}

// AxisAction reports the directional Action a left-stick axis reading
// represents right now, if its value is past axisThreshold. It only
// answers "what does this raw value mean", not "is this a new press" --
// Poller handles edge-triggering so a held stick doesn't repeat-fire on
// every poll.
func AxisAction(axis sdl.GameControllerAxis, value int16) (Action, bool) {
	switch axis {
	case sdl.CONTROLLER_AXIS_LEFTX:
		switch {
		case value <= -axisThreshold:
			return ActionLeft, true
		case value >= axisThreshold:
			return ActionRight, true
		}
	case sdl.CONTROLLER_AXIS_LEFTY:
		switch {
		case value <= -axisThreshold:
			return ActionUp, true
		case value >= axisThreshold:
			return ActionDown, true
		}
	}
	return 0, false
}

// axisKey identifies one axis on one physical controller, so Poller can
// track each axis's press/release edge independently when multiple
// controllers are connected.
type axisKey struct {
	controller sdl.JoystickID
	axis       sdl.GameControllerAxis
}

// Poller initializes SDL's game controller subsystem and polls it in a
// background goroutine, reporting translated Actions on a channel. It
// deliberately initializes no other SDL subsystem (no video, no audio) so
// it can run alongside a separate UI toolkit's own window/event loop
// without contending for window-system ownership.
type Poller struct {
	actions chan Action
	quit    chan struct{}
	done    chan struct{}
	logf    func(format string, args ...any)
}

// NewPoller initializes SDL's controller subsystem and starts polling for
// input. logf receives one line per controller connect/disconnect (pass
// nil to discard them) -- this is the tool for diagnosing setups where a
// single physical pad shows up as more than one SDL device (e.g. Steam
// Input on Steam Deck can expose both a raw HID interface and a virtual
// emulated controller for the same hardware), which would otherwise
// silently double-fire every Action without any other visible symptom.
// Call Close when done to stop the goroutine and release SDL resources.
func NewPoller(logf func(format string, args ...any)) (*Poller, error) {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if err := sdl.InitSubSystem(sdl.INIT_GAMECONTROLLER); err != nil {
		return nil, fmt.Errorf("init SDL game controller subsystem: %w", err)
	}

	p := &Poller{
		actions: make(chan Action, 16),
		quit:    make(chan struct{}),
		done:    make(chan struct{}),
		logf:    logf,
	}
	go p.run()
	return p, nil
}

// Actions returns the channel translated Actions are delivered on. The
// send side never blocks: a consumer that falls behind drops older input
// rather than stalling the poll loop, since a directional press that's a
// few frames stale is worthless and the game controller state it came
// from has already moved on.
func (p *Poller) Actions() <-chan Action {
	return p.actions
}

// Close stops the polling loop and releases SDL's controller subsystem.
// It blocks until the polling goroutine has exited, then closes the
// Actions channel -- safe only once run's goroutine (its sole sender) has
// returned, which the wait on p.done guarantees.
func (p *Poller) Close() {
	close(p.quit)
	<-p.done
	close(p.actions)
	sdl.QuitSubSystem(sdl.INIT_GAMECONTROLLER)
}

func (p *Poller) run() {
	// SDL's own docs only document a hard main-thread requirement for the
	// video subsystem; the controller subsystem has none. Locking this
	// goroutine to one OS thread regardless keeps every SDL call here on
	// a single, consistent thread, which SDL does still expect.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(p.done)

	controllers := map[sdl.JoystickID]*sdl.GameController{}
	defer func() {
		for _, c := range controllers {
			c.Close()
		}
	}()

	axisActive := map[axisKey]bool{}
	type buttonKey struct {
		controller sdl.JoystickID
		button     uint8
	}
	lastButtonPress := map[buttonKey]time.Time{}
	lastAxisEmit := map[axisKey]time.Time{}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.quit:
			return
		case <-ticker.C:
		}

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.ControllerDeviceEvent:
				p.handleDeviceEvent(e, controllers)
			case *sdl.ControllerButtonEvent:
				p.logf("raw button event: controller=%d button=%d state=%s", e.Which, e.Button, buttonStateName(e.State))
				if e.State != sdl.PRESSED {
					continue
				}
				action, ok := ButtonAction(sdl.GameControllerButton(e.Button))
				if !ok {
					continue
				}
				bKey := buttonKey{controller: e.Which, button: e.Button}
				if debounced(lastButtonPress, bKey, debounceWindow) {
					p.logf("-> debounced (button %d pressed again within %v)", e.Button, debounceWindow)
					continue
				}
				p.logf("-> emitting %s", action)
				p.emit(action)
			case *sdl.ControllerAxisEvent:
				key := axisKey{controller: e.Which, axis: sdl.GameControllerAxis(e.Axis)}
				action, active := AxisAction(sdl.GameControllerAxis(e.Axis), e.Value)
				if active != axisActive[key] {
					p.logf("raw axis edge: controller=%d axis=%d value=%d active=%v", e.Which, e.Axis, e.Value, active)
				}
				if active && !axisActive[key] {
					if debounced(lastAxisEmit, key, debounceWindow) {
						p.logf("-> debounced (axis %d re-crossed threshold within %v)", e.Axis, debounceWindow)
					} else {
						p.logf("-> emitting %s", action)
						p.emit(action)
					}
				}
				axisActive[key] = active
			}
		}
	}
}

// debounced reports whether an event for key follows a previously
// accepted one too closely to be a new, distinct press (see
// debounceWindow), recording key as pressed now if not.
func debounced[K comparable](last map[K]time.Time, key K, window time.Duration) bool {
	now := time.Now()
	if t, ok := last[key]; ok && now.Sub(t) < window {
		return true
	}
	last[key] = now
	return false
}

// handleDeviceEvent opens newly connected controllers and closes ones
// that were unplugged, so hot-plugging a controller mid-session doesn't
// require restarting the app.
func (p *Poller) handleDeviceEvent(e *sdl.ControllerDeviceEvent, controllers map[sdl.JoystickID]*sdl.GameController) {
	switch e.Type {
	case sdl.CONTROLLERDEVICEADDED:
		// For this event only, Which is the joystick *device index*, not
		// an instance ID -- see the field doc on sdl.ControllerDeviceEvent.
		c := sdl.GameControllerOpen(int(e.Which))
		if c == nil {
			return
		}
		id := c.Joystick().InstanceID()
		controllers[id] = c
		p.logf("controller connected: %q (instance %d, GUID %s) -- %d total connected", c.Name(), id, sdl.JoystickGetGUIDString(c.Joystick().GUID()), len(controllers))
	case sdl.CONTROLLERDEVICEREMOVED:
		if c, ok := controllers[e.Which]; ok {
			p.logf("controller disconnected: %q (instance %d)", c.Name(), e.Which)
			c.Close()
			delete(controllers, e.Which)
		}
	}
}

func buttonStateName(state uint8) string {
	if state == sdl.PRESSED {
		return "pressed"
	}
	return "released"
}

func (p *Poller) emit(a Action) {
	select {
	case p.actions <- a:
	default:
	}
}

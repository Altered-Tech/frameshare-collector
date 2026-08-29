package controllerinput

import (
	"testing"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

func TestButtonAction(t *testing.T) {
	tests := []struct {
		button sdl.GameControllerButton
		want   Action
		wantOK bool
	}{
		{sdl.CONTROLLER_BUTTON_DPAD_UP, ActionUp, true},
		{sdl.CONTROLLER_BUTTON_DPAD_DOWN, ActionDown, true},
		{sdl.CONTROLLER_BUTTON_DPAD_LEFT, ActionLeft, true},
		{sdl.CONTROLLER_BUTTON_DPAD_RIGHT, ActionRight, true},
		{sdl.CONTROLLER_BUTTON_A, ActionActivate, true},
		{sdl.CONTROLLER_BUTTON_B, ActionBack, true},
		{sdl.CONTROLLER_BUTTON_RIGHTSHOULDER, ActionFocusNext, true},
		{sdl.CONTROLLER_BUTTON_LEFTSHOULDER, ActionFocusPrevious, true},
		{sdl.CONTROLLER_BUTTON_X, 0, false},
		{sdl.CONTROLLER_BUTTON_START, 0, false},
	}
	for _, tc := range tests {
		got, ok := ButtonAction(tc.button)
		if ok != tc.wantOK || (ok && got != tc.want) {
			t.Errorf("ButtonAction(%v) = (%v, %v), want (%v, %v)", tc.button, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestAxisAction(t *testing.T) {
	tests := []struct {
		name   string
		axis   sdl.GameControllerAxis
		value  int16
		want   Action
		wantOK bool
	}{
		{"leftX past threshold negative is left", sdl.CONTROLLER_AXIS_LEFTX, -20000, ActionLeft, true},
		{"leftX past threshold positive is right", sdl.CONTROLLER_AXIS_LEFTX, 20000, ActionRight, true},
		{"leftY past threshold negative is up", sdl.CONTROLLER_AXIS_LEFTY, -20000, ActionUp, true},
		{"leftY past threshold positive is down", sdl.CONTROLLER_AXIS_LEFTY, 20000, ActionDown, true},
		{"leftX within dead zone is inactive", sdl.CONTROLLER_AXIS_LEFTX, 500, 0, false},
		{"leftX exactly at threshold boundary counts", sdl.CONTROLLER_AXIS_LEFTX, axisThreshold, ActionRight, true},
		{"leftX just under threshold boundary does not count", sdl.CONTROLLER_AXIS_LEFTX, axisThreshold - 1, 0, false},
		{"right stick axis is unmapped", sdl.CONTROLLER_AXIS_RIGHTX, 20000, 0, false},
		{"trigger axis is unmapped", sdl.CONTROLLER_AXIS_TRIGGERLEFT, 20000, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := AxisAction(tc.axis, tc.value)
			if ok != tc.wantOK || (ok && got != tc.want) {
				t.Errorf("AxisAction(%v, %d) = (%v, %v), want (%v, %v)", tc.axis, tc.value, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestActionString(t *testing.T) {
	tests := []struct {
		action Action
		want   string
	}{
		{ActionUp, "up"},
		{ActionDown, "down"},
		{ActionLeft, "left"},
		{ActionRight, "right"},
		{ActionActivate, "activate"},
		{ActionFocusNext, "focus-next"},
		{ActionFocusPrevious, "focus-previous"},
		{ActionBack, "back"},
		{Action(99), "unknown action 99"},
	}
	for _, tc := range tests {
		if got := tc.action.String(); got != tc.want {
			t.Errorf("Action(%d).String() = %q, want %q", tc.action, got, tc.want)
		}
	}
}

func TestDebounced(t *testing.T) {
	last := map[string]time.Time{}
	window := 40 * time.Millisecond

	if debounced(last, "a", window) {
		t.Error("first call for a key should not be debounced")
	}
	if !debounced(last, "a", window) {
		t.Error("an immediate repeat for the same key should be debounced")
	}
	if debounced(last, "b", window) {
		t.Error("a different key should not be debounced by another key's history")
	}

	time.Sleep(window + 20*time.Millisecond)
	if debounced(last, "a", window) {
		t.Error("a call after window has elapsed should not be debounced")
	}
}

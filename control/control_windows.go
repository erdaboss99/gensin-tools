// Package control talks to the running game on Windows: finding its window,
// capturing frames and sending input.
package control

import (
	"errors"
	"image"
)

// Errors returned while attaching to or controlling the game.
var (
	ErrNotRunning  = errors.New("Genshin Impact is not running, or its window could not be found")
	ErrNotElevated = errors.New("the game runs as Administrator, so it ignores input from unelevated programs: run this from an Administrator terminal")
	ErrFocusLost   = errors.New("the game window lost focus, so scanning stopped")
	ErrBlackFrame  = errors.New("the captured frame is entirely black: switch the game to windowed or borderless mode")
)

// Game is a live game window. Coordinates are relative to its client area.
//
// Every input method first checks that the game has focus and otherwise returns
// ErrFocusLost, so input can never land in another application. Alt-Tab is
// therefore the emergency stop.
type Game interface {
	// Size returns the client area's size in physical pixels.
	Size() (width, height int)
	// Capture returns the current contents of the client area.
	Capture() (*image.RGBA, error)
	// MoveTo places the cursor without clicking.
	MoveTo(x, y int) error
	// Click moves the cursor and presses the left mouse button.
	Click(x, y int) error
	// Scroll turns the mouse wheel. Negative ticks scroll down.
	Scroll(ticks int) error
	// Type enters text into the focused control.
	Type(text string) error
	// PressEnter presses the Enter key.
	PressEnter() error
}

// Status is the environment report shown by the doctor command.
type Status struct {
	Elevated    bool
	GameRunning bool
	// Minimized means the window has no size to report until it is restored.
	Minimized bool
	Width     int
	Height    int
}

// AttachOptions configures Attach.
type AttachOptions struct {
	// NeedInput requires the privileges for sending input. Screenshot-only
	// commands leave it false.
	NeedInput bool
}

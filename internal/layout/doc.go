// Package layout describes where things are on the game's screens.
//
// Coordinates are expressed for a window 1280 pixels wide and scaled to the real
// window. A negative coordinate is measured from the right or bottom edge, so the
// same layout works for 16:9, 16:10 and 4:3 windows, where the extra height
// appears in the middle of the screen.
//
// Game patches move things, so the whole layout can be overridden from a JSON
// file. The calibrate command draws these boxes on a screenshot to check them.
package layout

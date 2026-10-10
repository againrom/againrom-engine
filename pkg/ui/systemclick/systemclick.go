// Package systemclick reads the system's double-click setting: the time
// within which a second press, and the rectangle centred on the first press
// within which its point, make a double click. It imports no window library,
// so each platform's reading builds and vets on its own. Windows reads the
// user's double-click time and rectangle, macOS its double-click interval;
// another platform, and any value a platform does not give, keeps Fallback.
package systemclick

import "time"

// Setting is a double-click time and the rectangle's size in pixels.
type Setting struct {
	Time          time.Duration
	Width, Height int
}

// Fallback is the Windows default: 500 ms and a 4 by 4 pixel rectangle.
var Fallback = Setting{Time: 500 * time.Millisecond, Width: 4, Height: 4}

// Read is the system's setting now.
func Read() Setting { return read() }

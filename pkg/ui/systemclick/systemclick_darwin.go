//go:build darwin

package systemclick

import (
	"time"

	"github.com/ebitengine/purego/objc"
)

// read is NSEvent's doubleClickInterval, sent through the Objective-C runtime
// the game window already loads, without cgo. macOS states no double-click
// distance, so Fallback's rectangle holds. Before AppKit is loaded the class
// is absent and Fallback holds.
func read() Setting {
	s := Fallback
	class := objc.GetClass("NSEvent")
	if class == 0 {
		return s
	}
	if sec := objc.Send[float64](objc.ID(class), objc.RegisterName("doubleClickInterval")); sec > 0 && sec < 60 {
		s.Time = time.Duration(sec * float64(time.Second))
	}
	return s
}

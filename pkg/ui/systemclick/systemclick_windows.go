//go:build windows

package systemclick

import (
	"syscall"
	"time"
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	getDoubleClickTime = user32.NewProc("GetDoubleClickTime")
	getSystemMetrics   = user32.NewProc("GetSystemMetrics")
)

// The GetSystemMetrics indices of the double-click rectangle's width and
// height.
const (
	smCXDoubleClk = 36
	smCYDoubleClk = 37
)

// read is the user's double-click time and rectangle.
func read() Setting {
	s := Fallback
	if getDoubleClickTime.Find() != nil || getSystemMetrics.Find() != nil {
		return s
	}
	if ms, _, _ := getDoubleClickTime.Call(); uint32(ms) > 0 {
		s.Time = time.Duration(uint32(ms)) * time.Millisecond
	}
	if w, _, _ := getSystemMetrics.Call(smCXDoubleClk); int32(w) > 0 {
		s.Width = int(int32(w))
	}
	if h, _, _ := getSystemMetrics.Call(smCYDoubleClk); int32(h) > 0 {
		s.Height = int(int32(h))
	}
	return s
}

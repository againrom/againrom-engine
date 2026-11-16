package ui

import (
	"image"

	"againrom/pkg/ui/cursorclip"
)

type cursorConfinement struct {
	enabled, clipped bool
	native           cursorclip.System
	read             func() (image.Rectangle, bool)
	clip             func(*image.Rectangle) bool
}

// SetCursorInWindow keeps the pointer inside the focused game window.
func (a *App) SetCursorInWindow(on bool) { a.pointerClip.enabled = on }

// CursorInWindow reports the launch choice, independent of window focus.
func (a *App) CursorInWindow() bool { return a.pointerClip.enabled }

func (c *cursorConfinement) apply() {
	if !c.enabled {
		c.release()
		return
	}
	if c.read == nil {
		c.read = c.native.Rectangle
	}
	if c.clip == nil {
		c.clip = c.native.Clip
	}
	r, focused := c.read()
	if !focused || r.Empty() {
		c.release()
		return
	}
	if c.clip(&r) {
		c.clipped = true
	} else {
		c.release()
	}
}

func (c *cursorConfinement) release() {
	if c.clipped && c.clip(nil) {
		c.clipped = false
	}
}

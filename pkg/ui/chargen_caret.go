package ui

import "time"

// nameCaret is the pre-create name field's caret, a `|` appended to the drawn
// name while it shows (TEXT-076). The field reads no focus state.
//
// The original draws a frame first and only then flips the caret when more
// than the caret phase has passed, stamping that frame's time. The field is built
// once, with the screen, and not on each opening; here its stamp is taken to
// be older than the phase when the page first shows (DIV-1496), so the first
// frame shows the caret and the check after it flips. A character under the
// cap shows the caret and stamps its own time. Here one App tick composes one
// frame, and paint runs at the tick's end.
type nameCaret struct {
	hidden  bool      // the zero value shows the caret, as the field's construction does
	stamp   time.Time // the last flip or restart; zero is before the page first showed
	frame   time.Time // the last composed pre-create frame; zero is none since the last paint
	restart bool      // a character under the cap arrived during this tick
}

// visible reports whether the frame now being composed shows the caret.
func (k *nameCaret) visible() bool {
	return !k.hidden || k.restart
}

// paint settles the flip that follows the previous pre-create frame, applies a
// restart from this tick's characters at now, and records whether this tick's
// frame is a pre-create frame.
// A flip needs more than phase since the last flip or restart.
func (k *nameCaret) paint(now time.Time, preCreate bool, phase time.Duration) {
	if !k.frame.IsZero() && (k.stamp.IsZero() || k.frame.Sub(k.stamp) > phase) {
		k.hidden = !k.hidden
		k.stamp = k.frame
	}
	if k.restart {
		k.hidden, k.stamp, k.restart = false, now, false
	}
	k.frame = time.Time{}
	if preCreate {
		k.frame = now
	}
}

// CaretVisible reports whether the pre-create name field's caret shows in the
// frame composed now. It is false on the detailed page, which has no field.
func (c *Chargen) CaretVisible() bool {
	return c != nil && c.stage == PreCreateStage && c.setup.PreCreate != nil && c.caret.visible()
}

// paintChargenCaret runs at the end of every App tick for the generator's
// caret. A cutscene over the screen composes no pre-create frame.
func (a *App) paintChargenCaret(now time.Time) {
	c := a.flow.chargen
	if c == nil {
		return
	}
	phase := time.Duration(0)
	if l := c.layout(); l != nil {
		phase = ms(l.PreCreate.Name.CaretMS)
	}
	c.caret.paint(now, a.flow.screen == ScreenChargen && a.cutscene == nil && c.stage == PreCreateStage, phase)
}

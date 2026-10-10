// Package latch is the one press latch every push button uses (MENU-116,
// DIALOGUE-045). A press on an enabled button latches it; the release
// activates it only when it lands inside the latched button. A release
// elsewhere, a lost focus or a closed screen clears the latch and
// activates nothing.
package latch

// Latch holds the button a press began on. The zero value holds none.
type Latch struct {
	held bool
	id   int
}

// Press latches id when ok. While a button is held a further press changes
// nothing: the original sets the latch only when none is set.
func (l *Latch) Press(id int, ok bool) {
	if l.held || !ok {
		return
	}
	l.held, l.id = true, id
}

// Pressed reports whether id holds the latch.
func (l Latch) Pressed(id int) bool { return l.held && l.id == id }

// Holds reports whether any button holds the latch.
func (l Latch) Holds() bool { return l.held }

// Latched reports the button holding the latch.
func (l Latch) Latched() (int, bool) { return l.id, l.held }

// Release clears the latch and reports the latched button and whether the
// release activates it: it does when the release lands inside that button.
func (l *Latch) Release(id int, inside bool) (int, bool) {
	held, latched := l.held, l.id
	l.held, l.id = false, 0
	return latched, held && inside && latched == id
}

// Clear drops the latch without activating anything.
func (l *Latch) Clear() { l.held, l.id = false, 0 }

package latch

import "testing"

func TestLatchActivatesOnlyOnAReleaseInsideTheLatchedButton(t *testing.T) {
	var l Latch
	l.Press(3, true)
	if !l.Pressed(3) || l.Pressed(4) {
		t.Fatal("press did not latch button 3 alone")
	}
	if at, ok := l.Release(3, true); !ok || at != 3 {
		t.Fatal("release inside did not activate", at, ok)
	}
	if l.Holds() {
		t.Fatal("release kept the latch")
	}

	l.Press(3, true)
	if _, ok := l.Release(3, false); ok {
		t.Fatal("release outside activated")
	}
	l.Press(3, true)
	if _, ok := l.Release(4, true); ok {
		t.Fatal("release inside another button activated")
	}

	l.Press(3, true)
	l.Clear()
	if _, ok := l.Release(3, true); ok {
		t.Fatal("release after a cleared latch activated")
	}

	l.Press(3, false)
	if l.Holds() {
		t.Fatal("a press on a disabled button or on no button latched")
	}

	l.Press(3, true)
	l.Press(4, true)
	if !l.Pressed(3) {
		t.Fatal("a second press replaced the held latch")
	}
	if at, ok := l.Release(3, true); !ok || at != 3 {
		t.Fatal("double press lost the first button", at, ok)
	}
	if _, ok := l.Release(3, true); ok {
		t.Fatal("one press activated twice")
	}
}

package ui

// The attack mode's SECOND writer — the held modifier — and the two events that
// lower the mode whichever writer raised it.
//
// Everything here is driven through App.step, for 0075's own reason: what is
// being fixed is which surface raises the mode and what takes it away, and both
// are properties of that dispatch. A test that set the fields directly would
// witness a struct and not a front-end.

import (
	"reflect"
	"testing"
)

// afHeld is a frame carrying the modifier at a window position, and afUp the
// same frame without it. The pair is what makes a LEVEL observable: the field is
// assigned on every frame, so "the modifier is up" has to be a frame that says
// so rather than the absence of one.
func afHeld(x, y int) appInput {
	in := atFrame(x, y)
	in.AttackHeld = true
	return in
}

// afEmpty is the window position of ground holding no unit, where the modifier
// cases are made so that nothing but the modifier is in the frame.
func afEmpty(v *Viewer) (int, int) { return cellPoint(v, atEmptyCol, atEmptyRow) }

// TestTheModifierIsALevelAndNotAToggle is AC-1.
//
// THE THIRD FRAME IS THE ONE THAT MATTERS. A toggle would have been flipped
// twice by then and would read DOWN; a level reads whatever the frame carries.
func TestTheModifierIsALevelAndNotAToggle(t *testing.T) {
	a, v, _ := atOnMap(t)
	x, y := afEmpty(v)

	a.step(afHeld(x, y), atAt)
	if !v.AttackArmed() {
		t.Fatal("the modifier down did not raise the mode")
	}

	a.step(atFrame(x, y), atAt)
	if v.AttackArmed() {
		t.Fatal("the modifier up did not lower the mode — it is behaving as a latch, not a level")
	}

	a.step(afHeld(x, y), atAt)
	if !v.AttackArmed() {
		t.Error("the modifier down a second time did not raise the mode — it is behaving as a toggle")
	}
}

// TestTheModifierSurfaceIsUngated is AC-2.
//
// The two selections are exactly the two the KEY refuses, asked of the modifier.
// The asymmetry is the decode's: the ownership gate belongs to the command
// panel's arming path, which the key stands in for, and the modifier path reads
// no player value at the hover or at the click.
func TestTheModifierSurfaceIsUngated(t *testing.T) {
	t.Run("an empty selection", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		x, y := afEmpty(v)

		atArm(a, v)
		if v.AttackArmed() {
			t.Fatal("premise: the key armed over an empty selection, so this case tests nothing")
		}
		a.step(afHeld(x, y), atAt)
		if !v.AttackArmed() {
			t.Error("the modifier refused an empty selection")
		}
	})

	t.Run("a selection the local participant does not own", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		x, y := afEmpty(v)
		ents := atEntities()
		for i := range ents {
			ents[i].Owner = atOwner + 1
		}
		v.SetEntities(ents)
		v.SetLocalOwner(atOwner)
		v.sel = selection{atLoID}

		atArm(a, v)
		if v.AttackArmed() {
			t.Fatal("premise: the key armed over a foreign selection, so this case tests nothing")
		}
		a.step(afHeld(x, y), atAt)
		if !v.AttackArmed() {
			t.Error("the modifier refused a selection the participant does not own")
		}
	})
}

// TestTheModifierIsNotSpentByThePressAndTheKeyStillIs is AC-3 and AC-10 — the
// one behaviour a merged field could not have.
func TestTheModifierIsNotSpentByThePressAndTheKeyStillIs(t *testing.T) {
	t.Run("a press under the held modifier attacks, and a second press attacks again", func(t *testing.T) {
		a, v, s := atOnMap(t)
		v.sel = selection{atLoID, atHiID}
		x, y := cellPoint(v, atFoeCol, atFoeRow)

		afTap := func() {
			down := afHeld(x, y)
			down.PrimaryPressed = true
			down.Viewer.PrimaryDown = true
			a.step(down, atAt)
			up := afHeld(x, y)
			up.PrimaryReleased = true
			a.step(up, atAt)
		}
		afTap()
		// The tap SPENDS the mode, exactly as the key's own press does. What
		// the modifier has that the key does not is that it is a LEVEL: the
		// very next tick with it still down raises the mode again, with no
		// second key press, which is what the second tap below then finds.
		a.step(afHeld(x, y), atAt)
		if !v.AttackArmed() {
			t.Fatal("the modifier did not raise the mode again on the tick after the tap that spent it")
		}
		afTap()

		want := []aimed{{atLoID, atFoeID, 0}, {atHiID, atFoeID, 0}, {atLoID, atFoeID, 0}, {atHiID, atFoeID, 0}}
		if !reflect.DeepEqual(s.attacks, want) {
			t.Errorf("two presses under one held modifier gave %v, want %v", s.attacks, want)
		}
		if len(s.orders) != 0 {
			t.Errorf("the move seam received %v, want nothing", s.orders)
		}
	})

	t.Run("the key is still spent by its press", func(t *testing.T) {
		a, v, s := atOnMap(t)
		v.sel = selection{atLoID}
		atArm(a, v)
		atPress(a, v, atFoeCol, atFoeRow)
		if v.AttackArmed() {
			t.Fatal("the key's mode survived the press that consumed it")
		}
		atPress(a, v, atFoeCol, atFoeRow)

		want := []aimed{{atLoID, atFoeID, 0}}
		if !reflect.DeepEqual(s.attacks, want) {
			t.Errorf("attacks = %v, want %v — the second press must be a move", s.attacks, want)
		}
	})
}

// TestFocusLossLowersTheModeWhicheverSurfaceRaisedIt is AC-4.
func TestFocusLossLowersTheModeWhicheverSurfaceRaisedIt(t *testing.T) {
	t.Run("the key's mode", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		v.sel = selection{atLoID}
		atArm(a, v)
		if !v.AttackArmed() {
			t.Fatal("premise: the key did not arm")
		}

		x, y := afEmpty(v)
		in := atFrame(x, y)
		in.Unfocused = true
		a.step(in, atAt)
		if v.AttackArmed() {
			t.Error("an unfocused tick left the key's mode up")
		}
	})

	t.Run("the modifier's mode, and it does not come back on its own", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		x, y := afEmpty(v)

		a.step(afHeld(x, y), atAt)
		if !v.AttackArmed() {
			t.Fatal("premise: the modifier did not arm")
		}

		// The unfocused frame still carries the modifier: the window has lost
		// focus, so what the keyboard is doing is exactly what cannot be trusted.
		lost := afHeld(x, y)
		lost.Unfocused = true
		a.step(lost, atAt)
		if v.AttackArmed() {
			t.Fatal("an unfocused tick left the modifier's mode up")
		}

		// Focus back, modifier released. Nothing restores the mode.
		a.step(atFrame(x, y), atAt)
		if v.AttackArmed() {
			t.Error("the mode came back when focus did")
		}
	})
}

// TestAPopupTakesTheModeAndDoesNotGiveItBack is AC-9's state half.
//
// This is the one the story was briefed to prove: a gate that swallows every
// input has a hole exactly where a later story adds an input below it without
// looking. The modifier is read below that gate, and the mode is lowered above
// it, so both directions are closed.
func TestAPopupTakesTheModeAndDoesNotGiveItBack(t *testing.T) {
	t.Run("the modifier raises nothing under an open popup", func(t *testing.T) {
		f := newPopupFix(t, haltOpts{})
		f.open()

		in := f.at(popupACol, popupARow)
		in.AttackHeld = true
		f.frame(in)
		if f.v.AttackArmed() {
			t.Error("the modifier armed under an open popup")
		}
	})

	t.Run("a mode already up is lowered by the popup and not restored by its dismissal", func(t *testing.T) {
		f := newPopupFix(t, haltOpts{})

		in := f.at(popupACol, popupARow)
		in.AttackHeld = true
		f.frame(in)
		if !f.v.AttackArmed() {
			t.Fatal("premise: the modifier did not arm on an ordinary frame")
		}

		f.open()
		f.frame(in) // the modifier is STILL held across the popup
		if f.v.AttackArmed() {
			t.Fatal("the mode survived a popup opening over it")
		}

		f.v.ClearNotice()
		if f.a.flow.popupOpen() {
			t.Fatal("setup: the notice did not close")
		}
		// The very next frame carries no modifier, so nothing may raise it.
		f.frame(f.at(popupACol, popupARow))
		if f.v.AttackArmed() {
			t.Error("the mode came back when the popup was dismissed")
		}
	})
}

package ui

// The readout's attack row: what it states, and that the picture behind it
// actually changes when the mode does.

import (
	"testing"
)

// TestTheReadoutStatesBothPositionsOfTheAttackMode is AC-12's first half.
//
// BOTH positions are asserted, and the row must be PRESENT in both: a row that
// vanished while the mode was down would make "the key did nothing" and "there
// is no such row" one picture, which is the confusion the row exists to remove.
func TestTheReadoutStatesBothPositionsOfTheAttackMode(t *testing.T) {
	for _, tc := range []struct {
		armed bool
		want  string
	}{
		{false, readoutDisarmed},
		{true, readoutArmed},
	} {
		got, ok := readoutText(readoutSubject{Armed: tc.armed}, PanelFieldAttack)
		if !ok {
			t.Errorf("armed=%v: the row has no value; both positions are values", tc.armed)
			continue
		}
		if got != tc.want {
			t.Errorf("armed=%v: the row reads %q, want %q", tc.armed, got, tc.want)
		}
	}
	if readoutArmed == readoutDisarmed {
		t.Error("the two positions read the same, so the row states nothing")
	}
}

// TestTheAuthoredLayoutCarriesTheAttackRow is AC-12's other half: the field is
// wired into the box this project actually ships, not merely defined.
func TestTheAuthoredLayoutCarriesTheAttackRow(t *testing.T) {
	rows := AuthoredReadoutLayout().Rows
	for _, r := range rows {
		if r.Field == PanelFieldAttack {
			if r.Label == "" {
				t.Error("the attack row ships with no label")
			}
			return
		}
	}
	t.Errorf("the shipped layout's %d row(s) carry no attack field", len(rows))
}

// TestTheSubjectFollowsTheArmedFlag is what makes the row honest: the value the
// box is a function of is taken from the mode, so a frame composed after the key
// cannot show what the frame before it showed.
//
// It is asserted through the SUBJECT and not through pixels, for the refresh
// rule's own reason: the picture is recomposed exactly when this value changes,
// so a subject that did not move is a box that would not redraw.
func TestTheSubjectFollowsTheArmedFlag(t *testing.T) {
	a, v, _ := atOnMap(t)
	v.sel = selection{atLoID}

	down := v.readoutSubjectOf(60)
	if down.Armed {
		t.Fatal("a freshly opened map screen reports an armed attack")
	}

	atArm(a, v)
	up := v.readoutSubjectOf(60)
	if !up.Armed {
		t.Error("the subject did not follow the key that armed the mode")
	}
	if up == down {
		t.Error("the subject is unchanged, so the box would not be recomposed")
	}

	// And it follows the press that spends it, which is the half a reader would
	// otherwise have to take on trust: the row is what tells a player the mode
	// is gone.
	atPress(a, v, atEmptyCol, atEmptyRow)
	if v.readoutSubjectOf(60).Armed {
		t.Error("the subject still reports an armed attack after the press that spent it")
	}
}

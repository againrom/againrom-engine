package sim

import "testing"

// sgPair is a member and a candidate in cells 2 apart, ready for candidateCost.
func sgPair(t *testing.T, second bool, member, candidate Entity) *World {
	t.Helper()
	w := engWorld(t, Relations{}, member, candidate)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

func sgMember(tokenSize uint8, mind int32) Entity {
	e := engReachFighter(1, 2, 10, 10, DomainGround, 1)
	e.TokenSize, e.Mind = tokenSize, mind
	return e
}

// sgCandidate is one cell away from sgMember's cell plus one, a ground
// candidate of reach 4 (preference 4) or a ghost (preference 1).
func sgCandidate(prefFour bool, tokenSize uint8, mind int32) Entity {
	domain, reach := DomainGhost, uint8(1)
	if prefFour {
		domain, reach = DomainGround, 4
	}
	e := engReachFighter(2, 3, 11, 10, domain, reach)
	e.TokenSize, e.Mind = tokenSize, mind
	return e
}

func TestPreferenceModifiersReadTheProfileGate(t *testing.T) {
	t.Parallel()
	const plain, one, four int32 = 256, 384, 192 // d=1 cost, pref 1's +half, pref 4's -quarter
	cases := []struct {
		name         string
		second       bool
		mSize, cSize uint8
		mMind, cMind int32
		order        uint8
		prefFour     bool
		want         int32
	}{
		{name: "first control", mSize: 1, cSize: 1, mMind: 20, cMind: 20, order: orderGuard, want: one},
		{name: "first control pref 4", mSize: 1, cSize: 1, mMind: 20, cMind: 20, order: orderGuard, prefFour: true, want: four},
		{name: "first candidate size withholds", mSize: 1, cSize: 2, mMind: 20, cMind: 20, order: orderGuard, want: plain},
		{name: "first candidate Mind 14 withholds", mSize: 1, cSize: 1, mMind: 20, cMind: 14, order: orderGuard, prefFour: true, want: plain},
		{name: "first candidate Mind 15 admits", mSize: 1, cSize: 1, mMind: 20, cMind: 15, order: orderGuard, want: one},
		{name: "first member size is not read", mSize: 2, cSize: 1, mMind: 3, cMind: 20, order: orderGuard, want: one},
		{name: "first stand ground has no gate", mSize: 1, cSize: 2, mMind: 20, cMind: 3, order: orderStandGround, want: 512},
		{name: "second control", second: true, mSize: 1, cSize: 1, mMind: 20, cMind: 20, order: orderGuard, want: one},
		{name: "second actor size withholds", second: true, mSize: 2, cSize: 1, mMind: 20, cMind: 20, order: orderGuard, prefFour: true, want: plain},
		{name: "second actor Mind 14 withholds", second: true, mSize: 1, cSize: 1, mMind: 14, cMind: 20, order: orderGuard, want: plain},
		{name: "second actor Mind 15 admits", second: true, mSize: 1, cSize: 1, mMind: 15, cMind: 20, order: orderGuard, prefFour: true, want: four},
		{name: "second candidate size is not read", second: true, mSize: 1, cSize: 2, mMind: 20, cMind: 3, order: orderGuard, want: one},
		{name: "second stand ground reads the actor", second: true, mSize: 1, cSize: 1, mMind: 14, cMind: 20, order: orderStandGround, want: plain},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			w := sgPair(t, c.second, sgMember(c.mSize, c.mMind), sgCandidate(c.prefFour, c.cSize, c.cMind))
			if got := w.candidateCost(0, 1, c.order); got != c.want {
				t.Errorf("cost %d, want %d", got, c.want)
			}
		})
	}
}

// The veto stands whatever the gate would answer: a pref-0 cell is excluded for
// a withheld pair too.
func TestTheVetoIsNotGated(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		member := sgMember(2, 3)
		flier := engReachFighter(2, 3, 11, 10, DomainAir, 1)
		flier.TokenSize, flier.Mind = 2, 3
		w := sgPair(t, second, member, flier)
		if got := w.candidateCost(0, 1, orderGuard); got != scoreSeed {
			t.Errorf("second=%v: a vetoed pair scored %d", second, got)
		}
	}
}

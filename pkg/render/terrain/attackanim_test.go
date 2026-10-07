package terrain

// The swing selection: what it draws, the four ways it refuses, and its
// totality (AC-8).
//
// Nothing here reads SelectAttackFrame to decide what to expect. The frame is
// computed in the test from the block arithmetic written out by hand —
// AttackBase + slot*AttackSlot + track[step] — so a selector that took its slot
// from the wrong block, or its step from the wrong clock, disagrees rather than
// agreeing with itself.

import "testing"

// swingAnim is a descriptor whose attack block sits well past a move block and
// an idle one, so a selection that read the wrong base lands somewhere this
// file's own arithmetic does not.
//
// D is 8, so the direction rule is the identity and the mirror is never set —
// the mirroring layout is exercised in its own case below, where it is the
// subject rather than a background condition.
func swingAnim() UnitAnim {
	return UnitAnim{
		S: 16, D: 8,
		MoveBase: 16, AttackBase: 100, DyingBase: 300, TailBase: 400,
		MoveSlot: 4, MoveWind: 1, DyingSlot: 5, IdleSlot: 6, AttackSlot: 7,
		MoveTrack: []int{0, 1, 2}, IdleTrack: []int{0, 1},
		AttackTrack: []int{0, 2, 1, 3},
		MoveOK:      true, IdleOK: true, AttackOK: true,
	}
}

// TestTheSwingIsTheAttackBlocksOwnFrame is AC-8's first half, over every octant
// and every tick of the run: the frame is the block arithmetic and nothing else,
// and the clock walks the track in the order it was expanded in, one entry per
// tick and NO modulo.
func TestTheSwingIsTheAttackBlocksOwnFrame(t *testing.T) {
	a := swingAnim()
	const count = 1000
	for oct := 0; oct < 8; oct++ {
		for swing := 0; swing < len(a.AttackTrack); swing++ {
			want := a.AttackBase + oct*a.AttackSlot + a.AttackTrack[swing]
			frame, mirror, ok := SelectAttackFrame(a, count, oct, swing)
			if !ok {
				t.Fatalf("oct %d swing %d: refused inside its own run", oct, swing)
			}
			if frame != want {
				t.Errorf("oct %d swing %d: frame %d, want %d", oct, swing, frame, want)
			}
			if mirror {
				t.Errorf("oct %d swing %d: mirrored at D 8, which never mirrors", oct, swing)
			}
		}
	}
}

// TestTheRunPlaysOnceAndEnds is the decoded length (ANIM-RUN-004,
// ANIM-PHASE-003): the run lasts exactly the expanded track's own length, and a
// clock at or past it is a run that is OVER — refused, so the caller falls
// through to the standing drawing, which is what the engine forcing the state
// back to 0 looks like from this side.
//
// A modulus here would loop the swing forever, and that is the behaviour of the
// MOVE arm — the only one of the engine's own switch that takes one.
func TestTheRunPlaysOnceAndEnds(t *testing.T) {
	a := swingAnim()
	n := len(a.AttackTrack)
	if _, _, ok := SelectAttackFrame(a, 1000, 3, n-1); !ok {
		t.Errorf("the run's LAST tick (%d of %d) was refused", n-1, n)
	}
	for _, swing := range []int{n, n + 1, 2 * n, 1 << 20} {
		if frame, _, ok := SelectAttackFrame(a, 1000, 3, swing); ok {
			t.Errorf("a clock of %d is past a run of %d and answered frame %d; the run has ended",
				swing, n, frame)
		}
	}
}

// TestTheSwingHalvesItsFacingAtTheMirroringLayout is the direction rule, shared
// with the two selections that ship: at D 5 octants 5..7 fold onto slots 3..1
// about the vertical axis. Written out here rather than read off unitSlot.
func TestTheSwingHalvesItsFacingAtTheMirroringLayout(t *testing.T) {
	a := swingAnim()
	a.S, a.D = 9, 5
	for _, tc := range []struct {
		oct, slot int
		mirror    bool
	}{
		{0, 0, false}, {1, 1, false}, {2, 2, false}, {3, 3, false}, {4, 4, false},
		{5, 3, true}, {6, 2, true}, {7, 1, true},
	} {
		want := a.AttackBase + tc.slot*a.AttackSlot + a.AttackTrack[0]
		frame, mirror, ok := SelectAttackFrame(a, 1000, tc.oct, 0)
		if !ok {
			t.Fatalf("oct %d: refused", tc.oct)
		}
		if frame != want || mirror != tc.mirror {
			t.Errorf("oct %d: frame %d mirror %v, want %d %v", tc.oct, frame, mirror, want, tc.mirror)
		}
	}
}

// TestTheSwingRefusesWhatItCannotDraw is AC-8's second half: the four refusals,
// each reported through the third return so the caller can fall through to the
// live selection rather than draw a frame it cannot tell from a real one.
func TestTheSwingRefusesWhatItCannotDraw(t *testing.T) {
	for _, tc := range []struct {
		what  string
		mut   func(*UnitAnim)
		count int
	}{
		{"no attack block at all", func(a *UnitAnim) { a.AttackSlot = 0 }, 1000},
		{"a negative slot, which an absent phase clamps to", func(a *UnitAnim) { a.AttackSlot = -1 }, 1000},
		{"a failed gate over a full track", func(a *UnitAnim) { a.AttackOK = false }, 1000},
		{"a true gate over an empty track", func(a *UnitAnim) { a.AttackTrack = nil }, 1000},
		{"an index past the sheet's own frames", func(a *UnitAnim) {}, 100},
		{"a sheet with no frames at all", func(a *UnitAnim) {}, 0},
		{"a base before the sheet", func(a *UnitAnim) { a.AttackBase = -500 }, 1000},
	} {
		t.Run(tc.what, func(t *testing.T) {
			a := swingAnim()
			tc.mut(&a)
			frame, mirror, ok := SelectAttackFrame(a, tc.count, 3, 1)
			if ok {
				t.Errorf("accepted, answering frame %d", frame)
			}
			if frame != 0 || mirror {
				t.Errorf("a refusal answered frame %d mirror %v, want 0 and false", frame, mirror)
			}
		})
	}
}

// TestTheSwingIsTotalOverAnyClock is AC-8's last clause. The clock is the
// drawing side's own count and nothing bounds it before it arrives here, so
// every int must ANSWER — with a frame inside the run or with a refusal, never
// with a panic and never with an index taken outside the track.
func TestTheSwingIsTotalOverAnyClock(t *testing.T) {
	a := swingAnim()
	for _, swing := range []int{-1 << 40, -7, -1, 0, 1, 2, 3, 4, 1 << 40} {
		frame, _, ok := SelectAttackFrame(a, 1000, 2, swing)
		if !ok {
			if frame != 0 {
				t.Errorf("swing %d: refused, answering frame %d rather than 0", swing, frame)
			}
			continue
		}
		lo := a.AttackBase + 2*a.AttackSlot
		if frame < lo || frame >= lo+a.AttackSlot {
			t.Errorf("swing %d: frame %d, outside its own direction slot [%d,%d)",
				swing, frame, lo, lo+a.AttackSlot)
		}
	}
	// A NEGATIVE clock is out of the run and refused with the rest — it is not
	// folded back into the track, which is what a euclidean reduction would do
	// and what this arm must not.
	if _, _, ok := SelectAttackFrame(a, 1000, 2, -1); ok {
		t.Error("a negative clock answered a frame; it is outside the run")
	}
	// And equal inputs give equal answers, which is what makes it a function of
	// its arguments rather than of anything it reads.
	f1, m1, o1 := SelectAttackFrame(a, 1000, 5, 2)
	f2, m2, o2 := SelectAttackFrame(a, 1000, 5, 2)
	if f1 != f2 || m1 != m2 || o1 != o2 {
		t.Errorf("two identical calls answered (%d,%v,%v) and (%d,%v,%v)", f1, m1, o1, f2, m2, o2)
	}
}

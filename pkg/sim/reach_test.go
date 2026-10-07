package sim

// The reach field, the strike distance and the version-23 byte form (0104).
//
// combat_test.go's own cbEnt/cbWorld/cbAt/cbOrder/cbFighter fixtures are
// reused here rather than duplicated: they build exactly the bare, one-blow
// fighters this file needs, and a second copy of them would be the thing
// this package's own doc comments warn a reader against trusting.

import (
	"strconv"
	"strings"
	"testing"
)

func TestStrikeDistanceEqualsTheChebyshevFloorOverASeparationRange(t *testing.T) {
	for dx := int32(0); dx <= 24; dx++ {
		for dy := int32(0); dy <= 24; dy++ {
			a := Entity{X: 0, Y: 0}
			b := Entity{X: dx, Y: dy}
			chebyshev := dx
			if dy > chebyshev {
				chebyshev = dy
			}
			want := chebyshev
			if want < 1 {
				want = 1
			}
			if got := strikeDistance(a, b); got != want {
				t.Fatalf("strikeDistance at separation (%d,%d) is %d, want max(1, Chebyshev) = %d",
					dx, dy, got, want)
			}
			// Symmetric: the distance is a property of the pair, not of which
			// entity is the attacker.
			if got := strikeDistance(b, a); got != want {
				t.Fatalf("strikeDistance(b, a) at separation (%d,%d) is %d, want %d", dx, dy, got, want)
			}
		}
	}
}

func TestAReachOfFourStrikesAtFourRefusesAtFiveAndStopsItsWalkAtFour(t *testing.T) {
	t.Run("in reach at exactly the boundary", func(t *testing.T) {
		a := Entity{X: 0, Y: 0, Reach: 4}
		v := Entity{X: 4, Y: 0}
		if !inReach(a, v) {
			t.Error("a reach of 4 did not strike at a Chebyshev distance of exactly 4")
		}
	})

	t.Run("refused one cell past the boundary", func(t *testing.T) {
		a := Entity{X: 0, Y: 0, Reach: 4}
		v := Entity{X: 5, Y: 0}
		if inReach(a, v) {
			t.Error("a reach of 4 struck at a Chebyshev distance of 5")
		}
	})

	t.Run("an approach stops exactly at the boundary and then strikes", func(t *testing.T) {
		// A charge of 1 and AlwaysHits, so the first strike attempt that
		// finds the victim in reach lands with nothing else able to refuse
		// it — cbFighter's own fixture.
		a := cbFighter(1, 0, 0, 1, 0)
		a.Reach = 4
		v := cbEnt(2, 14, 0) // stationary; the attacker's own walk covers the gap
		w := cbWorld(t, 5, a, v)

		Step(w, []Command{cbOrder(1, 2)})
		for k := 0; k < 30; k++ {
			Step(w, nil)
		}

		if got := cbAt(t, w, 1); got.X != 10 || got.Y != 0 {
			t.Errorf("the attacker rests at (%d,%d), want (10,0) — a Chebyshev distance of 4 from the victim",
				got.X, got.Y)
		}
		if hp := cbAt(t, w, 2).HP; hp == 100 {
			t.Error("no blow landed once the attacker's walk stopped at its own reach")
		}

		// A further advance moves nothing more: the walk does not continue
		// closing once it is already close enough to strike.
		before := cbAt(t, w, 1)
		Step(w, nil)
		if after := cbAt(t, w, 1); after.X != before.X || after.Y != before.Y {
			t.Errorf("the attacker kept closing after reaching its own reach: (%d,%d) to (%d,%d)",
				before.X, before.Y, after.X, after.Y)
		}
	})

	// AC-3's OTHER HALF, and it is the half that makes the first one mean
	// anything: a spear-armed attacker in the identical situation still walks
	// all the way in.
	t.Run("a reach of one closes to an adjacent cell in the same situation", func(t *testing.T) {
		a := cbFighter(1, 0, 0, 1, 0)
		a.Reach = 1
		v := cbEnt(2, 14, 0)
		w := cbWorld(t, 5, a, v)

		Step(w, []Command{cbOrder(1, 2)})
		for k := 0; k < 30; k++ {
			Step(w, nil)
		}

		if got := cbAt(t, w, 1); got.X != 13 || got.Y != 0 {
			t.Errorf("the reach-1 attacker rests at (%d,%d), want (13,0) — adjacent to the victim",
				got.X, got.Y)
		}
		if hp := cbAt(t, w, 2).HP; hp == 100 {
			t.Error("no blow landed once the reach-1 attacker stood adjacent")
		}
	})
}

func TestTheConstructorFoldsAZeroReachToOneAndLeavesOthersAlone(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{
		{ID: 1, X: 1, Y: 1},           // names none
		{ID: 2, X: 2, Y: 2, Reach: 1}, // already the floor
		{ID: 3, X: 3, Y: 3, Reach: 7},
		{ID: 4, X: 4, Y: 4, Reach: 255}, // the top of the byte
	})
	want := map[EntityID]uint8{1: 1, 2: 1, 3: 7, 4: 255}
	for _, e := range w.Entities() {
		if e.Reach != want[e.ID] {
			t.Errorf("entity %d carries reach %d, want %d", e.ID, e.Reach, want[e.ID])
		}
	}
}

// ------------------------------------------------------------------ AC-4

// TestAWorldHoldingSeveralReachesRoundTrips is AC-4's first clause: several
// distinct reaches on one world, marshalled and read back unchanged, byte for
// byte and digest for digest.
func TestAWorldHoldingSeveralReachesRoundTrips(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10, Reach: 1},
		{ID: 2, X: 2, Y: 2, HP: 10, MaxHP: 10, Reach: 4},
		{ID: 3, X: 3, Y: 3, HP: 10, MaxHP: 10, Reach: 100},
		{ID: 4, X: 4, Y: 4, HP: 10, MaxHP: 10, Reach: 255},
	}
	w := mustWorld(t, 1, Bounds{Width: 16, Height: 16}, ents)

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if !equalState(snap(&back), snap(w)) {
		t.Errorf("the round trip did not reproduce a world holding several reaches")
	}
	if got, want := back.Hash(), w.Hash(); got != want {
		t.Errorf("the round trip hashes %#016x, the original %#016x", got, want)
	}
	wantReach := map[EntityID]uint8{1: 1, 2: 4, 3: 100, 4: 255}
	for _, e := range back.Entities() {
		if e.Reach != wantReach[e.ID] {
			t.Errorf("entity %d decoded with reach %d, want %d", e.ID, e.Reach, wantReach[e.ID])
		}
	}
}

func TestUnmarshalRefusesAZeroReachAndAcceptsTheTopOfItsRange(t *testing.T) {
	valid, err := mustWorld(t, 1, Bounds{Width: 4, Height: 4},
		[]Entity{{ID: 1, X: 1, Y: 1, Reach: 5}}).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	// The record's own reach byte, written out from the contract rather than
	// read off entityLen for the offset itself: entityLen now measures past
	// it to the post's own tail (0106), so this is no longer the record's
	// last byte.
	reachAt := headerLen + 3*16 + 118

	var w World
	if err := w.UnmarshalBinary(withByte(valid, reachAt, 0)); err == nil {
		t.Fatal("accepted a reach of 0")
	}
	if err := w.UnmarshalBinary(withByte(valid, reachAt, 255)); err != nil {
		t.Fatalf("a reach of 255 was refused: %v", err)
	}
	if got := w.Entities()[0].Reach; got != 255 {
		t.Errorf("a reach of 255 decoded as %d", got)
	}
	// And the unspoiled form is accepted, so the two cases above measure the
	// reach check and not something else about this fixture.
	if err := w.UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
}

// TestAVersion22FormIsRefused is AC-4's remaining clause, on
// TestAVersion20FormIsRefused's own pattern (actorform_test.go): a
// byte-for-byte valid current-version form, spoiled at its version byte
// alone, isolates the version check from everything else this story
// changed, and the message must name both the version refused and the one
// this build reads.
func TestAVersion22FormIsRefused(t *testing.T) {
	valid, err := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}}).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var w World
	err = w.UnmarshalBinary(withByte(valid, 0, 22))
	if err == nil {
		t.Fatal("accepted a version-22 form")
	}
	// As in actorform_test.go: the refused version is this fixture's own and
	// stays a literal, the version this build reads comes from formatVersion.
	if !strings.Contains(err.Error(), "22") || !strings.Contains(err.Error(), strconv.Itoa(int(formatVersion))) {
		t.Errorf("refusal %q does not name both versions", err.Error())
	}
}

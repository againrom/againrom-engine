package sim

import (
	"bytes"
	"slices"
	"testing"
)

// setFleeSearchRadius gives actor id's Mover record the search radius r, as a
// LOAD of a SAV holding r at mover byte +0x08 does.
func setFleeSearchRadius(t *testing.T, w *World, id EntityID, r byte) {
	t.Helper()
	rec := w.moverFor(id)
	if rec == nil {
		t.Fatalf("actor %d holds no Mover record", id)
	}
	held := *rec
	held.Radius = r
	w.setMover(held)
}

// fled reports whether actor 0 holds a move away from a hostile on its right.
func fled(w *World) bool {
	e := w.entities[0]
	return e.HasTarget && e.TargetX < e.X
}

// Wimpy and explicit Retreat search the square of the mover byte +0x08, the
// constructor's 5, not ScanRange (AI-690, AI-692): a hostile at Chebyshev 5
// is found with ScanRange 1, one at 6 is not with ScanRange 20.
func TestWimpyAndRetreatSearchTheMoverRadiusNotScanRange(t *testing.T) {
	for _, tc := range []struct {
		name      string
		dx        int32
		scanRange uint8
		want      bool
	}{
		{"Chebyshev 5 found", 5, 1, true},
		{"Chebyshev 6 not found", 6, 20, false},
	} {
		t.Run("Wimpy "+tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, 20, 20, 8)
			self.Wimpy, self.ScanRange = 8, tc.scanRange
			hostile := withdrawalFighter(2, 3, 20+tc.dx, 20-tc.dx/2, 100)
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
			if got := w.fleeSearchRadius(0); got != 5 {
				t.Fatalf("constructed search radius = %d, want 5", got)
			}
			regWithdrawalPass(w, nil)
			if got := fled(w); got != tc.want {
				t.Fatalf("Wimpy fled = %t, want %t: %+v", got, tc.want, w.entities[0])
			}
		})
		t.Run("Retreat "+tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, 20, 20, 100)
			self.ScanRange = tc.scanRange
			hostile := withdrawalFighter(2, 3, 20+tc.dx, 20+tc.dx/2, 100)
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
			commandBeforePass(w, []Command{GroupRetreat(1, 2, 91)})
			if got := fled(w); got != tc.want {
				t.Fatalf("Retreat fled = %t, want %t: %+v", got, tc.want, w.entities[0])
			}
		})
	}
}

// A radius LOAD restored is the one searched, a 0 included (AI-693).
func TestWimpySearchesALoadedRadius(t *testing.T) {
	for _, r := range []byte{0, 1} {
		self := withdrawalFighter(1, 2, 20, 20, 8)
		self.Wimpy = 8
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, withdrawalFighter(2, 3, 21, 20, 100))
		setFleeSearchRadius(t, w, 1, r)
		regWithdrawalPass(w, nil)
		if got, want := fled(w), r == 1; got != want {
			t.Fatalf("radius %d: Wimpy fled = %t, want %t", r, got, want)
		}
	}
}

// Withdraw keeps the literal radius 2 (AI-690), whatever the mover byte holds.
func TestWithdrawSearchesRadiusTwo(t *testing.T) {
	for _, tc := range []struct {
		dx   int32
		want bool
	}{{2, true}, {3, false}} {
		self := withdrawalFighter(1, 2, 20, 20, 9)
		self.Withdraw = 30
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, withdrawalFighter(2, 3, 20+tc.dx, 20, 100))
		setFleeSearchRadius(t, w, 1, 40)
		regWithdrawalPass(w, nil)
		if got := fled(w); got != tc.want {
			t.Fatalf("Withdraw at %d fled = %t, want %t", tc.dx, got, tc.want)
		}
	}
}

// The square is centred on the top-left cell of a 3x3 actor (AI-696): it
// reaches 5 cells past the left edge and 3 past the right edge.
func TestWithdrawalSquareIsCentredOnTheTopLeftFootprintCell(t *testing.T) {
	for _, tc := range []struct {
		name string
		x    int32
		want bool
	}{
		{"5 left of the top-left cell", 15, true},
		{"3 right of the right edge", 25, true},
		{"5 right of the right edge", 27, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, 20, 20, 8)
			self.Wimpy, self.TokenSize = 8, 3
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, withdrawalFighter(2, 3, tc.x, 20, 100))
			if got := len(w.withdrawalHostiles(0, w.fleeSearchRadius(0))) == 1; got != tc.want {
				t.Fatalf("hostile at x=%d listed = %t, want %t", tc.x, got, tc.want)
			}
			regWithdrawalPass(w, nil)
			if got := w.entities[0].HasTarget; got != tc.want {
				t.Fatalf("hostile at x=%d: Wimpy moved = %t, want %t", tc.x, got, tc.want)
			}
		})
	}
}

// A 2x2 hostile wholly inside the square is listed once per cell, and the
// flee point averages the four entries (AI-697, AI-698); one with two cells
// inside is listed twice.
func TestWideHostileWeighsOncePerCellInsideTheSquare(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 8)
	self.Wimpy = 8
	wide := withdrawalFighter(2, 3, 22, 16, 100)
	wide.TokenSize = 2
	small := withdrawalFighter(3, 3, 17, 24, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, wide, small)
	list := w.withdrawalHostiles(0, 5)
	if want := []int{1, 1, 1, 1, 2}; !slices.Equal(list, want) {
		t.Fatalf("listed %v, want %v", list, want)
	}
	tx, ty, _ := w.fleeCell(0, list)
	ox, oy, _ := w.fleeCell(0, []int{1, 2})
	if tx == ox && ty == oy {
		t.Fatalf("four entries and one give the same flee cell (%d,%d)", tx, ty)
	}
	regWithdrawalPass(w, nil)
	if e := w.entities[0]; !e.HasTarget || e.TargetX != tx || e.TargetY != ty {
		t.Fatalf("Wimpy destination (%d,%d,%t), want the four-entry flee cell (%d,%d)", e.TargetX, e.TargetY, e.HasTarget, tx, ty)
	}
	w.entities[1].X, w.entities[1].Y = 24, 14
	if got := w.withdrawalHostiles(0, 5); !slices.Equal(got, []int{1, 1, 2}) {
		t.Fatalf("2x2 hostile with two cells inside listed %v, want two entries", got)
	}
}

// The square reads only the ground slot of each cell: a flyer is never
// listed (AI-697); a ghost is.
func TestWithdrawalSquareListsNoFlyer(t *testing.T) {
	for _, tc := range []struct {
		domain Domain
		want   bool
	}{{DomainAir, false}, {DomainGhost, true}} {
		self := withdrawalFighter(1, 2, 20, 20, 8)
		self.Wimpy = 8
		hostile := withdrawalFighter(2, 3, 21, 20, 100)
		hostile.Domain = tc.domain
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
		if got := len(w.withdrawalHostiles(0, 5)) != 0; got != tc.want {
			t.Fatalf("domain %d listed = %t, want %t", tc.domain, got, tc.want)
		}
		regWithdrawalPass(w, nil)
		if got := fled(w); got != tc.want {
			t.Fatalf("domain %d: Wimpy fled = %t, want %t", tc.domain, got, tc.want)
		}
	}
}

// The second game keeps ScanRange and lists each hostile once by its anchor.
func TestSecondGameWithdrawalKeepsScanRange(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 8)
	self.ScanRange = 7
	wide := withdrawalFighter(2, 3, 26, 20, 100)
	wide.TokenSize = 2
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, wide)
	w.rom2 = &rom2ScriptState{}
	if got := w.fleeSearchRadius(0); got != 7 {
		t.Fatalf("second-game radius = %d, want ScanRange 7", got)
	}
	if got := w.withdrawalHostiles(0, 7); !slices.Equal(got, []int{1}) {
		t.Fatalf("second-game list %v, want one anchor entry", got)
	}
}

// The World byte form keeps a radius that is not the constructor's, 0 and 40
// alike, and writes no radius form while every radius is 5.
func TestMoverRadiusSurvivesTheByteForm(t *testing.T) {
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), withdrawalFighter(1, 2, 20, 20, 8), withdrawalFighter(2, 3, 30, 20, 100))
	plain, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(plain, []byte("MRD1")) {
		t.Fatal("constructed radii wrote a radius form")
	}
	for _, r := range []byte{0, 40} {
		setFleeSearchRadius(t, w, 1, r)
		back := worldRoundTripForTest(t, w)
		if got := back.fleeSearchRadius(0); got != int64(r) {
			t.Fatalf("decoded radius = %d, want %d", got, r)
		}
		var m [180]byte
		if !back.ApplyMover(1, &m) || m[8] != r {
			t.Fatalf("SAVE writes +0x08 = %d, want %d", m[8], r)
		}
		if back.Hash() == mustHashOf(t, plain) {
			t.Fatalf("radius %d hashes as the constructor's", r)
		}
	}
}

func mustHashOf(t *testing.T, b []byte) uint64 {
	t.Helper()
	var w World
	if err := w.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	return w.Hash()
}

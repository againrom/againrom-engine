package sim

// The grid, the mode and the stall count as CARRIED state: what the byte form
// and the digest have to tell apart, and what they have to treat as one world.
//
// The refusal sweep and the offset table live in binary_test.go; this file is
// the positive half — which pairs of worlds must differ, which single pair must
// not, and that a form written out and read back is the same world twice.

import (
	"bytes"
	"testing"
)

// gfBounds and gfGrid are one small world's extent and passability, spelled out
// so that a row and a column cannot be confused: no row repeats, both defined
// bits are used, and the two ends differ.
var gfBounds = Bounds{Width: 5, Height: 4}

func gfGrid() []byte {
	return []byte{
		0, 1, 0, 0, 2,
		0, 0, 1, 0, 0,
		3, 0, 0, 1, 0,
		0, 2, 0, 0, 1,
	}
}

func gfEnts() []Entity {
	return []Entity{
		{ID: 2, X: 0, Y: 0, TargetX: 4, TargetY: 3, Class: 6, HasTarget: true, Stall: 9},
		{ID: 5, X: 4, Y: 0, Class: -2},
	}
}

func gfWorld(t *testing.T, mode Mode, grid []byte) *World {
	t.Helper()
	return mustWorldGrid(t, 0x5eed, gfBounds, mode, grid, gfEnts())
}

func gfForm(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

// TestTheGridAndTheModeAreCanonicalState is AC-6. Four worlds: a base, the same
// with one grid bit toggled on the ground layer, the same with one toggled on
// the AIR layer, and the same in the other mode. All four must differ pairwise
// in byte form and in digest.
//
// The air case is the one worth stating: nothing in this package reads that bit,
// so a design that folded the grid down to what it uses — a bitset of blocked
// ground cells, say — would pass every movement test in the tree and fail here.
// A byte the simulation ignores is still a byte the map carried, and two worlds
// that differ in it are two worlds.
func TestTheGridAndTheModeAreCanonicalState(t *testing.T) {
	ground := gfGrid()
	ground[7] ^= blockGround
	air := gfGrid()
	air[7] ^= blockAir

	worlds := []struct {
		what string
		w    *World
	}{
		{"the base world", gfWorld(t, ModeCanonical, gfGrid())},
		{"one blocks-ground bit toggled", gfWorld(t, ModeCanonical, ground)},
		{"one blocks-air bit toggled", gfWorld(t, ModeCanonical, air)},
		{"the other routing mode", gfWorld(t, ModeOptimised, gfGrid())},
	}

	for i := range worlds {
		for j := i + 1; j < len(worlds); j++ {
			a, b := worlds[i], worlds[j]
			if bytes.Equal(gfForm(t, a.w), gfForm(t, b.w)) {
				t.Errorf("%s and %s marshal identically", a.what, b.what)
			}
			if a.w.Hash() == b.w.Hash() {
				t.Errorf("%s and %s hash %#016x", a.what, b.what, a.w.Hash())
			}
		}
	}
}

// TestNoGridAndAnAllZeroGridAreOneWorld is AC-6's last clause, and it is an
// equality rather than an equivalence: the two must be the same BYTES, not
// merely behave alike, because "behaves alike" is a claim every later story
// would have to keep true by hand.
func TestNoGridAndAnAllZeroGridAreOneWorld(t *testing.T) {
	absent := gfWorld(t, ModeCanonical, nil)
	zeroes := gfWorld(t, ModeCanonical, make([]byte, gfBounds.Width*gfBounds.Height))

	if !bytes.Equal(gfForm(t, absent), gfForm(t, zeroes)) {
		t.Errorf("a no-grid world and an all-zero-grid world marshal to\n % x\n % x",
			gfForm(t, absent), gfForm(t, zeroes))
	}
	if absent.Hash() != zeroes.Hash() {
		t.Errorf("digests %#016x and %#016x", absent.Hash(), zeroes.Hash())
	}

	// And the form they share really does carry the cells: a form that dropped
	// the section would be equal here for the wrong reason.
	if want := 61 + 34 + 3*int(gfBounds.Width*gfBounds.Height) + relationLen + 2*492 + 2*4 + groupCountLen + sackCountLen + 2*carryCountLen + 2*equipRecordLen + 2*treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(2)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(gfForm(t, absent)) != want {
		t.Errorf("the shared form is %d bytes, want %d", len(gfForm(t, absent)), want)
	}
}

// TestAGridWorldSurvivesTheRoundTrip is AC-7 in both modes: marshal, unmarshal
// into a second world, and the second world's whole form is the first's.
func TestAGridWorldSurvivesTheRoundTrip(t *testing.T) {
	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		first := gfWorld(t, mode, gfGrid())
		form := gfForm(t, first)

		var back World
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatalf("mode %d: UnmarshalBinary: %v", mode, err)
		}
		if got := gfForm(t, &back); !bytes.Equal(got, form) {
			t.Errorf("mode %d: the decoded world marshals to\n % x\nwant\n % x", mode, got, form)
		}
		if back.Hash() != first.Hash() {
			t.Errorf("mode %d: digests %#016x and %#016x", mode, back.Hash(), first.Hash())
		}
		if back.mode != mode {
			t.Errorf("mode %d decoded as %d", mode, back.mode)
		}
		if !bytes.Equal(back.grid, gfGrid()) {
			t.Errorf("mode %d: the grid decoded as % x", mode, back.grid)
		}
		if got := back.entities[0].Stall; got != 9 {
			t.Errorf("mode %d: the stall count decoded as %d, want 9", mode, got)
		}

		// The decoded world owns its grid: the buffer it came from is the
		// caller's to reuse.
		for i := range form {
			form[i] ^= 0xff
		}
		if !bytes.Equal(back.grid, gfGrid()) {
			t.Errorf("mode %d: mutating the source bytes reached the decoded grid: % x", mode, back.grid)
		}
	}
}

// TestTheGridCellCountFieldIsFourBytesWide: every other world in this package
// has a cell count that fits in two bytes, so a count written half as wide would
// leave all of them byte-identical. This one needs the third byte, and a 16-bit
// field would declare 24464 cells for it.
//
// The grid is not written out here — 90000 cells cannot be read by hand — so the
// check is on the count field, on the total length, and on one cell placed where
// only row-major order puts it.
func TestTheGridCellCountFieldIsFourBytesWide(t *testing.T) {
	const w, h = 300, 300
	grid := make([]byte, w*h)
	// The last cell of row 1, which row-major order puts at index 599 and
	// column-major order would not.
	grid[599] = blockGround | blockAir

	world := mustWorldGrid(t, 1, Bounds{Width: w, Height: h}, ModeCanonical, grid, nil)
	form := gfForm(t, world)

	if want := 61 + 34 + 3*w*h + relationLen + groupCountLen + sackCountLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(0)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(form) != want {
		t.Fatalf("a no-entity world over %d cells is %d bytes, want %d", w*h, len(form), want)
	}
	if got, want := form[30:34], []byte{0x90, 0x5f, 0x01, 0x00}; !bytes.Equal(got, want) {
		t.Errorf("the grid cell count field is % x, want % x for %d cells", got, want, w*h)
	}
	if got := form[34+599]; got != blockGround|blockAir {
		t.Errorf("the cell at index 599 encoded as %#02x, want %#02x", got, blockGround|blockAir)
	}

	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != world.Hash() {
		t.Errorf("the round trip hashes %#016x, the original %#016x", back.Hash(), world.Hash())
	}
}

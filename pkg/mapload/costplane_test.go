package mapload

import (
	"testing"

	"againrom/pkg/formats/alm"
)

// The two planes a map describes, and the one thing deriving them must not
// do.

// cpWord assembles a tile word from the three fields the classifier splits it
// into, so a case says what it means rather than carrying a number a reader has
// to decode.
func cpWord(g, b, s int) uint16 { return uint16(g<<6 | b<<4 | s) }

// TestTheClassifierAnswersTheContractsTable — AC-9.
//
// Seven words: the water arm, the primary at the top of the blend, a MIDDLE
// blend level in each direction, and all three reject arms. The two middle
// levels are the ones a table of extremes would miss — at levels 1 and 5 the
// blend collapses to one scalar, so a reading that returned the named class's
// own cost instead of the blend would agree everywhere except at 2, 3 and 4.
func TestTheClassifierAnswersTheContractsTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what  string
		word  uint16
		class uint8
		cost  uint8
	}{
		// Water: taken before anything else, and costing the entry literal.
		{"a water cell", 512, classWater, 8},
		{"the water range's Land exception", 512 | 0x10 | 4, classLand, 8},
		// Strip group 7 — Mountain over Stones — at the top of the blend and in
		// the middle of it. Level 5 is the primary's own 16; level 2 is
		// (3*12 + 16) >> 2 = 13, which is neither scalar.
		{"group 7 at blend level 5", cpWord(7, 3, 0), classMountain, 16},
		{"group 7 at blend level 2", cpWord(7, 0, 0), classStones, 13},
		// Strip group 5 — Cracked over Stones — at the midpoint: (12 + 6) >> 1 = 9.
		{"group 5 at blend level 3", cpWord(5, 0, 1), classCracked, 9},
		// The three rejects.
		{"a sub-cell of 14", cpWord(0, 0, 14), classReject, 255},
		{"a low nibble of 9 inside the water range", 512 | 9, classReject, 255},
		{"strip group 13", cpWord(13, 0, 0), classReject, 255},
	} {
		class, cost := classify(tc.word)
		if class != tc.class || cost != tc.cost {
			t.Errorf("%s (word %#x) classifies as (%d, %d), want (%d, %d)",
				tc.what, tc.word, class, cost, tc.class, tc.cost)
		}
	}
}

// cpPreStoryMountain is the mountain arm EXACTLY as this package derived it
// before the classifier existed, transcribed and not called: strip group 7, a
// sub-cell of 13 or less, and a blend level of 3 or more.
//
// It is the fixed point AC-11a measures against, so it must not be rewritten to
// follow the code — if the classifier and this ever disagree, this one is right
// about what the tree used to do and the classifier is what changed.
func cpPreStoryMountain(word uint16) bool {
	i := int(alm.TileIndex(word))
	s := i & 0xf
	b := (i >> 4) & 3
	g := (i >> 6) & 0xf
	return g == 7 && s <= 13 && blendLevel[b][s] >= 3
}

// TestTheBlockArmIsUnchangedOverEveryTileWord — AC-11a.
//
// The block plane's mountain arm now reads the classifier's class instead of
// testing the strip group itself, and the whole risk of this story is that the
// swap moved a block byte. It is settled EXHAUSTIVELY rather than by sample or
// by corpus: all 65 536 tile words, the old predicate against the new class.
//
// A corpus could not do this. Shipped maps reach 1024 of the 65 536 words at
// most, they reach no reject arm at all, and they never set bits 10 to 15 — so
// every word this test covers that a map does not is a word a customised map
// could carry and a census could never speak for.
func TestTheBlockArmIsUnchangedOverEveryTileWord(t *testing.T) {
	t.Parallel()

	mismatches, mountains := 0, 0
	for w := 0; w <= 0xffff; w++ {
		word := uint16(w)
		class, _ := classify(word)
		got, want := class == classMountain, cpPreStoryMountain(word)
		if got {
			mountains++
		}
		if got != want {
			mismatches++
			if mismatches <= 8 {
				t.Errorf("tile word %#06x: the classifier says Mountain=%v, the pre-story arm says %v",
					word, got, want)
			}
		}
	}
	if mismatches != 0 {
		t.Errorf("%d of 65536 tile words disagree", mismatches)
	}
	// And the predicate is not vacuous on either side: a classifier that never
	// said Mountain would agree with a pre-story arm that never did either.
	if mountains == 0 {
		t.Errorf("no tile word classifies as Mountain, so the agreement above is empty")
	}
}

// TestEveryTileWordCostsSomethingTheContractAllows — the cost half of the same
// exhaustive sweep.
//
// Over all 65 536 words the cost is either a reject's 255 or inside the range
// the ten scalars can blend to, which is 6 through 16. The bound matters because
// the blend is the story's only new arithmetic: a wrong shift, or a blend taken
// between the wrong pair, escapes it immediately.
func TestEveryTileWordCostsSomethingTheContractAllows(t *testing.T) {
	t.Parallel()

	rejects := 0
	for w := 0; w <= 0xffff; w++ {
		class, cost := classify(uint16(w))
		if class == classReject {
			rejects++
			if cost != 255 {
				t.Fatalf("tile word %#06x is a reject costing %d, want 255", w, cost)
			}
			continue
		}
		if cost < 6 || cost > 16 {
			t.Fatalf("tile word %#06x classifies as %d costing %d, outside the 6..16 the scalars blend to",
				w, class, cost)
		}
	}
	if rejects == 0 {
		t.Errorf("no tile word reaches a reject arm, so the case above measured nothing")
	}
}

// TestBothPlanesAreTotalAndSizedAtTheExtent — AC-10.
//
// The derivations are total, so a map assembled by hand with a short plane is
// answered rather than refused: the missing tail reads as zero — for the height
// plane the byte itself, for the cost plane the word zero, which classifies as
// strip group 0 at blend level 2, Land at 8.
func TestBothPlanesAreTotalAndSizedAtTheExtent(t *testing.T) {
	t.Parallel()

	// What the word zero comes to, stated once so the expectations below cannot
	// drift from the classifier without saying so.
	_, zeroCost := classify(0)

	m := &alm.Map{
		Width: 3, Height: 2,
		Tiles:     []uint16{cpWord(7, 3, 0), cpWord(5, 0, 1)}, // two of six
		Altitudes: []uint8{9, 8, 7, 6},                        // four of six
	}
	cost, height := Cost(m), Height(m)
	if len(cost) != 6 || len(height) != 6 {
		t.Fatalf("the planes are %d and %d byte(s), want 6 apiece", len(cost), len(height))
	}
	wantCost := []uint8{16, 9, zeroCost, zeroCost, zeroCost, zeroCost}
	wantHeight := []uint8{9, 8, 7, 6, 0, 0}
	for i := range wantCost {
		if cost[i] != wantCost[i] {
			t.Errorf("cost cell %d is %d, want %d", i, cost[i], wantCost[i])
		}
		if height[i] != wantHeight[i] {
			t.Errorf("height cell %d is %d, want %d", i, height[i], wantHeight[i])
		}
	}

	// A LONG plane is ignored past the extent, and a map with no cells has no
	// planes at all rather than empty ones of some other length.
	long := &alm.Map{Width: 1, Height: 1, Tiles: make([]uint16, 9), Altitudes: make([]uint8, 9)}
	if len(Cost(long)) != 1 || len(Height(long)) != 1 {
		t.Errorf("a long plane gave %d and %d byte(s) over one cell",
			len(Cost(long)), len(Height(long)))
	}
	for _, m := range []*alm.Map{nil, {Width: 0, Height: 5}, {Width: 5, Height: -1}} {
		if Cost(m) != nil || Height(m) != nil {
			t.Errorf("a map with no cells gave planes of %d and %d byte(s)",
				len(Cost(m)), len(Height(m)))
		}
	}
}

func TestTheCostPlaneReadsTheTilePlaneAndNothingElse(t *testing.T) {
	t.Parallel()

	tiles := []uint16{cpWord(7, 3, 0), cpWord(5, 0, 1), 512, cpWord(12, 3, 0)}
	bare := &alm.Map{Width: 4, Height: 1, Tiles: tiles}
	dressed := &alm.Map{
		Width: 4, Height: 1, Tiles: tiles,
		Overlay:   []uint8{1, 2, 3, 4},
		Altitudes: []uint8{200, 100, 50, 25},
	}
	a, b := Cost(bare), Cost(dressed)
	if len(a) != len(b) {
		t.Fatalf("the two planes are %d and %d byte(s)", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("cell %d costs %d on a bare map and %d on a dressed one", i, a[i], b[i])
		}
	}
	// And the Road cell is the cheap one, which is the story's own headline
	// where a reader can see it: strip group 12 at the top of the blend is 6
	// against Land's 8.
	if a[3] != 6 {
		t.Errorf("a road cell costs %d, want 6", a[3])
	}
}

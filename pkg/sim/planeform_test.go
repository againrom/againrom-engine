package sim

import (
	"bytes"
	"testing"
)

// The cost plane and the height plane as CANONICAL STATE: materialised when
// a world names neither, refused at any length but the one the bounds give,
// and visible to the byte form and the digest a byte at a time.
//
// What is NOT here is either plane being spent — no route and no rate reads one
// in this file. That separation is the point: a plane can be carried, hashed and
// round-tripped correctly and still reach nobody, and a test that mixed the two
// could not say which half had failed.

var pfBounds = Bounds{Width: 4, Height: 3}

const pfCells = 12

// pfWorld is a world over pfBounds carrying the three planes given, with one
// entity so the form has a record to put after them.
func pfWorld(t *testing.T, block, cost, height []byte) (*World, error) {
	t.Helper()
	return NewTerrainWorld(1, pfBounds, ModeCanonical,
		Terrain{Block: block, Cost: cost, Height: height},
		[]Entity{{ID: 1, X: 1, Y: 1}}, nil)
}

func pfMust(t *testing.T, block, cost, height []byte) *World {
	t.Helper()
	w, err := pfWorld(t, block, cost, height)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	return w
}

// TestAnUnnamedPlaneIsMaterialisedIntoTheForm — AC-1.
//
// The two planes are read out of the FORM rather than off the fields, because
// the claim is about what a world IS and not about what its constructor happened
// to store: a world that kept an absent plane as a nil and materialised it on
// the way out would pass a field check and fail this one on the day something
// else read the field.
func TestAnUnnamedPlaneIsMaterialisedIntoTheForm(t *testing.T) {
	t.Parallel()

	form, err := pfMust(t, nil, nil, nil).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if want := 61 + headerLen + 3*pfCells + relationLen + entityLen + routeCountLen +
		groupCountLen + sackCountLen + carryCountLen + equipRecordLen + treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(1)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(form) != want {
		t.Fatalf("the form is %d byte(s), want %d — a header, three planes of %d cells, "+
			"one record and one empty route", len(form), want, pfCells)
	}

	block := form[headerLen : headerLen+pfCells]
	cost := form[headerLen+pfCells : headerLen+2*pfCells]
	height := form[headerLen+2*pfCells : headerLen+3*pfCells]

	wantCost := bytes.Repeat([]byte{defaultCost}, pfCells)
	if !bytes.Equal(cost, wantCost) {
		t.Errorf("the materialised cost plane is % x, want % x", cost, wantCost)
	}
	if !bytes.Equal(height, make([]byte, pfCells)) {
		t.Errorf("the materialised height plane is % x, want all zero", height)
	}
	if !bytes.Equal(block, make([]byte, pfCells)) {
		t.Errorf("the materialised block plane is % x, want all zero", block)
	}

	// And the two ways of asking for that world are ONE world, which is what
	// "no flag records whether a plane was named" means where it can be seen.
	named := pfMust(t, nil, wantCost, make([]byte, pfCells))
	other, err := named.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(form, other) {
		t.Errorf("a world naming neither plane and one naming what absence materialises "+
			"write different forms:\n % x\n % x", form, other)
	}
	if a, b := pfMust(t, nil, nil, nil).Hash(), named.Hash(); a != b {
		t.Errorf("and they hash %#016x against %#016x", a, b)
	}
}

// TestAPlaneOfTheWrongLengthIsRefused — AC-1a, both halves.
//
// The decode half matters as much as the construction half and for the reason
// every refusal in this package pairs that way: a constructor that refused what
// the decoder accepted would let a world exist that could be read back but never
// built, and the asymmetry is one no caller could do anything about.
func TestAPlaneOfTheWrongLengthIsRefused(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what string
		n    int
	}{
		{"one byte short", pfCells - 1},
		{"one byte long", pfCells + 1},
	} {
		if _, err := pfWorld(t, nil, make([]byte, tc.n), nil); err == nil {
			t.Errorf("a cost plane %s was accepted", tc.what)
		}
		if _, err := pfWorld(t, nil, nil, make([]byte, tc.n)); err == nil {
			t.Errorf("a height plane %s was accepted", tc.what)
		}
	}

	// The decode half. The form is well formed in every other particular, and
	// one plane's worth of bytes is added to it and taken from it — so what the
	// decoder refuses is the length and not something else about the buffer.
	form, err := pfMust(t, nil, nil, nil).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if err := (&World{}).UnmarshalBinary(form); err != nil {
		t.Fatalf("the unspoiled form was refused: %v", err)
	}
	short := append(append([]byte(nil), form[:headerLen+3*pfCells-1]...), form[headerLen+3*pfCells:]...)
	if err := (&World{}).UnmarshalBinary(short); err == nil {
		t.Errorf("a form one plane byte short was accepted")
	}
	long := append(append([]byte(nil), form[:headerLen+3*pfCells]...), 0)
	long = append(long, form[headerLen+3*pfCells:]...)
	if err := (&World{}).UnmarshalBinary(long); err == nil {
		t.Errorf("a form one plane byte long was accepted")
	}
}

// TestOneByteOfEitherPlaneMovesTheDigest — AC-13.
//
// One byte, in each plane, at each end of the plane. A digest taken over a
// prefix of the form, or over the block plane alone, passes a whole-plane
// comparison and fails this.
func TestOneByteOfEitherPlaneMovesTheDigest(t *testing.T) {
	t.Parallel()

	base := pfMust(t, nil, nil, nil).Hash()
	for _, i := range []int{0, pfCells - 1} {
		cost := bytes.Repeat([]byte{defaultCost}, pfCells)
		cost[i] = defaultCost + 1
		if got := pfMust(t, nil, cost, nil).Hash(); got == base {
			t.Errorf("cost cell %d moved and the digest did not: both %#016x", i, got)
		}
		height := make([]byte, pfCells)
		height[i] = 1
		if got := pfMust(t, nil, nil, height).Hash(); got == base {
			t.Errorf("height cell %d moved and the digest did not: both %#016x", i, got)
		}
	}
}

// TestEveryPlaneByteRoundTrips — the other half of AC-12, over values a
// derivation can produce and two it cannot: 0, which makes a ground step free,
// and 255, which is the classifier's reject.
//
// The planes are NOT copied from one another, which is the shape this catches:
// an encoder that wrote the cost plane twice, or a decoder that read the height
// plane out of the cost plane's offset, agrees with itself on a uniform world.
func TestEveryPlaneByteRoundTrips(t *testing.T) {
	t.Parallel()

	cost := make([]byte, pfCells)
	height := make([]byte, pfCells)
	for i := range cost {
		cost[i] = byte(i * 23) // 0, 23, 46, … including 0 at cell 0
		height[i] = byte(255 - i*7)
	}
	w := pfMust(t, nil, cost, height)

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if !bytes.Equal(back.cost, cost) {
		t.Errorf("the cost plane read back as % x, want % x", back.cost, cost)
	}
	if !bytes.Equal(back.height, height) {
		t.Errorf("the height plane read back as % x, want % x", back.height, height)
	}
	if got := back.Hash(); got != w.Hash() {
		t.Errorf("the round trip hashes %#016x, want %#016x", got, w.Hash())
	}
}

// TestTheConstructorCopiesEveryPlane — a caller's slice stays the caller's, and
// the world's is the world's.
//
// It is the one shape a digest cannot catch after the fact: a retained backing
// array makes hashed simulation state writable from outside the package, and
// every check would go on agreeing right up until somebody wrote through it.
func TestTheConstructorCopiesEveryPlane(t *testing.T) {
	t.Parallel()

	block := make([]byte, pfCells)
	cost := bytes.Repeat([]byte{defaultCost}, pfCells)
	height := make([]byte, pfCells)
	w := pfMust(t, block, cost, height)
	before := w.Hash()

	block[0], cost[0], height[0] = 1, 99, 99
	if got := w.Hash(); got != before {
		t.Errorf("mutating the caller's slices moved the world's digest from %#016x to %#016x",
			before, got)
	}
}

package sim

// The carried load and what it costs: the two terms it is built from, the
// saturation that replaces them, the speed penalty that is its only consumer,
// and the three facts the byte form has to carry so that two worlds differing
// only in what they carry are two worlds.
//
// Every expectation here is written out as a literal worked from the claim's
// own arithmetic, never read back off the constants or the helpers under test:
// a test that computed "worn + sum/carryHalving" would pass against any
// divisor.

import (
	"bytes"
	"testing"
)

// wtBounds is the extent these fixtures are built against. Nothing here reads
// the terrain, so it is small enough to state and large enough for one step.
var wtBounds = Bounds{Width: 10, Height: 10}

// mustWeighedWorld builds a world over ents and stock and declares weights into
// it. The declaration goes through the production door (DeclareItemWeights)
// rather than through the field, so every load these tests read was written by
// the same recompute a map load reaches.
func mustWeighedWorld(t *testing.T, ents []Entity, stock []Stock, weights []ItemWeight) *World {
	t.Helper()
	w, err := NewStockedWorld(1, wtBounds, ModeCanonical, Terrain{}, ents, nil, Relations{}, nil, stock)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	if err := w.DeclareItemWeights(weights); err != nil {
		t.Fatalf("DeclareItemWeights: %v", err)
	}
	return w
}

// loadOfEntity reads the load off the world's own entity list, which is what a
// consumer sees.
func loadOfEntity(t *testing.T, w *World, id EntityID) int32 {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e.Load
		}
	}
	t.Fatalf("the world holds no entity %d", id)
	return 0
}

// wtForm marshals a world or fails the test.
func wtForm(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

func TestWhatIsWornCountsInFullAndWhatIsCarriedCountsHalf(t *testing.T) {
	var worn [EquipSlots]uint16
	worn[0], worn[3], worn[EquipSlots-1] = 0x1001, 0x1002, 0x1003
	w := mustWeighedWorld(t,
		[]Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{0x2001, 0x2001, 0x2001, 0x2002, 0x2002}, Equipped: worn}},
		[]ItemWeight{
			{Code: 0x1001, Weight: 100}, {Code: 0x1002, Weight: 30}, {Code: 0x1003, Weight: 7},
			{Code: 0x2001, Weight: 40}, {Code: 0x2002, Weight: 11},
		})

	if got := loadOfEntity(t, w, 1); got != 208 {
		t.Errorf("the load is %d, want 208", got)
	}
}

// TestTheHalvedContainerSumTruncatesTowardZero is the divisor's own
// rounding. The original's CDQ/SUB/SAR pair truncates toward zero, so an odd
// positive sum loses its half and an odd negative sum loses its half in the
// same direction. A floored halving would read -3 for the negative case.
//
// A negative weight is a state the shipped tables can produce: the Weapons
// collection carries a row whose price and weight columns are both -1.
func TestTheHalvedContainerSumTruncatesTowardZero(t *testing.T) {
	for _, tc := range []struct {
		name   string
		weight int32
		want   int32
	}{
		{"an odd positive sum", 5, 2},
		{"an odd negative sum", -5, -2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := mustWeighedWorld(t,
				[]Entity{{ID: 1, X: 1, Y: 1}},
				[]Stock{{ID: 1, Items: []uint16{0x2001}}},
				[]ItemWeight{{Code: 0x2001, Weight: tc.weight}})
			if got := loadOfEntity(t, w, 1); got != tc.want {
				t.Errorf("a container summing to %d gives load %d, want %d", tc.weight, got, tc.want)
			}
		})
	}
}

func TestAContainerAtOrPastSaturationAssignsAFlatLoad(t *testing.T) {
	var worn [EquipSlots]uint16
	worn[0] = 0x1001
	for _, tc := range []struct {
		name   string
		weight int32
		want   int32
	}{
		{"exactly at the threshold", 64000, 32000},
		{"one below the threshold", 63999, 32499},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := mustWeighedWorld(t,
				[]Entity{{ID: 1, X: 1, Y: 1}},
				[]Stock{{ID: 1, Items: []uint16{0x2001}, Equipped: worn}},
				[]ItemWeight{{Code: 0x1001, Weight: 500}, {Code: 0x2001, Weight: tc.weight}})
			if got := loadOfEntity(t, w, 1); got != tc.want {
				t.Errorf("a container summing to %d gives load %d, want %d", tc.weight, got, tc.want)
			}
		})
	}
}

func TestOnlyALoadAtOrAboveCapacityCostsSpeed(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		base, load, capacity int32
		want                 int32
	}{
		{"one unit below capacity is free", 20, 300, 301, 20},
		{"exactly at capacity costs one", 20, 301, 301, 19},
		{"twice capacity costs two", 20, 602, 301, 18},
		{"the quotient truncates", 20, 900, 301, 18},
		{"the floor stops the fall at six", 20, 6020, 301, 6},
		{"the floor raises a slow overloaded actor", 3, 301, 301, 6},
		{"no capacity is no penalty", 20, 100000, 0, 20},
		{"an unrated actor stays unrated", 0, 100000, 301, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := humanSpeedWord(tc.base, 0, tc.load, tc.capacity); got != tc.want {
				t.Errorf("base %d, load %d, capacity %d gives %d, want %d",
					tc.base, tc.load, tc.capacity, got, tc.want)
			}
		})
	}
}

// TestAnOverloadedActorStepsSlower is the penalty observed through a production
// consumer rather than at its own function. StepRate is the movement law's own
// door and reads the effective speed through moverSpeed. Two actors identical
// in every field, one carrying enough to pass his capacity, must get different
// rates for the same step.
//
// The rate itself is not asserted against a formula here: the step law's rate
// is terrain and domain arithmetic this test has nothing to say about. What is
// asserted is that the loaded actor's rate is strictly worse, which a build
// that computed the load and never consumed it would fail.
func TestAnOverloadedActorStepsSlower(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10, Speed: 20, Capacity: 301, Humanoid: true},
		{ID: 2, X: 1, Y: 1, HP: 10, MaxHP: 10, Speed: 20, Capacity: 301, Humanoid: true},
	}
	// Entity 2 carries 1000 units, halved to 500, which is past 301.
	w := mustWeighedWorld(t, ents,
		[]Stock{{ID: 2, Items: []uint16{0x2001}}},
		[]ItemWeight{{Code: 0x2001, Weight: 1000}})

	if got := loadOfEntity(t, w, 1); got != 0 {
		t.Fatalf("the unloaded actor's load is %d, want 0", got)
	}
	if got := loadOfEntity(t, w, 2); got != 500 {
		t.Fatalf("the loaded actor's load is %d, want 500", got)
	}

	freeRate, freeTransit, _, ok := w.StepRate(1, 2, 1)
	if !ok {
		t.Fatal("StepRate refused the unloaded actor")
	}
	loadedRate, loadedTransit, _, ok := w.StepRate(2, 2, 1)
	if !ok {
		t.Fatal("StepRate refused the loaded actor")
	}
	if loadedRate >= freeRate {
		t.Errorf("the loaded actor's rate is %d and the unloaded actor's is %d: the penalty did not reach the step law",
			loadedRate, freeRate)
	}
	if loadedTransit <= freeTransit {
		t.Errorf("the loaded actor crosses in %d and the unloaded actor in %d", loadedTransit, freeTransit)
	}
}

func TestACodeNoTableNamesWeighsNothing(t *testing.T) {
	w := mustWeighedWorld(t,
		[]Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{0x2001, 0x9999}}},
		[]ItemWeight{{Code: 0x2001, Weight: 40}})

	if got := loadOfEntity(t, w, 1); got != 20 {
		t.Errorf("the load is %d, want 20: only the declared code weighs", got)
	}
	if got, ok := w.CarriedStacks(1); !ok || len(got) != 2 {
		t.Errorf("the container holds %v, want both codes", got)
	}
}

// TestDeclaringOneCodeAtTwoWeightsIsRefusedAndChangesNothing is
// normaliseItemWeights' disagreement rule asserted at the door: the table and
// every load must be exactly what they were before the refused call.
func TestDeclaringOneCodeAtTwoWeightsIsRefusedAndChangesNothing(t *testing.T) {
	w := mustWeighedWorld(t,
		[]Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{0x2001, 0x2001}}},
		[]ItemWeight{{Code: 0x2001, Weight: 40}})
	before := loadOfEntity(t, w, 1)
	if before != 40 {
		t.Fatalf("the load starts at %d, want 40", before)
	}

	if err := w.DeclareItemWeights([]ItemWeight{{Code: 0x2001, Weight: 41}}); err == nil {
		t.Fatal("declaring code 0x2001 at a second weight was accepted")
	}
	if got := w.ItemWeights(); len(got) != 1 || got[0].Weight != 40 {
		t.Errorf("the refused call left the table as %v", got)
	}
	if got := loadOfEntity(t, w, 1); got != before {
		t.Errorf("the refused call moved the load to %d", got)
	}

	// The same code at the same weight is not a disagreement, and a second
	// producer's list is merged rather than replacing the first's.
	if err := w.DeclareItemWeights([]ItemWeight{{Code: 0x2001, Weight: 40}, {Code: 0x2002, Weight: 6}}); err != nil {
		t.Fatalf("an additive declaration was refused: %v", err)
	}
	if got := w.ItemWeights(); len(got) != 2 || got[0].Code != 0x2001 || got[1].Code != 0x2002 {
		t.Errorf("the merged table is %v, want both codes in code order", got)
	}
}

// TestReplaceStockRecomputesTheLoad is the one runtime door that replaces both
// terms at once. A build that wrote the containers and left the field naming
// the old contents would pass every other test in this file.
func TestReplaceStockRecomputesTheLoad(t *testing.T) {
	w := mustWeighedWorld(t,
		[]Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{0x2001, 0x2001}}},
		[]ItemWeight{{Code: 0x2001, Weight: 40}, {Code: 0x1001, Weight: 9}})
	if got := loadOfEntity(t, w, 1); got != 40 {
		t.Fatalf("the load starts at %d, want 40", got)
	}

	var worn [EquipSlots]uint16
	worn[0] = 0x1001
	if !w.ReplaceStock(Stock{ID: 1, Items: []uint16{0x2001}, Equipped: worn}) {
		t.Fatal("ReplaceStock refused the world's own entity")
	}
	if got := loadOfEntity(t, w, 1); got != 29 {
		t.Errorf("after the replacement the load is %d, want 29", got)
	}
}

// TestTheLoadTheCapacityAndTheWeightTableAreCanonicalState is the byte form's
// half. Four worlds: a base, one whose actor carries more, one whose actor has
// a different capacity, and one whose weight table names a different weight for
// the same code. All four must differ pairwise in form and in digest, and each
// must survive a round trip.
//
// The weight-table case is the one worth stating: nothing about the container
// differs between it and the base, so a build that carried the load but not the
// table it was derived from would produce two worlds that marshal identically
// and then reload with different loads.
func TestTheLoadTheCapacityAndTheWeightTableAreCanonicalState(t *testing.T) {
	worlds := []struct {
		what string
		w    *World
	}{
		{"the base world", mustWeighedWorld(t,
			[]Entity{{ID: 1, X: 1, Y: 1, Capacity: 301}},
			[]Stock{{ID: 1, Items: []uint16{0x2001}}},
			[]ItemWeight{{Code: 0x2001, Weight: 40}})},
		{"one more of the same item carried", mustWeighedWorld(t,
			[]Entity{{ID: 1, X: 1, Y: 1, Capacity: 301}},
			[]Stock{{ID: 1, Items: []uint16{0x2001, 0x2001}}},
			[]ItemWeight{{Code: 0x2001, Weight: 40}})},
		{"a different capacity", mustWeighedWorld(t,
			[]Entity{{ID: 1, X: 1, Y: 1, Capacity: 401}},
			[]Stock{{ID: 1, Items: []uint16{0x2001}}},
			[]ItemWeight{{Code: 0x2001, Weight: 40}})},
		{"the same item declared at another weight", mustWeighedWorld(t,
			[]Entity{{ID: 1, X: 1, Y: 1, Capacity: 301}},
			[]Stock{{ID: 1, Items: []uint16{0x2001}}},
			[]ItemWeight{{Code: 0x2001, Weight: 41}})},
	}

	for i := range worlds {
		for j := i + 1; j < len(worlds); j++ {
			a, b := worlds[i], worlds[j]
			if bytes.Equal(wtForm(t, a.w), wtForm(t, b.w)) {
				t.Errorf("%s and %s marshal identically", a.what, b.what)
			}
			if a.w.Hash() == b.w.Hash() {
				t.Errorf("%s and %s hash %#016x", a.what, b.what, a.w.Hash())
			}
		}
	}
	for _, c := range worlds {
		var back World
		if err := back.UnmarshalBinary(wtForm(t, c.w)); err != nil {
			t.Fatalf("%s: UnmarshalBinary: %v", c.what, err)
		}
		if got := wtForm(t, &back); !bytes.Equal(got, wtForm(t, c.w)) {
			t.Errorf("%s does not survive the round trip", c.what)
		}
		if back.Hash() != c.w.Hash() {
			t.Errorf("%s round-trips to digest %#016x, want %#016x", c.what, back.Hash(), c.w.Hash())
		}
		got, want := back.ItemWeights(), c.w.ItemWeights()
		if len(got) != len(want) || (len(got) > 0 && got[0] != want[0]) {
			t.Errorf("%s round-trips its weight table as %v, want %v", c.what, got, want)
		}
	}
}

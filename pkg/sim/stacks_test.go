package sim

// A container is a list of STACKS (0138 T1): an element is one item code
// together with the count of it held there, and two elements naming one code
// cannot coexist. Every act that puts an item into a container merges it —
// construction, a decode, a take, a give-all, and an item displaced out of an
// equipment slot — and every reader that answers in CODES answers the flat
// expansion, exactly as it did before this story.
//
// WHAT IS NOT HERE: the pinned digests. AC-9's clause that "every digest this
// package pins is unchanged" is witnessed by those pins THEMSELVES — pinDigest
// (hash_test.go), rtfDigest (routeform_test.go) and every pre-story digest in
// binary_test.go's peel chain — none of which this story touched and all of
// which are green. A copy of any of them here would be a second place to keep
// them, and re-taking one to make this file agree with the encoder is exactly
// what a pin exists to refuse.

import (
	"testing"
)

// The codes every fixture below is built out of. stPotion is the one that
// stacks in most of them; stOther, stThird and stWorn are distinct from it and
// from each other, so a case that folded the wrong pair would show as a
// changed ORDER and not merely as a changed count.
const (
	stPotion uint16 = 0x0e06
	stOther  uint16 = 0x0102
	stThird  uint16 = 0x0103
	stWorn   uint16 = 0x0104
)

// equalStacks compares two element lists for equal length, codes, counts and
// ORDER, treating nil and empty as equal — equalCodes' own rule (sackform_test.go),
// restated for an element. Order is compared because it is canonical state:
// element order decides the flat expansion, which decides the bytes of the form
// and the digest.
func equalStacks(a, b []ItemStack) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !StackStateEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------- AC-1, AC-2, AC-3

// TestAnAuthoredContainerFoldsAndReadsBackBothWays is AC-1, AC-2 and AC-3 —
// one actor authored with three of one code and one of another, read three
// ways. They are one test because they are one fixture measured at the two
// grains plus its own shape: splitting them would build the same world three
// times and could let the element view and the code view drift apart without
// either case saying so.
//
// The second code is authored BETWEEN two of the first, so first-seen order is
// told apart from sorted order and from last-seen order: [p, o, p, p] folds to
// [(p,3), (o,1)] and never to [(o,1), (p,3)].
func TestAnAuthoredContainerFoldsAndReadsBackBothWays(t *testing.T) {
	t.Parallel()

	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{stPotion, stOther, stPotion, stPotion}}})

	// AC-1: two elements, the first at count 3, in first-seen order.
	// AC-3: read as elements, the counts are 3 and 1.
	got, ok := w.CarriedStacks(1)
	if !ok {
		t.Fatal("CarriedStacks(1) answered not-ok for an entity this world holds")
	}
	want := []ItemStack{{Code: stPotion, Count: 3}, {Code: stOther, Count: 1}}
	if !equalStacks(got, want) {
		t.Errorf("CarriedStacks(1) = %+v, want %+v — one place per code, first-seen order, counts summed",
			got, want)
	}

	// AC-2: read as codes, four come back, the three equal ones adjacent and
	// first — the element's own place, expanded.
	codes, ok := w.Carried(1)
	if !ok {
		t.Fatal("Carried(1) answered not-ok for an entity this world holds")
	}
	if !equalCodes(codes, []uint16{stPotion, stPotion, stPotion, stOther}) {
		t.Errorf("Carried(1) = %#x, want three of %#x adjacent and first, then %#x",
			codes, stPotion, stOther)
	}

	// An id this world does not hold answers nothing at either grain.
	if got, ok := w.CarriedStacks(99); ok || got != nil {
		t.Errorf("CarriedStacks(99) = (%+v, %v), want (nil, false)", got, ok)
	}
}

func TestCarriedStacksHandsBackACopy(t *testing.T) {
	t.Parallel()

	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{stPotion, stPotion}}})

	first, _ := w.CarriedStacks(1)
	first[0].Count = 99
	first[0].Code = stOther

	second, _ := w.CarriedStacks(1)
	if !equalStacks(second, []ItemStack{{Code: stPotion, Count: 2}}) {
		t.Errorf("mutating one call's result reached the world: CarriedStacks(1) = %+v", second)
	}
}

// ---------------------------------------------------------------------- AC-4

// TestTakingASackMergesIntoTheTaker is AC-4: an actor holding one potion picks
// up a sack of two more and holds ONE element at count 3, not three places and
// not two. The sack's own order and its all-or-nothing outcome are
// carry_test.go's business and are unchanged; what this measures is the merge.
func TestTakingASackMergesIntoTheTaker(t *testing.T) {
	t.Parallel()

	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3}}, nil, Relations{},
		[]Sack{{X: 3, Y: 3, Items: []uint16{stPotion, stPotion}}},
		[]Stock{{ID: 1, Items: []uint16{stPotion}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	if err := w.TakeSack(1, 3, 3); err != nil {
		t.Fatalf("TakeSack: %v", err)
	}

	got, _ := w.CarriedStacks(1)
	if !equalStacks(got, []ItemStack{{Code: stPotion, Count: 3}}) {
		t.Errorf("CarriedStacks(1) = %+v, want one element of %#x at count 3", got, stPotion)
	}
	if codes, _ := w.Carried(1); len(codes) != 3 {
		t.Errorf("Carried(1) = %#x, want three units — a merge sums counts, it does not drop any", codes)
	}
}

// ---------------------------------------------------------------------- AC-5

func TestAFelledBodyDropsUnitsAndNotElements(t *testing.T) {
	t.Parallel()

	var worn [EquipSlots]uint16
	worn[slotWeapon] = stWorn
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 4, Y: 6}},
		[]Stock{{ID: 1, Items: []uint16{stPotion, stPotion, stPotion}, Equipped: worn}})

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	got := w.Sacks()
	if len(got) != 1 {
		t.Fatalf("got %d sack(s), want 1: %+v", len(got), got)
	}
	want := []uint16{stPotion, stPotion, stPotion, stWorn}
	if !equalCodes(got[0].Items, want) {
		t.Errorf("the sack holds %#x, want %#x — three units of the stack, then the weapon slot",
			got[0].Items, want)
	}
	if left, _ := w.CarriedStacks(1); len(left) != 0 {
		t.Errorf("CarriedStacks(1) = %+v after the drop, want nothing — the whole container poured", left)
	}
}

// ---------------------------------------------------------------------- AC-6

func TestGiveAllMergesIntoTheReceiver(t *testing.T) {
	t.Parallel()

	const giver, receiver EntityID = 1, 2
	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: giver, HasUnit: true, Unit2: receiver, HasUnit2: true},
	}, 0)
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: giver, X: 1, Y: 1}, {ID: receiver, X: 2, Y: 2}}, s, Relations{}, nil,
		[]Stock{
			{ID: giver, Items: []uint16{stPotion, stPotion}},
			{ID: receiver, Items: []uint16{stPotion, stPotion}},
		})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	runPass(t, w)

	got, _ := w.CarriedStacks(receiver)
	if !equalStacks(got, []ItemStack{{Code: stPotion, Count: 4}}) {
		t.Errorf("CarriedStacks(receiver) = %+v, want one element of %#x at count 4", got, stPotion)
	}
	if left, _ := w.CarriedStacks(giver); len(left) != 0 {
		t.Errorf("CarriedStacks(giver) = %+v, want nothing — the whole container moved", left)
	}
}

// ------------------------------------------------------------------ AC-7, AC-8

func TestEquipTakesOneUnitOffAnElement(t *testing.T) {
	t.Parallel()

	t.Run("AC-7 a count above 1 into an empty slot", func(t *testing.T) {
		t.Parallel()

		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
			[]Stock{{ID: 1, Items: []uint16{stPotion, stPotion, stPotion}}})

		Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 1}})

		if got, _ := w.Equipped(1); got[0] != stPotion {
			t.Errorf("slot 1 = %#x, want %#x", got[0], stPotion)
		}
		got, _ := w.CarriedStacks(1)
		if !equalStacks(got, []ItemStack{{Code: stPotion, Count: 2}}) {
			t.Errorf("CarriedStacks(1) = %+v, want one element of %#x at count 2 — one unit left, in its own place",
				got, stPotion)
		}
	})

	t.Run("a count above 1 into an occupied slot displaces through the merge", func(t *testing.T) {
		t.Parallel()

		t.Run("a code the container does not hold takes the tail", func(t *testing.T) {
			t.Parallel()

			w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
				[]Stock{{ID: 1, Items: []uint16{stPotion, stPotion, stPotion}}})
			w.equipment[0][0] = PlainItem(stWorn)

			Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 1}})

			if got, _ := w.Equipped(1); got[0] != stPotion {
				t.Errorf("slot 1 = %#x, want %#x", got[0], stPotion)
			}
			got, _ := w.CarriedStacks(1)
			want := []ItemStack{{Code: stPotion, Count: 2}, {Code: stWorn, Count: 1}}
			if !equalStacks(got, want) {
				t.Errorf("CarriedStacks(1) = %+v, want %+v", got, want)
			}
		})

		t.Run("a code the container already holds joins that element", func(t *testing.T) {
			t.Parallel()

			w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
				[]Stock{{ID: 1, Items: []uint16{stPotion, stPotion, stPotion, stWorn}}})
			w.equipment[0][0] = PlainItem(stWorn)

			Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 1}})

			if got, _ := w.Equipped(1); got[0] != stPotion {
				t.Errorf("slot 1 = %#x, want %#x", got[0], stPotion)
			}
			got, _ := w.CarriedStacks(1)
			want := []ItemStack{{Code: stPotion, Count: 2}, {Code: stWorn, Count: 2}}
			if !equalStacks(got, want) {
				t.Errorf("CarriedStacks(1) = %+v, want %+v — the displaced unit joins the element "+
					"already holding its code, and takes no place of its own", got, want)
			}
		})
	})

	t.Run("AC-8 a count of 1 into an occupied slot", func(t *testing.T) {
		t.Parallel()

		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
			[]Stock{{ID: 1, Items: []uint16{stOther}}})
		// equip_test.go's own one direct write, for the one shape no command
		// built here can produce: AC-8 needs slot 1 already occupied BEFORE the
		// equip that displaces it, and nothing exported writes a slot except
		// the command under test.
		w.equipment[0][0] = PlainItem(stWorn)

		Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 1}})

		if got, _ := w.Equipped(1); got[0] != stOther {
			t.Errorf("slot 1 = %#x, want %#x", got[0], stOther)
		}
		got, _ := w.CarriedStacks(1)
		if !equalStacks(got, []ItemStack{{Code: stWorn, Count: 1}}) {
			t.Errorf("CarriedStacks(1) = %+v, want the displaced %#x at count 1, in the vacated place",
				got, stWorn)
		}
	})
}

// TestEquipMergesADisplacedCodeIntoTheElementThatHoldsIt is D-7's own case,
// and it is here rather than in equip_test.go because it is the behaviour
// 0112's AC-3 and AC-4 could not see: BOTH of those use distinct codes, so
// they stay green on their own terms while saying nothing about a displaced
// code the container already holds.
//
// An actor holds [a, b, c] at one unit apiece and is already WEARING c. He
// equips a, so c is displaced back into the container — and the container
// already holds c. Before this story the result was the code list [c, b, c],
// two separate places naming one code; now the displaced unit joins the
// element that holds it and the container is [(c,2), (b,1)], whose expansion
// is c, c, b.
//
// THE ORDER IS THE POINT, not merely the count: the merge keeps the FIRST
// place the code occupies, which here is the place the equipped element just
// vacated, so b moves up rather than staying at index 1.
func TestEquipMergesADisplacedCodeIntoTheElementThatHoldsIt(t *testing.T) {
	t.Parallel()

	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{stOther, stThird, stWorn}}})
	w.equipment[0][0] = PlainItem(stWorn) // AC-8's own direct write, for the same reason

	Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 1}})

	if got, _ := w.Equipped(1); got[0] != stOther {
		t.Errorf("slot 1 = %#x, want the equipped %#x", got[0], stOther)
	}
	got, _ := w.CarriedStacks(1)
	want := []ItemStack{{Code: stWorn, Count: 2}, {Code: stThird, Count: 1}}
	if !equalStacks(got, want) {
		t.Errorf("CarriedStacks(1) = %+v, want %+v — the displaced code joins the element holding it, "+
			"at that element's own place", got, want)
	}
	if codes, _ := w.Carried(1); !equalCodes(codes, []uint16{stWorn, stWorn, stThird}) {
		t.Errorf("Carried(1) = %#x, want %#x — and NOT the [c b c] the pre-0138 tree left",
			codes, []uint16{stWorn, stWorn, stThird})
	}
}

// ---------------------------------------------------------------------- AC-9

// TestAWorldHoldingACountedElementRoundTripsByteIdentically is AC-9: a world
// holding an element of count 3 marshals, unmarshals and re-marshals to the
// same bytes, hashes the same either side, and comes back holding the same
// elements at the same counts. The count is canonical state that survives
// the form even though the form gained no field for it: it is written as the
// expansion and folded back on the way in.
//
// The fixture holds a second entity carrying two DISTINCT codes and a third
// carrying nothing, so what round-trips is a carry section with a counted
// record, a flat record and an empty one rather than only the interesting one.
func TestAWorldHoldingACountedElementRoundTripsByteIdentically(t *testing.T) {
	t.Parallel()

	w := mustStockedWorld(t, 1,
		[]Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}, {ID: 3, X: 3, Y: 3}},
		[]Stock{
			{ID: 1, Items: []uint16{stPotion, stPotion, stPotion}},
			{ID: 2, Items: []uint16{stOther, stThird}},
		})

	first := mustMarshal(t, w)
	var back World
	if err := back.UnmarshalBinary(first); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	second := mustMarshal(t, &back)
	if string(first) != string(second) {
		t.Fatal("a world holding a counted element does not round-trip through its byte form")
	}
	if back.Hash() != w.Hash() {
		t.Fatalf("the round-tripped world hashes %#016x, the original %#016x", back.Hash(), w.Hash())
	}

	if got, _ := back.CarriedStacks(1); !equalStacks(got, []ItemStack{{Code: stPotion, Count: 3}}) {
		t.Errorf("entity 1 decoded as %+v, want one element of %#x at count 3", got, stPotion)
	}
	want2 := []ItemStack{{Code: stOther, Count: 1}, {Code: stThird, Count: 1}}
	if got, _ := back.CarriedStacks(2); !equalStacks(got, want2) {
		t.Errorf("entity 2 decoded as %+v, want %+v", got, want2)
	}
	if got, _ := back.CarriedStacks(3); len(got) != 0 {
		t.Errorf("entity 3 decoded as %+v, want an empty container", got)
	}
}

func TestACountedElementIsTwoWorldsFromTheSameElementAtAnotherCount(t *testing.T) {
	t.Parallel()

	two := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{stPotion, stPotion}}})
	three := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{stPotion, stPotion, stPotion}}})

	if two.Hash() == three.Hash() {
		t.Errorf("one element at count 2 and the same element at count 3 both hash %#016x", two.Hash())
	}
	if string(mustMarshal(t, two)) == string(mustMarshal(t, three)) {
		t.Error("one element at count 2 and the same element at count 3 marshal alike")
	}
}

// --------------------------------------------------------------------- AC-10

// stOneEntityCarryCountAt is the offset of the single-entity carry record's own
// uint32 code count in a 4x4 world's byte form: the header, three 16-cell
// planes, one entity record, one empty route count, one empty group count and
// one empty sack count. It is written out from the contract exactly as
// TestUnmarshalRefusesACarriedCodeOfZeroNamingTheEntity writes its own out,
// not read back off the encoder, and each case below checks the bytes it lands
// on before trusting it.
const stOneEntityCarryCountAt = headerLen + 3*16 + entityLen + routeCountLen + groupCountLen + sackCountLen

// TestARecordNamingOneCodeSeventyThousandTimesFoldsWithoutLoss is AC-10's
// second half, and it is the case that fixes the count's WIDTH (D-1). The carry
// record's length is a uint32 bounded only by the buffer and normaliseHoldings
// takes a Stock.Items of any length, so a container of 70 000 of one code is a
// payload the PREVIOUS build both writes and reads back. A uint16 count would
// fold it to 70000 - 65536 = 4 464 — silent loss, and a decode that no longer
// inverted the encode, which is the whole argument for leaving formatVersion
// where it is (D-5).
//
// It witnesses the encoder's own preallocation too, at no extra cost: encode
// sizes the carry section from Σ Count and writes the expansion, so a
// preallocation still counting ELEMENTS would reserve 2 bytes where 140 000 are
// written and this marshal would fail on the spot rather than subtly.
func TestARecordNamingOneCodeSeventyThousandTimesFoldsWithoutLoss(t *testing.T) {
	t.Parallel()

	const units uint32 = 70000

	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}})
	w.carried[0] = []ItemStack{{Code: stPotion, Count: units}}
	form := mustMarshal(t, w)

	// The payload really does name the code that many times: the record's own
	// declared count, read out of the bytes rather than taken on trust.
	if got := leU32(form, stOneEntityCarryCountAt); got != units {
		t.Fatalf("the carry record declares %d code(s), want %d", got, units)
	}

	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	got, _ := back.CarriedStacks(1)
	if !equalStacks(got, []ItemStack{{Code: stPotion, Count: units}}) {
		t.Errorf("the folded record is %+v, want one element of %#x at count %d — "+
			"a 16-bit count would have left %d here", got, stPotion, units, units-65536)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round-tripped world hashes %#016x, the original %#016x — "+
			"decode did not invert encode at this length", back.Hash(), w.Hash())
	}
}

// leU32 reads a little-endian uint32 out of a form at off. It is spelled here
// rather than reached for through encoding/binary so that a test reading the
// bytes back never shares an instrument with the encoder that wrote them —
// fnv1a's own reason (hash_test.go).
func leU32(b []byte, off int) uint32 {
	return uint32(b[off]) | uint32(b[off+1])<<8 | uint32(b[off+2])<<16 | uint32(b[off+3])<<24
}

func TestFoldingIsIdempotentAndInvertsExpansion(t *testing.T) {
	t.Parallel()

	cases := [][]ItemStack{
		nil,
		{{Code: stPotion, Count: 1}},
		{{Code: stPotion, Count: 3}},
		{{Code: stPotion, Count: 3}, {Code: stOther, Count: 1}},
		{{Code: stOther, Count: 1}, {Code: stPotion, Count: 70000}, {Code: stThird, Count: 2}},
	}
	for _, c := range cases {
		if got := foldContainer(c); !equalStacks(got, c) {
			t.Errorf("foldContainer(%+v) = %+v, want it unchanged — P-1", c, got)
		}
		if got := foldContainer(appendUnits(nil, expandContainer(c))); !equalStacks(got, c) {
			t.Errorf("folding the expansion of %+v gave %+v, want the container back — P-3", c, got)
		}
		if got := containerUnits(c); got != uint32(len(expandContainer(c))) {
			t.Errorf("containerUnits(%+v) = %d, but its expansion is %d code(s) long — "+
				"the encoder sizes with one and writes the other", c, got, len(expandContainer(c)))
		}
	}
}

func TestFoldingMergesIntoTheFirstPlaceAndDropsACountOfZero(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   []ItemStack
		want []ItemStack
	}{
		{"two runs of one code join at the first place",
			[]ItemStack{{Code: stPotion, Count: 1}, {Code: stOther, Count: 2}, {Code: stPotion, Count: 4}},
			[]ItemStack{{Code: stPotion, Count: 5}, {Code: stOther, Count: 2}}},
		{"a count of zero is dropped and takes no place",
			[]ItemStack{{Code: stPotion, Count: 1}, {Code: stOther, Count: 0}, {Code: stThird, Count: 2}},
			[]ItemStack{{Code: stPotion, Count: 1}, {Code: stThird, Count: 2}}},
		{"a code held only at zero disappears entirely",
			[]ItemStack{{Code: stOther, Count: 0}},
			nil},
		{"counts sum with no limit test, and wrap rather than saturate",
			[]ItemStack{{Code: stPotion, Count: 1 << 31}, {Code: stPotion, Count: 1 << 31},
				{Code: stPotion, Count: 7}},
			[]ItemStack{{Code: stPotion, Count: 7}}}, // 2^31 + 2^31 wraps to 0, then + 7
	}
	for _, tc := range cases {
		if got := foldContainer(tc.in); !equalStacks(got, tc.want) {
			t.Errorf("%s: foldContainer(%+v) = %+v, want %+v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestEquippingTheLastUnitRemovesTheElement(t *testing.T) {
	t.Parallel()

	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{stOther, stThird}}})

	Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 1}})

	got, _ := w.CarriedStacks(1)
	if !equalStacks(got, []ItemStack{{Code: stThird, Count: 1}}) {
		t.Errorf("CarriedStacks(1) = %+v, want only %#x left — an element at 0 does not exist",
			got, stThird)
	}
}

func TestUnitsAreConservedAcrossTheActsThatMerge(t *testing.T) {
	t.Parallel()

	const giver, taker EntityID = 1, 2
	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: giver, HasUnit: true, Unit2: taker, HasUnit2: true},
	}, 0)
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: giver, X: 1, Y: 1}, {ID: taker, X: 3, Y: 3}}, s, Relations{},
		[]Sack{{X: 3, Y: 3, Items: []uint16{stPotion, stPotion}}},
		[]Stock{
			{ID: giver, Items: []uint16{stPotion, stPotion}}, // 2 units
			{ID: taker, Items: []uint16{stPotion, stPotion}}, // 2 units
		})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	// 4 authored + 2 in the sack.
	const total = 6

	if err := w.TakeSack(taker, 3, 3); err != nil {
		t.Fatalf("TakeSack: %v", err)
	}
	runPass(t, w)

	held, _ := w.Carried(taker)
	if len(held) != total {
		t.Fatalf("the taker holds %d unit(s) after the take and the pour, want %d", len(held), total)
	}
	if left, _ := w.Carried(giver); len(left) != 0 {
		t.Errorf("the giver holds %#x, want nothing", left)
	}

	// And the equip moves exactly one unit out of the container and into a
	// slot: the container is one shorter and the slot holds the code.
	Step(w, []Command{{Kind: KindEquip, Entity: taker, X: 0, Y: 1}})

	after, _ := w.Carried(taker)
	if len(after) != total-1 {
		t.Errorf("the taker holds %d unit(s) after the equip, want %d — one unit moves, not an element",
			len(after), total-1)
	}
	if eq, _ := w.Equipped(taker); eq[0] != stPotion {
		t.Errorf("slot 1 = %#x, want the one unit the equip moved (%#x)", eq[0], stPotion)
	}
}

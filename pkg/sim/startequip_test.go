package sim

// The starting-equipment hotfix, this package's half: a Stock states what an
// actor WEARS as well as what he carries, a body drops every slot it wears
// into its container before the container becomes a sack (ITEM-DEATH-012,
// widened from the two base-actor slots to all twelve by 0132), and Stock()
// reads both halves back so a rebuild cannot hold one and drop the other.
//
// Every fixture dies through Step's KindTerminalKill arm, corpseloot_test.go's own
// reason: what is under test is the drop, not the arm that fells.

import (
	"bytes"
	"testing"
)

// seBounds is the extent every fixture here is built against — ddBounds'
// own size, named separately so a case that needs a different one does not
// have to move that file's variable.
var seBounds = Bounds{Width: 10, Height: 10}

// mustWornWorld is mustStockedWorld with a loadout: one entity at (4,6)
// wearing worn and carrying items.
func mustWornWorld(t *testing.T, items []uint16, worn [EquipSlots]uint16) *World {
	t.Helper()
	w, err := NewStockedWorld(1, seBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 4, Y: 6}}, nil, Relations{}, nil,
		[]Stock{{ID: 1, Items: items, Equipped: worn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

// TestAnActorWearingSomethingAndCarryingNothingStillLeavesASack is the
// owner's first reported defect, at its narrowest: before this hotfix the
// death path read the container alone, so a unit whose whole kit was in his
// hand left the sack list untouched and nothing appeared where he fell.
func TestAnActorWearingSomethingAndCarryingNothingStillLeavesASack(t *testing.T) {
	w := mustWornWorld(t, nil, [EquipSlots]uint16{0x0107})

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	got := w.Sacks()
	if len(got) != 1 {
		t.Fatalf("got %d sack(s), want 1: %+v", len(got), got)
	}
	if got[0].X != 4 || got[0].Y != 6 || !equalCodes(got[0].Items, []uint16{0x0107}) {
		t.Errorf("the sack is %+v, want (4,6) holding [0x107]", got[0])
	}
	eq, ok := w.Equipped(1)
	if !ok {
		t.Fatal("Equipped(1) answered not-ok for an entity this world still holds")
	}
	if eq[0] != 0 {
		t.Errorf("slot 1 = 0x%04x after the drop, want empty — a dropped weapon is not still worn", eq[0])
	}
}

func TestAKillDropsCarriedThenSlotTwoThenSlotOneThenArmourAscendingAndEmptiesTheBody(t *testing.T) {
	worn := [EquipSlots]uint16{0x0111, 0x0222, 0x0033, 0, 0, 0, 0x0066, 0, 0, 0, 0, 0x00cc}
	w := mustWornWorld(t, []uint16{0x0301, 0x0302}, worn)

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	got := w.Sacks()
	if len(got) != 1 {
		t.Fatalf("got %d sack(s), want 1: %+v", len(got), got)
	}
	want := []uint16{0x0301, 0x0302, 0x0222, 0x0111, 0x0033, 0x0066, 0x00cc}
	if !equalCodes(got[0].Items, want) {
		t.Errorf("the sack holds %v, want %v — the container's own order, then slot 2, then slot 1, "+
			"then the armour ascending", got[0].Items, want)
	}

	eq, ok := w.Equipped(1)
	if !ok {
		t.Fatal("Equipped(1) answered not-ok for an entity this world still holds")
	}
	if eq != ([EquipSlots]uint16{}) {
		t.Errorf("Equipped(1) = %v after the drop, want the zero array — AC-2", eq)
	}
	carried, ok := w.Carried(1)
	if !ok {
		t.Fatal("Carried(1) answered not-ok for an entity this world still holds")
	}
	if len(carried) != 0 {
		t.Errorf("Carried(1) = %v after the drop, want nothing — AC-2", carried)
	}
}

// TestTheTenArmourSlotsAreDroppedAscending is the other half of
// ITEM-DEATH-012, TURNED OVER BY 0132: before that story only the two
// BASE-ACTOR slots were unequipped on death and the ten at `actor+0x198` —
// a humanoid's armour — stayed on the corpse, so a body whose only kit was
// armour left no sack (this test's own old name and old expectation). The
// four instructions after the weapon arm, read for 0132, dispatch a call
// that empties those ten too, so an armour-only body now leaves exactly one
// sack, holding those ten codes in ascending slot order (0132 AC-3), and its
// Equipped record reads back as the zero array (0132 AC-2). Same fixture as
// before the turnover; only the expectation is the opposite of what it was.
func TestTheTenArmourSlotsAreDroppedAscending(t *testing.T) {
	var worn [EquipSlots]uint16
	for i := 2; i < EquipSlots; i++ {
		worn[i] = uint16(0x0400 + i)
	}
	w := mustWornWorld(t, nil, worn)

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	got := w.Sacks()
	if len(got) != 1 {
		t.Fatalf("got %d sack(s), want 1: %+v", len(got), got)
	}
	want := worn[2:]
	if got[0].X != 4 || got[0].Y != 6 || !equalCodes(got[0].Items, want) {
		t.Errorf("the sack is %+v, want (4,6) holding %v — the ten armour codes, ascending", got[0], want)
	}
	eq, _ := w.Equipped(1)
	if eq != ([EquipSlots]uint16{}) {
		t.Errorf("Equipped(1) = %v after the death, want the zero array — the armour is on the ground now", eq)
	}
}

func TestADeathOffTheBoundsLeavesAllTwelveSlotsAndTheContainerOnTheCorpse(t *testing.T) {
	small := Bounds{Width: 5, Height: 5}
	var worn [EquipSlots]uint16
	for i := range worn {
		worn[i] = uint16(0x0100 + i)
	}
	items := []uint16{0x0301, 0x0302}
	w, err := NewStockedWorld(1, small, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: small.Width, Y: 2}}, nil, Relations{}, nil,
		[]Stock{{ID: 1, Items: items, Equipped: worn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	if got := w.Sacks(); len(got) != 0 {
		t.Errorf("Sacks() = %+v after a corpse off the map, want none", got)
	}
	if eq, _ := w.Equipped(1); eq != worn {
		t.Errorf("Equipped(1) = %v after a corpse off the map, want %v — refused, not moved, all twelve", eq, worn)
	}
	if got, _ := w.Carried(1); !equalCodes(got, items) {
		t.Errorf("Carried(1) = %v after a corpse off the map, want %v — refused, not moved", got, items)
	}
}

// TestAKillDropsOnlySlotsOneFiveAndTwelveAsThreeCodesWithNoZero is AC-5: an
// entity with slots 1, 5 and 12 filled and every other slot — 2, 3, 4, 6
// through 11 — empty leaves a sack holding exactly three codes, slot 1
// then slot 5 then slot 12, with no zero among them.
func TestAKillDropsOnlySlotsOneFiveAndTwelveAsThreeCodesWithNoZero(t *testing.T) {
	var worn [EquipSlots]uint16
	worn[0] = 0x0511  // slot 1
	worn[4] = 0x0515  // slot 5
	worn[11] = 0x0512 // slot 12
	w := mustWornWorld(t, nil, worn)

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	got := w.Sacks()
	if len(got) != 1 {
		t.Fatalf("got %d sack(s), want 1: %+v", len(got), got)
	}
	want := []uint16{0x0511, 0x0515, 0x0512}
	if !equalCodes(got[0].Items, want) {
		t.Errorf("the sack holds %v, want %v — slot 1, then slot 5, then slot 12, no zero among them",
			got[0].Items, want)
	}
	for _, c := range got[0].Items {
		if c == 0 {
			t.Errorf("the sack holds a zero code among %v — an empty slot must be passed over, not dropped", got[0].Items)
		}
	}
}

// TestStockReadsTheLoadoutBackForARebuild is the property pkg/mapload's two
// mission-start rebuilds stand on: feeding Stock() back into NewStockedWorld
// over the same entities reproduces the loadout as well as the container. It
// is asserted for an actor with NO container at all, because that is the
// entry Stock() used to skip entirely — the shape in which a rebuild would
// have silently disarmed every unit on the map.
func TestStockReadsTheLoadoutBackForARebuild(t *testing.T) {
	worn := [EquipSlots]uint16{0x0107}
	first := mustWornWorld(t, nil, worn)

	back := first.Stock()
	if len(back) != 1 {
		t.Fatalf("Stock() = %+v, want one entry for an actor who wears something and carries nothing", back)
	}
	if back[0].ID != 1 || back[0].Equipped != worn || len(back[0].Items) != 0 {
		t.Fatalf("Stock() = %+v, want {ID:1 Items:[] Equipped:%v}", back, worn)
	}

	again, err := NewStockedWorld(1, seBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 4, Y: 6}}, nil, Relations{}, nil, back)
	if err != nil {
		t.Fatalf("NewStockedWorld on the rebuild: %v", err)
	}
	if eq, _ := again.Equipped(1); eq != worn {
		t.Errorf("the rebuilt world's Equipped(1) = %v, want %v", eq, worn)
	}
}

// TestALoadoutSurvivesTheByteForm is the section 0124 built read at values
// it had never carried. Its own witness (binary_test.go) measures a world
// "equipping nothing" — every one of the twelve slots at the zero code —
// because until this hotfix no path could produce anything else. Every
// shipped mission now opens with that section non-zero, so the decoder is on
// a path it was never on, and a form that came back with an empty loadout
// would lose a saved actor's weapon in silence.
func TestALoadoutSurvivesTheByteForm(t *testing.T) {
	worn := [EquipSlots]uint16{0x0107, 0x0208, 0, 0, 0x0505}
	w := mustWornWorld(t, []uint16{0x0301}, worn)

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if eq, ok := back.Equipped(1); !ok || eq != worn {
		t.Errorf("the decoded world's Equipped(1) = %v (ok=%v), want %v", eq, ok, worn)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary on the decoded world: %v", err)
	}
	if !bytes.Equal(form, again) {
		t.Error("re-encoding the decoded world does not reproduce its own bytes")
	}
}

func TestTheFirstEquipDisplacesTheStartingWeaponIntoThePack(t *testing.T) {
	const starting, taken = uint16(0x0107), uint16(0x0114)
	w := mustWornWorld(t, []uint16{0x0301, taken}, [EquipSlots]uint16{starting})

	Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 1, Y: 1}})

	if eq, _ := w.Equipped(1); eq[0] != taken {
		t.Errorf("slot 1 = 0x%04x after the first equip, want 0x%04x", eq[0], taken)
	}
	got, _ := w.Carried(1)
	if !equalCodes(got, []uint16{0x0301, starting}) {
		t.Errorf("Carried(1) = %v after the first equip, want [0x301 0x107] — "+
			"the starting weapon back at the source index, not superseded", got)
	}
}

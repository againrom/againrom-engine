package sim

// The corpse drop (0123 T2, widened by 0132 T2): the container, and now
// every equipment slot too, becomes a ground sack when a body reaches the
// terminal -10 boundary, in decayPass, through the pourSack primitive
// (sack.go).
//
// Every fixture here dies through Step's KindTerminalKill arm, which is the debug
// tool this package uses for putting a unit at the terminal boundary without a
// combat roll (step.go) — nothing here exercises resolveBlow, because what
// is under test is the drop decayPass performs once a unit is terminally dead by
// whatever arm, not the arm itself.

import (
	"bytes"
	"testing"
)

// ddBounds is the small extent every fixture here is built against, large
// enough to hold every cell a case names.
var ddBounds = Bounds{Width: 10, Height: 10}

func TestFallenUnitKeepsItsLoadUntilTerminalDeath(t *testing.T) {
	w := mustStockedWorld(t, 1,
		[]Entity{{ID: 1, X: 4, Y: 6, HP: 1, MaxHP: 10}},
		[]Stock{{ID: 1, Items: []uint16{0x101}, Equipped: [EquipSlots]uint16{0x201}}})

	Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 1}})
	if got := w.Entities()[0].HP; got != 0 {
		t.Fatalf("downing blow left HP %d, want 0", got)
	}
	if len(w.Sacks()) != 0 {
		t.Fatalf("downed unit dropped a sack: %+v", w.Sacks())
	}
	if carried, _ := w.Carried(1); !equalCodes(carried, []uint16{0x101}) {
		t.Fatalf("downed unit carry = %v, want retained [0x101]", carried)
	}

	Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 9}})
	if got := w.Entities()[0].HP; got != -9 {
		t.Fatalf("finishable body left HP %d, want -9", got)
	}
	if len(w.Sacks()) != 0 {
		t.Fatalf("HP -9 body dropped a sack: %+v", w.Sacks())
	}
	if worn, _ := w.Equipped(1); worn[0] != 0x201 {
		t.Fatalf("HP -9 body lost worn item: %v", worn)
	}

	Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 1}})
	if got := w.Entities()[0]; got.HP > decayBonesHP || got.Decay < DecayBones {
		t.Fatalf("terminal body = HP %d decay %d, want <=%d and bones", got.HP, got.Decay, decayBonesHP)
	}
	if sacks := w.Sacks(); len(sacks) != 1 || !equalCodes(sacks[0].Items, []uint16{0x101, 0x201}) {
		t.Fatalf("terminal sack = %+v, want carried item then worn slot", sacks)
	}
}

// ---------------------------------------------------------------- AC-1

// TestAKillDropsTheContainerInOrderAndEmptiesIt is AC-1 and the spec's own
// first I/O example: a unit at (4,6) carrying two codes, killed, leaves one
// sack at that cell holding both codes in the container's own order, and the
// unit itself carries nothing afterwards.
func TestAKillDropsTheContainerInOrderAndEmptiesIt(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 4, Y: 6}},
		[]Stock{{ID: 1, Items: []uint16{0x101, 0x102}}})

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	got := w.Sacks()
	if len(got) != 1 {
		t.Fatalf("got %d sack(s), want 1: %+v", len(got), got)
	}
	if got[0].X != 4 || got[0].Y != 6 || !equalCodes(got[0].Items, []uint16{0x101, 0x102}) {
		t.Errorf("the sack is %+v, want (4,6) holding [0x101 0x102]", got[0])
	}

	carried, ok := w.Carried(1)
	if !ok {
		t.Fatal("Carried(1) answered not-ok for an entity this world still holds")
	}
	if len(carried) != 0 {
		t.Errorf("Carried(1) = %v after the drop, want nothing", carried)
	}
}

// ---------------------------------------------------------------- AC-2

// TestAnEmptyContainerLeavesTheSackListUnchanged is AC-2 in 0123's numbering
// and 0132's AC-4: the entity here is named no Stock at all, so it carries
// nothing AND wears nothing — holding nothing at all — and dying leaves the
// sack list byte-identical to what it was, witnessed here against a
// standing, unrelated sack elsewhere on the map, so the case is not passing
// merely because the world held no sack to disturb in the first place.
func TestAnEmptyContainerLeavesTheSackListUnchanged(t *testing.T) {
	w, err := NewStockedWorld(1, ddBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 4, Y: 6}}, nil, Relations{},
		[]Sack{{X: 0, Y: 0, Gold: 3, Items: []uint16{0x050}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	after := w.Sacks()
	if len(after) != 1 || after[0].X != 0 || after[0].Y != 0 || after[0].Gold != 3 ||
		!equalCodes(after[0].Items, []uint16{0x050}) {
		t.Errorf("the sack list is %+v after killing an empty-handed entity, want it unchanged: "+
			"one sack at (0,0), gold 3, items [0x050]", after)
	}
}

func TestCorpseLootSuppressionDeletesAnNPCTemplateLoadoutButKeepsOrdinaryDrops(t *testing.T) {
	loadout := [EquipSlots]uint16{0x101, 0x102, 0x103, 0, 0x105}
	w := mustStockedWorld(t, 1,
		[]Entity{
			{ID: 1, X: 3, Y: 3, SuppressCorpseLoot: true},
			{ID: 2, X: 6, Y: 6},
		},
		[]Stock{
			{ID: 1, Items: []uint16{0x110, 0x111}, Equipped: loadout},
			{ID: 2, Items: []uint16{0x210, 0x211}, Equipped: loadout},
		})

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}, {Kind: KindTerminalKill, Entity: 2}})

	got := w.Sacks()
	if len(got) != 1 || got[0].X != 6 || got[0].Y != 6 {
		t.Fatalf("Sacks() = %+v, want only the ordinary actor's sack at (6,6)", got)
	}
	want := []uint16{0x210, 0x211, 0x102, 0x101, 0x103, 0x105}
	if !equalCodes(got[0].Items, want) {
		t.Fatalf("ordinary actor drop = %v, want %v", got[0].Items, want)
	}
	for _, id := range []EntityID{1, 2} {
		if carried, ok := w.Carried(id); !ok || len(carried) != 0 {
			t.Errorf("Carried(%d) = %v, ok=%v; want an empty container", id, carried, ok)
		}
		if worn, ok := w.Equipped(id); !ok || worn != ([EquipSlots]uint16{}) {
			t.Errorf("Equipped(%d) = %v, ok=%v; want the zero array", id, worn, ok)
		}
	}
}

func TestCorpseLootSuppressionDeletesTheContainerBeforeAnOffMapSackRefusal(t *testing.T) {
	loadout := [EquipSlots]uint16{0x101, 0x102, 0x103}
	w := mustStockedWorld(t, 1,
		[]Entity{
			{ID: 1, X: -1, Y: 3, SuppressCorpseLoot: true},
			{ID: 2, X: -1, Y: 6},
		},
		[]Stock{
			{ID: 1, Items: []uint16{0x110}, Equipped: loadout},
			{ID: 2, Items: []uint16{0x210}, Equipped: loadout},
		})

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}, {Kind: KindTerminalKill, Entity: 2}})

	if len(w.Sacks()) != 0 {
		t.Fatalf("off-map deaths produced sacks %+v", w.Sacks())
	}
	if carried, _ := w.Carried(1); len(carried) != 0 {
		t.Fatalf("suppressed off-map container = %v, want deleted", carried)
	}
	if worn, _ := w.Equipped(1); worn != ([EquipSlots]uint16{}) {
		t.Fatalf("suppressed off-map equipment = %v, want deleted", worn)
	}
	if carried, _ := w.Carried(2); !equalCodes(carried, []uint16{0x210}) {
		t.Fatalf("ordinary off-map container = %v, want retained", carried)
	}
	if worn, _ := w.Equipped(2); worn != loadout {
		t.Fatalf("ordinary off-map equipment = %v, want retained %v", worn, loadout)
	}
}

func TestCorpseLootSuppressionIsCanonicalAndStrictlyDecoded(t *testing.T) {
	w := mustStockedWorld(t, 5,
		[]Entity{{ID: 1, X: 2, Y: 2, SuppressCorpseLoot: true}}, nil)
	form := mustMarshal(t, w)

	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	ents := back.Entities()
	if len(ents) != 1 || !ents[0].SuppressCorpseLoot {
		t.Fatalf("round-trip entities = %+v, want the suppression property set", ents)
	}
	if back.Hash() != w.Hash() {
		t.Fatalf("round-trip hash = %#016x, want %#016x", back.Hash(), w.Hash())
	}

	plain := mustStockedWorld(t, 5, []Entity{{ID: 1, X: 2, Y: 2}}, nil)
	if plain.Hash() == w.Hash() {
		t.Fatal("suppressed and ordinary actors hash equally")
	}

	broken := append([]byte(nil), form...)
	// Header 34 + three 10x10 planes + entity offset 260.
	broken[34+3*100+260] = 2
	if err := (&World{}).UnmarshalBinary(broken); err == nil ||
		!bytes.Contains([]byte(err.Error()), []byte("corpse-loot suppression")) {
		t.Fatalf("invalid suppression byte error = %v, want the named strict refusal", err)
	}
}

// ---------------------------------------------------------------- AC-3

// TestTwoEntitiesFellingOnOneCellInOneAdvanceLeaveOneSackInDeathOrder is
// AC-3: two entities standing on one cell, each carrying codes, killed by
// two commands in the one Step call, leave ONE sack on that cell holding
// both sets in death order — which, since Step applies commands in slice
// order (step.go's own contract), is the order the two kill commands were
// given in.
func TestTwoEntitiesFellingOnOneCellInOneAdvanceLeaveOneSackInDeathOrder(t *testing.T) {
	w := mustStockedWorld(t, 1,
		[]Entity{{ID: 1, X: 5, Y: 5}, {ID: 2, X: 5, Y: 5}},
		[]Stock{{ID: 1, Items: []uint16{1, 2}}, {ID: 2, Items: []uint16{3}}})

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}, {Kind: KindTerminalKill, Entity: 2}})

	got := w.Sacks()
	if len(got) != 1 {
		t.Fatalf("got %d sack(s), want 1: %+v", len(got), got)
	}
	if got[0].X != 5 || got[0].Y != 5 || !equalCodes(got[0].Items, []uint16{1, 2, 3}) {
		t.Errorf("the sack is %+v, want (5,5) holding [1 2 3] — death order", got[0])
	}
}

// ---------------------------------------------------------------- 0132 AC-11

func TestTwoFullLoadoutsFellingOnOneCellConcatenateEachInFR2Order(t *testing.T) {
	build := func(t *testing.T) *World {
		t.Helper()
		return mustStockedWorld(t, 1,
			[]Entity{{ID: 1, X: 5, Y: 5}, {ID: 2, X: 5, Y: 5}},
			[]Stock{
				{ID: 1, Items: []uint16{0x0a0}, Equipped: [EquipSlots]uint16{0x0a1, 0x0a2, 0x0a3}},
				{ID: 2, Items: []uint16{0x0b0}, Equipped: [EquipSlots]uint16{0x0b1, 0, 0, 0, 0x0b5}},
			})
	}
	block1 := []uint16{0x0a0, 0x0a2, 0x0a1, 0x0a3} //
	block2 := []uint16{0x0b0, 0x0b1, 0x0b5}        //

	forward := build(t)
	Step(forward, []Command{{Kind: KindTerminalKill, Entity: 1}, {Kind: KindTerminalKill, Entity: 2}})
	gotForward := forward.Sacks()
	if len(gotForward) != 1 {
		t.Fatalf("AC-11, entity 1 killed first: got %d sack(s), want 1: %+v", len(gotForward), gotForward)
	}
	wantForward := append(append([]uint16{}, block1...), block2...)
	if gotForward[0].X != 5 || gotForward[0].Y != 5 || !equalCodes(gotForward[0].Items, wantForward) {
		t.Errorf("AC-11, entity 1 killed first: the sack is %+v, want (5,5) holding %v — "+
			"entity 1's block then entity 2's", gotForward[0], wantForward)
	}

	reverse := build(t)
	Step(reverse, []Command{{Kind: KindTerminalKill, Entity: 2}, {Kind: KindTerminalKill, Entity: 1}})
	gotReverse := reverse.Sacks()
	if len(gotReverse) != 1 {
		t.Fatalf("AC-11, entity 2 killed first: got %d sack(s), want 1: %+v", len(gotReverse), gotReverse)
	}
	wantReverse := append(append([]uint16{}, block2...), block1...)
	if gotReverse[0].X != 5 || gotReverse[0].Y != 5 || !equalCodes(gotReverse[0].Items, wantReverse) {
		t.Errorf("AC-11, entity 2 killed first: the sack is %+v, want (5,5) holding %v — "+
			"the same two blocks as above, swapped, neither block itself changed",
			gotReverse[0], wantReverse)
	}
}

// ---------------------------------------------------------------- AC-4

// TestADropOntoAStandingSackAppendsAtItsTailAndLeavesItsGoldAlone is AC-4 in
// 0123's numbering and 0132's AC-8: a drop onto a cell that already holds a
// sack pours into that sack — its codes appended at that sack's tail, its
// gold left alone — and the list stays ascending by (Y, X), witnessed with a
// sack on either side of the merged cell. pourSack (sack.go) receives
// exactly the same flat item list whether it was assembled from the
// container alone or from the container and worn slots together, so this
// fixture's container-only drop already witnesses 0132's widened source.
func TestADropOntoAStandingSackAppendsAtItsTailAndLeavesItsGoldAlone(t *testing.T) {
	w, err := NewStockedWorld(1, ddBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3}}, nil, Relations{},
		[]Sack{
			{X: 1, Y: 1, Gold: 9},
			{X: 3, Y: 3, Gold: 25, Items: []uint16{0x090}},
			{X: 8, Y: 8, Gold: 1},
		},
		[]Stock{{ID: 1, Items: []uint16{0x101, 0x102}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	got := w.Sacks()
	want := [][2]int32{{1, 1}, {3, 3}, {8, 8}}
	if len(got) != len(want) {
		t.Fatalf("got %d sack(s), want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].X != want[i][0] || got[i].Y != want[i][1] {
			t.Errorf("sack %d is at (%d,%d), want (%d,%d) — the list must stay ascending by (Y,X)",
				i, got[i].X, got[i].Y, want[i][0], want[i][1])
		}
	}
	merged := got[1]
	if merged.Gold != 25 || !equalCodes(merged.Items, []uint16{0x090, 0x101, 0x102}) {
		t.Errorf("the merged sack is %+v, want gold 25 (untouched) and items [0x090 0x101 0x102] "+
			"(the standing sack's own codes, then the corpse's)", merged)
	}
}

// ---------------------------------------------------------------- AC-5

// TestASecondKillOnAnAlreadyDeadEntityChangesNothing is AC-6 (0132's
// numbering; AC-5 in 0123's): a second kill command on an entity that has
// already died leaves the sack list and every container AND every equipment
// slot unchanged — WIDENED BY 0132 with a worn armour slot, so the guard's
// once-per-death property is measured over the ten new fields and not only
// the container. It is measured against a QUIET tick — one carrying no
// command at all — on TestEveryNoOpBlowLeavesTheWorldWhereAQuietTickLeavesIt's
// own pattern (damage_test.go): the digest covers every field of every unit
// and every byte of the form, so a second drop that moved anything this file
// does not name still fails here.
func TestASecondKillOnAnAlreadyDeadEntityChangesNothing(t *testing.T) {
	build := func(t *testing.T) *World {
		t.Helper()
		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 2, Y: 2}},
			[]Stock{{ID: 1, Items: []uint16{7}, Equipped: [EquipSlots]uint16{0, 0, 0, 0, 0x0800}}})
		Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})
		return w
	}

	quiet := build(t)
	Step(quiet, nil)

	again := build(t)
	Step(again, []Command{{Kind: KindTerminalKill, Entity: 1}})

	if got, want := again.Hash(), quiet.Hash(); got != want {
		t.Errorf("a second kill on an already-dead entity hashes %#016x, a quiet tick %#016x", got, want)
	}
	if got := again.Sacks(); len(got) != 1 || got[0].X != 2 || got[0].Y != 2 ||
		!equalCodes(got[0].Items, []uint16{7, 0x0800}) {
		t.Errorf("Sacks() after a second kill is %+v, want the one sack the first death dropped, untouched", got)
	}
}

// TestADeathOffTheWorldsBoundsDropsNothingAndLeavesTheContainer is SC-5: an
// entity whose cell is outside the world's bounds leaves no sack, and its
// container stands exactly as it was — a refusal, not a clamp and not a
// fold onto a nearby cell.
func TestADeathOffTheWorldsBoundsDropsNothingAndLeavesTheContainer(t *testing.T) {
	small := Bounds{Width: 5, Height: 5}
	w, err := NewStockedWorld(1, small, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: small.Width, Y: 2}}, nil, Relations{}, nil,
		[]Stock{{ID: 1, Items: []uint16{3, 4}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	if got := w.Sacks(); len(got) != 0 {
		t.Errorf("Sacks() = %+v after a corpse off the map, want none", got)
	}
	got, ok := w.Carried(1)
	if !ok {
		t.Fatal("Carried(1) answered not-ok")
	}
	if !equalCodes(got, []uint16{3, 4}) {
		t.Errorf("Carried(1) = %v after a corpse off the map, want [3 4] — refused, not moved", got)
	}
}

// TestADeathAndDropDrawsNothingFromTheGenerator is AC-7: the generator state
// after an advance in which an entity dies and drops is the state that
// advance would have reached with no death in it.
func TestADeathAndDropDrawsNothingFromTheGenerator(t *testing.T) {
	build := func(t *testing.T, cmds []Command) *World {
		t.Helper()
		w := mustStockedWorld(t, 99, []Entity{{ID: 1, X: 3, Y: 3}},
			[]Stock{{ID: 1, Items: []uint16{0x777}}})
		Step(w, cmds)
		return w
	}

	dying := build(t, []Command{{Kind: KindTerminalKill, Entity: 1}})
	quiet := build(t, nil)

	if got, want := dying.rng.state, quiet.rng.state; got != want {
		t.Errorf("an advance in which an entity dies and drops leaves the generator at %#016x, "+
			"want %#016x — the same advance with nothing to drop", got, want)
	}
}

func TestEligibleCreatureDeathDropsGoldFromItsUnitsColumns(t *testing.T) {
	w, err := NewLootWorld(7, ddBounds, ModeCanonical, Terrain{}, []Entity{{
		ID: 1, X: 4, Y: 5, TypeID: 0x41, GoldChance: 101, TreasureMin: 37,
	}}, nil, Relations{}, nil)
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	got := w.Sacks()
	if len(got) != 1 || got[0].X != 4 || got[0].Y != 5 || got[0].Gold != 37 || len(got[0].Items) != 0 {
		t.Fatalf("Sacks() = %+v, want one gold-only sack at (4,5) carrying 37", got)
	}
}

func TestDeathGoldRequiresATypeIDAboveTheStrictThreshold(t *testing.T) {
	w, err := NewLootWorld(7, ddBounds, ModeCanonical, Terrain{}, []Entity{{
		ID: 1, X: 4, Y: 5, TypeID: 0x40, GoldChance: 101, TreasureMin: 37,
	}}, nil, Relations{}, nil)
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	before := w.rng.state

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	if got := w.Sacks(); len(got) != 0 {
		t.Fatalf("Sacks() = %+v, want none for type id 0x40", got)
	}
	if w.rng.state != before {
		t.Fatalf("ineligible death advanced RNG from %#x to %#x", before, w.rng.state)
	}
}

func TestDeathGoldMergesIntoTheOneSackOnTheCellWithWrapping(t *testing.T) {
	w, err := NewLootWorld(7, ddBounds, ModeCanonical, Terrain{}, []Entity{{
		ID: 1, X: 4, Y: 5, TypeID: 0x41, GoldChance: 101, TreasureMin: 2,
	}}, nil, Relations{}, []Sack{{X: 4, Y: 5, Gold: ^uint32(0)}})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	got := w.Sacks()
	if len(got) != 1 || got[0].Gold != 1 {
		t.Fatalf("Sacks() = %+v, want one sack whose gold wrapped to 1", got)
	}
}

// ---------------------------------------------------------------- AC-8

// TestAUnitStandingOnTheCorpseCellPicksUpTheDropWithTakeSack is AC-8: an
// entity standing on the cell where another died picks up the dropped codes
// with the existing transfer primitive (TakeSack, carry.go) and afterwards
// carries them.
func TestAUnitStandingOnTheCorpseCellPicksUpTheDropWithTakeSack(t *testing.T) {
	w := mustStockedWorld(t, 1,
		[]Entity{{ID: 1, X: 4, Y: 4}, {ID: 2, X: 4, Y: 4}},
		[]Stock{{ID: 1, Items: []uint16{0x201, 0x202}}})

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	if err := w.TakeSack(2, 4, 4); err != nil {
		t.Fatalf("TakeSack: %v", err)
	}
	got, ok := w.Carried(2)
	if !ok {
		t.Fatal("Carried(2) answered not-ok")
	}
	if !equalCodes(got, []uint16{0x201, 0x202}) {
		t.Errorf("Carried(2) = %v after picking up the corpse's drop, want [0x201 0x202]", got)
	}
	if got := w.Sacks(); len(got) != 0 {
		t.Errorf("Sacks() = %+v after the pick-up, want none left", got)
	}
}

// TestAWorldWithADeathRoundTripsAndKeepsTheFormVersion is AC-9: a world in
// which a death has occurred encodes and decodes back to an equal world, and
// the encoded form still declares formatVersion. WIDENED BY 0132: entity 1
// now also wears armour that the death drops, so the round trip exercises a
// world whose sack was built from equipment slots and not the container
// alone, while the version this test pins is unmoved — 0132 adds no field.
//
// THE TRIPWIRE HAS NOW FIRED SEVEN TIMES, AND EVERY TIME CORRECTLY. 0123
// wrote 26 here because 0123 adds no field, and that is still true — the
// record this test round-trips is unchanged. What moved each time is
// somebody else's field: first 0122 gave ScriptCheck a player reference,
// widening the compiled script's own record and taking version 31; then
// 0117 gave the entity record a CommandGroup word, taking version 32; then
// 0124 T2 added the equipment section between the carry section and the
// purse, taking version 34 (33 is out to a lane running in parallel off the
// same master this one branched from, and this tree carries none of its
// section); then 0125 gave the entity record an experience-from-use block,
// taking version 35; then 0127 gave the entity record a KnownSpells mask
// and the world a spell table, taking version 36; then 0129 gave the
// instant record a second unit reference so opcode 28 (Give All) has
// somewhere to put the receiver it names, taking version 38 (37 is out to a
// story running in parallel off the same master this one branched from, and
// this tree carries none of its change); then 0135 gives the entity record
// six skill levels, taking version 39; and now 0139 gives the entity record
// a weapon's own spell — an id and a level — and a fourth attack phase,
// taking version 41 (40 is out to a story running in parallel off the same
// master this one branched from, and this tree carries none of its change).
// So the literal is re-pinned rather than removed, and this note is why: an
// assertion that a story did not move the constant cannot survive a story
// that legitimately does, and deleting the guard to make the merge quiet
// would spend the one thing it exists to catch. Re-pin it again, with a
// sentence, at the next bump.
//
// IT FIRES AGAIN for 0156, which gives the compiled INSTANT record an item
// reference — a packed code and its presence flag — so instant opcodes 12
// and 13 have somewhere to read the item they name, taking version 46. The
// record this test round-trips is unchanged once more: 0156 adds no entity
// field.
//
// AND AGAIN for 0164, which adds the off-map bit to the entity record and
// takes version 48. This one DOES widen the record — by one byte, at +222 —
// so the round trip below covers a field it did not before, and it covers it
// clear: nothing but a script arm sets the bit and this fixture runs no
// script.
//
// AND AGAIN for 0166, which takes version 50 and widens the record by seven:
// six of escort triple at the tail and one from widening the spell mark's
// remaining ticks to a word. Both are covered clear here for the off-map bit's
// own reason — nothing but a script arm writes either and this fixture runs no
// script.
//
// THE VERSION LITERAL IS GONE, AT 0166. This guard spelled the live version
// number and had to be re-spelled by every story that bumped it — the exact
// shape that has gone stale and been repaired after the fact three times
// elsewhere in this package. What the test is named for is that a world with a
// death ROUND TRIPS AND COMES BACK AT THE VERSION IT WENT IN AT, and that is
// checked below against the constant rather than against a number written here.
// The version's own tripwire is the pinned form and digest in binary_test.go
// and hash_test.go, which cannot be satisfied by a bump nobody looked at.
func TestAWorldWithADeathRoundTripsAndKeepsTheFormVersion(t *testing.T) {

	w := mustStockedWorld(t, 5,
		[]Entity{{ID: 1, X: 2, Y: 2}, {ID: 2, X: 6, Y: 6}},
		[]Stock{
			{ID: 1, Items: []uint16{0x301, 0x302}, Equipped: [EquipSlots]uint16{0x0111, 0x0222, 0x0333}},
			{ID: 2, Items: []uint16{0x040}},
		})

	Step(w, []Command{{Kind: KindTerminalKill, Entity: 1}})

	if eq, ok := w.Equipped(1); !ok || eq != ([EquipSlots]uint16{}) {
		t.Fatalf("Equipped(1) = %v (ok=%v) after the kill, want the zero array", eq, ok)
	}

	form := mustMarshal(t, w)
	if form[0] != formatVersion {
		t.Fatalf("the encoded form declares version %d, want %d", form[0], formatVersion)
	}

	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, want %#016x", back.Hash(), w.Hash())
	}

	gotSacks, wantSacks := back.Sacks(), w.Sacks()
	if len(gotSacks) != len(wantSacks) {
		t.Fatalf("the round trip holds %d sack(s), want %d", len(gotSacks), len(wantSacks))
	}
	for i := range wantSacks {
		if gotSacks[i].X != wantSacks[i].X || gotSacks[i].Y != wantSacks[i].Y ||
			gotSacks[i].Gold != wantSacks[i].Gold || !equalCodes(gotSacks[i].Items, wantSacks[i].Items) {
			t.Errorf("sack %d is %+v, want %+v", i, gotSacks[i], wantSacks[i])
		}
	}

	gotCarried2, _ := back.Carried(2)
	wantCarried2, _ := w.Carried(2)
	if !equalCodes(gotCarried2, wantCarried2) {
		t.Errorf("entity 2's container round-trips as %v, want %v", gotCarried2, wantCarried2)
	}

	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round two): %v", err)
	}
	if !bytes.Equal(again, form) {
		t.Errorf("re-marshalling a world with a death gave\n % x\nwant\n % x", again, form)
	}
}

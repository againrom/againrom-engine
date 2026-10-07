package sim

// The ground drop (1005 round 2, ITEM-DROP-008, ITEM-CMD-007's destination
// code 3): KindDropCarried, KindDropWorn and the dropToGround geometry both
// share, over mustStockedWorld's own fixture (equip_test.go's own
// precedent).

import "testing"

// ---------------------------------------------------------------- ITEM-DROP-008, the window

// TestDropFromContainerLandsAtTheRequestedCellInsideTheWindow is the
// positive half of ITEM-DROP-008: a request at most 2 cells away on both
// axes from the entity's own cell (5, 5) plants the sack exactly there.
func TestDropFromContainerLandsAtTheRequestedCellInsideTheWindow(t *testing.T) {
	const a = 0x101
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{a}}})

	Step(w, []Command{{Kind: KindDropCarried, Entity: 1, X: 7, Y: 3, Spell: 0}})

	sacks := w.Sacks()
	if len(sacks) != 1 {
		t.Fatalf("Sacks() = %d sacks, want 1", len(sacks))
	}
	if sacks[0].X != 7 || sacks[0].Y != 3 {
		t.Errorf("sack at (%d,%d), want (7,3) — 2 east and 2 north is inside the 5x5 window", sacks[0].X, sacks[0].Y)
	}
	if !equalCodes(sacks[0].Items, []uint16{a}) {
		t.Errorf("sack items = %#x, want [%#x]", sacks[0].Items, a)
	}
	gotCarried, _ := w.Carried(1)
	if len(gotCarried) != 0 {
		t.Errorf("Carried(1) = %#x, want empty — the one unit moved to the ground", gotCarried)
	}
}

// TestDropFromContainerOutsideTheWindowLandsAtTheDroppersOwnCell is
// ITEM-DROP-008's own negative half: a request 3 cells away on one axis —
// outside the window on that axis alone — falls back to the entity's own
// cell rather than being refused.
func TestDropFromContainerOutsideTheWindowLandsAtTheDroppersOwnCell(t *testing.T) {
	const a = 0x101
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{a}}})

	Step(w, []Command{{Kind: KindDropCarried, Entity: 1, X: 8, Y: 5, Spell: 0}})

	sacks := w.Sacks()
	if len(sacks) != 1 {
		t.Fatalf("Sacks() = %d sacks, want 1", len(sacks))
	}
	if sacks[0].X != 5 || sacks[0].Y != 5 {
		t.Errorf("sack at (%d,%d), want (5,5) — the dropper's own cell, 3 east is outside the window", sacks[0].X, sacks[0].Y)
	}
}

// TestDropRefusesNoDistanceItIsNeverRefused witnesses ITEM-DROP-008's "never
// refused" directly: a request wildly outside the map still plants a sack,
// at the dropper's own cell.
func TestDropRefusesNoDistanceItIsNeverRefused(t *testing.T) {
	const a = 0x101
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 0, Y: 0}}, []Stock{{ID: 1, Items: []uint16{a}}})

	Step(w, []Command{{Kind: KindDropCarried, Entity: 1, X: -1_000_000, Y: 1_000_000, Spell: 0}})

	sacks := w.Sacks()
	if len(sacks) != 1 {
		t.Fatalf("Sacks() = %d sacks, want 1 — a request off the map is never refused", len(sacks))
	}
	if sacks[0].X != 0 || sacks[0].Y != 0 {
		t.Errorf("sack at (%d,%d), want (0,0) — the dropper's own cell, both requested axes out of bounds", sacks[0].X, sacks[0].Y)
	}
}

// TestDropOutOfBoundsRequestInsideTheWindowStillLandsAtTheDroppersOwnCell is
// this build's own encoding safety net, not ITEM-DROP-008's own clause: a
// request within the Chebyshev-2 window but off the map (the dropper stands
// at the corner) cannot be planted — sackFault refuses any cell outside
// Bounds — so it falls back exactly as an out-of-window request does.
func TestDropOutOfBoundsRequestInsideTheWindowStillLandsAtTheDroppersOwnCell(t *testing.T) {
	const a = 0x101
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 0, Y: 0}}, []Stock{{ID: 1, Items: []uint16{a}}})

	Step(w, []Command{{Kind: KindDropCarried, Entity: 1, X: -1, Y: 0, Spell: 0}})

	sacks := w.Sacks()
	if len(sacks) != 1 {
		t.Fatalf("Sacks() = %d sacks, want 1", len(sacks))
	}
	if sacks[0].X != 0 || sacks[0].Y != 0 {
		t.Errorf("sack at (%d,%d), want (0,0) — (-1,0) is inside the window but off the map", sacks[0].X, sacks[0].Y)
	}
}

// TestDropFromAnEntityOutsideBoundsPlantsNoSack is follow-up 1's own guard
// (round-2 adversarial review): dropToGround's own doc names no path in this
// package that leaves a live entity's X, Y outside w.bounds, but the
// constructor itself does not refuse that shape (world_test.go's own sample
// fixture carries a negative X for the same reason), so the drop is driven
// directly against an entity built that way. The window's own fallback
// resolves to the entity's own cell, and here that cell is itself off the
// map — the SECOND sackFault check (drop.go) must refuse to plant rather
// than hand decodeSacks (sack.go) a sack it will not read back.
func TestDropFromAnEntityOutsideBoundsPlantsNoSack(t *testing.T) {
	const a = 0x101
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: -1, Y: 0}}, []Stock{{ID: 1, Items: []uint16{a}}})

	Step(w, []Command{{Kind: KindDropCarried, Entity: 1, X: -1, Y: 0, Spell: 0}})

	if sacks := w.Sacks(); len(sacks) != 0 {
		t.Fatalf("Sacks() = %d, want 0 — an out-of-bounds entity's own cell must not become a sack", len(sacks))
	}
	if held, _ := w.Carried(1); !equalCodes(held, []uint16{a}) {
		t.Fatalf("Carried(1) = %#x, want the item kept — a refused drop must not lose it", held)
	}
}

// TestDropMergesIntoAnExistingSackAtTheTargetCell is pourSack's own rule
// (sack.go), exercised through the command: a sack already standing at the
// resolved cell gains the dropped item rather than a second sack opening
// beside it.
func TestDropMergesIntoAnExistingSackAtTheTargetCell(t *testing.T) {
	const a, b = 0x101, 0x102
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{a}}})
	w.pourSack(6, 6, 0, plainItems([]uint16{b}))

	Step(w, []Command{{Kind: KindDropCarried, Entity: 1, X: 6, Y: 6, Spell: 0}})

	sacks := w.Sacks()
	if len(sacks) != 1 {
		t.Fatalf("Sacks() = %d sacks, want 1 — the drop merges rather than planting a second sack", len(sacks))
	}
	if !equalCodes(sacks[0].Items, []uint16{b, a}) {
		t.Errorf("sack items = %#x, want [b a] = %#x — the existing sack's own item first, the drop appended", sacks[0].Items, []uint16{b, a})
	}
}

// ---------------------------------------------------------------- source: the container element

// TestDropFromContainerNamesAnElementNotAFlatUnit is equip's own D-3,
// restated for the drop source: a container [(a,2),(b,1)] dropping element 1
// loses b entirely and keeps a's own count of 2 untouched — the element
// index names a PLACE, not a position in the flat expansion.
func TestDropFromContainerNamesAnElementNotAFlatUnit(t *testing.T) {
	const a, b = 0x101, 0x102
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{a, a, b}}})

	Step(w, []Command{{Kind: KindDropCarried, Entity: 1, X: 5, Y: 5, Spell: 1}})

	gotCarried, _ := w.Carried(1)
	if !equalCodes(gotCarried, []uint16{a, a}) {
		t.Errorf("Carried(1) = %#x, want [a a] = %#x — element 1 (b) dropped whole", gotCarried, []uint16{a, a})
	}
	sacks := w.Sacks()
	if len(sacks) != 1 || !equalCodes(sacks[0].Items, []uint16{b}) {
		t.Errorf("sack items = %+v, want [b]", sacks)
	}
}

// TestDropFromContainerDecrementsAStackAboveOne is equip's own arithmetic
// for a source that loses a unit (equip.go), restated for the drop: an
// element at count 2 stays in its own place at count 1 rather than being
// removed.
func TestDropFromContainerDecrementsAStackAboveOne(t *testing.T) {
	const a = 0x101
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{}, []Entity{{ID: 1, X: 5, Y: 5}}, nil, Relations{}, nil,
		[]Stock{{ID: 1, Items: []uint16{a, a}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	stacks, _ := w.CarriedStacks(1)
	if len(stacks) != 1 || stacks[0].Count != 2 {
		t.Fatalf("fixture: CarriedStacks(1) = %+v, want one element at count 2", stacks)
	}

	Step(w, []Command{{Kind: KindDropCarried, Entity: 1, X: 5, Y: 5, Spell: 0}})

	stacksAfter, _ := w.CarriedStacks(1)
	if len(stacksAfter) != 1 || stacksAfter[0].Count != 1 {
		t.Errorf("CarriedStacks(1) = %+v, want one element at count 1", stacksAfter)
	}
	sacks := w.Sacks()
	if len(sacks) != 1 || !equalCodes(sacks[0].Items, []uint16{a}) {
		t.Errorf("sack items = %+v, want [a]", sacks)
	}
}

// ---------------------------------------------------------------- source: the equipment slot

// TestDropFromEquipmentClearsTheSlotAndPlantsTheCode is KindDropWorn's own
// positive: the worn code reaches the ground and the slot is left empty,
// unequip's own move (equip.go) with the ground in place of the container.
func TestDropFromEquipmentClearsTheSlotAndPlantsTheCode(t *testing.T) {
	const a = 0x201
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, nil)
	w.equipment[0][2] = PlainItem(a) // slot 3

	Step(w, []Command{{Kind: KindDropWorn, Entity: 1, X: 5, Y: 5, Spell: 3}})

	got, _ := w.Equipped(1)
	if got[2] != 0 {
		t.Errorf("slot 3 = %#x, want 0 (empty)", got[2])
	}
	sacks := w.Sacks()
	if len(sacks) != 1 || !equalCodes(sacks[0].Items, []uint16{a}) {
		t.Errorf("sack items = %+v, want [%#x]", sacks, a)
	}
	gotCarried, _ := w.Carried(1)
	if len(gotCarried) != 0 {
		t.Errorf("Carried(1) = %#x, want empty — a drop from the doll never touches the container", gotCarried)
	}
}

func TestDroppingWeaponLeavesTheShieldWorn(t *testing.T) {
	weapon := ItemInstance{Code: 0x0101, Kind: 2, Price: 411,
		Effects: []ItemEffect{{Kind: 12, Operand: 3}}}
	shield := ItemInstance{Code: 0x0201, Kind: 1, Price: 722,
		Effects: []ItemEffect{{Kind: 15, Operand: 9}}}
	var worn [EquipSlots]ItemInstance
	worn[0], worn[1] = weapon, shield
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5, HP: 20, MaxHP: 20}},
		[]Stock{{ID: 1, EquippedItems: worn}})

	Step(w, []Command{{Kind: KindDropWorn, Entity: 1, X: 5, Y: 5, Spell: 1}})

	gotWorn, _ := w.EquippedItems(1)
	gotPack, _ := w.CarriedItems(1)
	if !gotWorn[0].Empty() || !ItemEqual(gotWorn[1], shield) || gotWorn[1].Price != shield.Price || len(gotPack) != 0 {
		t.Fatalf("weapon drop left worn=%+v pack=%+v, want the complete shield still worn and an empty pack", gotWorn, gotPack)
	}
	sacks := w.Sacks()
	if len(sacks) != 1 || len(sacks[0].ItemInstances) != 1 ||
		!ItemEqual(sacks[0].ItemInstances[0], weapon) || sacks[0].ItemInstances[0].Price != weapon.Price {
		t.Fatalf("ground sacks = %+v, want complete weapon", sacks)
	}
}

func TestDropTouchesNoOtherSlotAndNoOtherEntity(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}, {ID: 2, X: 6, Y: 6}}, nil)
	w.equipment[0][2] = PlainItem(0x301) // entity 1, slot 3
	w.equipment[0][5] = PlainItem(0x302) // entity 1, slot 6 — must survive untouched
	w.equipment[1][2] = PlainItem(0x401) // entity 2, slot 3 — must survive untouched

	Step(w, []Command{{Kind: KindDropWorn, Entity: 1, X: 5, Y: 5, Spell: 3}})

	got1, _ := w.Equipped(1)
	if got1[2] != 0 {
		t.Errorf("entity 1 slot 3 = %#x, want 0", got1[2])
	}
	if got1[5] != 0x302 {
		t.Errorf("entity 1 slot 6 = %#x, want 0x302 (untouched)", got1[5])
	}
	got2, _ := w.Equipped(2)
	if got2[2] != 0x401 {
		t.Errorf("entity 2 slot 3 = %#x, want 0x401 (untouched)", got2[2])
	}
}

// ---------------------------------------------------------------- AC-5-style refusals

// dropNoopWorld is the fixture every refusal is measured on: one entity
// carrying two codes and wearing one item at slot 1, eqNoopWorld's own
// shape (equip_test.go) widened with an occupied slot the way unNoopWorld
// widens it for unequip.
func dropNoopWorld(t *testing.T) *World {
	t.Helper()
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{0x101, 0x102}}})
	w.equipment[0][0] = PlainItem(0x201) // slot 1 occupied; slot 2 and past EquipSlots stay empty
	return w
}

// TestDropRefusesAnAbsentEntityAndAnOutOfRangeSource is equip's own AC-5,
// restated for both drop kinds: each case leaves the world byte-identical
// to a tick carrying no command at all, over the whole byte form —
// sacks are not among snap()'s pinned fields (world_test.go, equip_test.go's
// own precedent), so only the marshalled bytes see either move.
func TestDropRefusesAnAbsentEntityAndAnOutOfRangeSource(t *testing.T) {
	cases := []struct {
		name string
		cmd  Command
	}{
		{"KindDropCarried, an entity the world does not hold", Command{Kind: KindDropCarried, Entity: 99, X: 5, Y: 5, Spell: 0}},
		{"KindDropCarried, an index past the container", Command{Kind: KindDropCarried, Entity: 1, X: 5, Y: 5, Spell: 2}},
		{"KindDropCarried, a negative index", Command{Kind: KindDropCarried, Entity: 1, X: 5, Y: 5, Spell: 0xffff}},
		{"KindDropWorn, an entity the world does not hold", Command{Kind: KindDropWorn, Entity: 99, X: 5, Y: 5, Spell: 1}},
		{"KindDropWorn, a slot of 0", Command{Kind: KindDropWorn, Entity: 1, X: 5, Y: 5, Spell: 0}},
		{"KindDropWorn, a slot past EquipSlots", Command{Kind: KindDropWorn, Entity: 1, X: 5, Y: 5, Spell: EquipSlots + 1}},
		{"KindDropWorn, a slot already empty", Command{Kind: KindDropWorn, Entity: 1, X: 5, Y: 5, Spell: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := dropNoopWorld(t)
			Step(w, []Command{tc.cmd})

			quiet := dropNoopWorld(t)
			Step(quiet, nil)

			gotForm, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			wantForm, err := quiet.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary (quiet): %v", err)
			}
			if string(gotForm) != string(wantForm) {
				t.Errorf("%s: the world's byte form moved", tc.name)
			}
			if w.Hash() != quiet.Hash() {
				t.Errorf("%s: the world's digest moved", tc.name)
			}
		})
	}

	// The control: a command that IS applied moves the digest, so the cases
	// above are not passing because Step ignores the two kinds outright.
	live := dropNoopWorld(t)
	Step(live, []Command{{Kind: KindDropCarried, Entity: 1, X: 5, Y: 5, Spell: 0}})
	quiet := dropNoopWorld(t)
	Step(quiet, nil)
	if live.Hash() == quiet.Hash() {
		t.Errorf("a drop that lands leaves the same digest %#016x as a quiet tick — "+
			"the cases above then witness nothing", quiet.Hash())
	}
}

// ---------------------------------------------------------------- round trip

// TestAWorldHoldingAPlayerPlacedSackRoundTripsByteIdentically is ITEM-SACK-010's
// own claim exercised through this story's own producer: a sack a drop
// planted is ordinary canonical sack state (decodeSacks, binary.go), already
// covered on the map-placed and corpse-drop paths, so it marshals,
// unmarshals and re-marshals to the same bytes and hashes the same before
// and after.
func TestAWorldHoldingAPlayerPlacedSackRoundTripsByteIdentically(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{0x101}}})
	Step(w, []Command{{Kind: KindDropCarried, Entity: 1, X: 5, Y: 5, Spell: 0}})

	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round trip): %v", err)
	}
	if string(again) != string(b) {
		t.Fatal("a world holding a player-placed sack does not round-trip through its byte form")
	}
	if back.Hash() != w.Hash() {
		t.Fatal("a round-tripped world holding a player-placed sack hashes differently from the original")
	}
}

func TestDropOntoGroundBlockedCellLandsAtTheDroppersOwnCell(t *testing.T) {
	const a, b = 0x101, 0x102
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{a, b}}})
	i, _ := w.cellIndex(6, 4)
	w.grid[i] |= blockGround
	w.equipment[0][2] = PlainItem(0x201)
	Step(w, []Command{DropCarried(1, 0, CellPoint{X: 6, Y: 4})})
	Step(w, []Command{DropWorn(1, 3, CellPoint{X: 6, Y: 4})})
	Step(w, []Command{DropCarried(1, 0, CellPoint{X: 4, Y: 6})})
	sacks := w.Sacks()
	if len(sacks) != 2 || sacks[0].X != 5 || sacks[0].Y != 5 || !equalCodes(sacks[0].Items, []uint16{a, 0x201}) || sacks[1].X != 4 || sacks[1].Y != 6 {
		t.Fatalf("sacks %+v, want the blocked request at the dropper's cell and the open one at 4,6", sacks)
	}
}

// ---------------------------------------------------------------- DropLanding

// TestDropLandingAimsBesideTheDropperWhereAPlainDropWouldHideTheSack: the window
// plants a drop beyond it, off the map, on a blocked cell or on the dropper at
// the dropper's own cell, beneath the dropper. DropLanding names the south
// neighbour instead, and a command aimed there lands there.
func TestDropLandingAimsBesideTheDropperWhereAPlainDropWouldHideTheSack(t *testing.T) {
	const a = 0x101
	for _, req := range []CellPoint{{X: 8, Y: 5}, {X: 5, Y: 5}, {X: -1_000_000, Y: 1_000_000}} {
		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{a}}})
		at, ok := w.DropLanding(1, req.X, req.Y)
		if !ok || at != (CellPoint{X: 5, Y: 6}) {
			t.Fatalf("request %v: DropLanding = %v, %v, want 5,6", req, at, ok)
		}
		Step(w, []Command{DropCarried(1, 0, at)})
		if sacks := w.Sacks(); len(sacks) != 1 || sacks[0].X != 5 || sacks[0].Y != 6 || !equalCodes(sacks[0].Items, []uint16{a}) {
			t.Fatalf("request %v: sacks %+v, want one sack holding the item at 5,6", req, sacks)
		}
	}
}

// TestDropLandingKeepsAFreeRequestInsideTheWindow: nothing is moved when the
// requested cell is open and holds no unit.
func TestDropLandingKeepsAFreeRequestInsideTheWindow(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, nil)
	for _, req := range []CellPoint{{X: 7, Y: 3}, {X: 4, Y: 5}, {X: 5, Y: 4}} {
		if at, ok := w.DropLanding(1, req.X, req.Y); !ok || at != req {
			t.Errorf("DropLanding(%v) = %v, %v, want the request", req, at, ok)
		}
	}
}

// TestDropLandingAvoidsAnotherLivingUnitButNotACorpse: a sack beneath another
// living unit is as hidden as one beneath the dropper; a dead body lies beneath
// its sack, so it hides nothing.
func TestDropLandingAvoidsAnotherLivingUnitButNotACorpse(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}, {ID: 2, X: 6, Y: 5}, {ID: 3, X: 4, Y: 5, HP: -3, MaxHP: 10}}, nil)
	if at, _ := w.DropLanding(1, 6, 5); at != (CellPoint{X: 5, Y: 6}) {
		t.Errorf("onto a living unit: DropLanding = %v, want 5,6", at)
	}
	if at, _ := w.DropLanding(1, 4, 5); at != (CellPoint{X: 4, Y: 5}) {
		t.Errorf("onto a corpse: DropLanding = %v, want the request", at)
	}
}

// TestDropLandingPoursIntoASackUnderTheDropper: one sack per cell, so a dropper
// standing on a sack adds to it rather than opening a second beside it.
func TestDropLandingPoursIntoASackUnderTheDropper(t *testing.T) {
	const a, b = 0x101, 0x102
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, []Stock{{ID: 1, Items: []uint16{a}}})
	w.pourSack(5, 5, 0, plainItems([]uint16{b}))
	at, _ := w.DropLanding(1, 5, 5)
	Step(w, []Command{DropCarried(1, 0, at)})
	if sacks := w.Sacks(); len(sacks) != 1 || sacks[0].X != 5 || sacks[0].Y != 5 || !equalCodes(sacks[0].Items, []uint16{b, a}) {
		t.Fatalf("sacks %+v, want one sack at 5,5 holding both items", sacks)
	}
}

// TestDropLandingPrefersAnEmptyNeighbourToASackedOne: a free neighbour is used
// before merging into a sack, a sacked neighbour when every free one is closed,
// and the dropper's own cell, with the item still on the ground, when all are.
func TestDropLandingPrefersAnEmptyNeighbourToASackedOne(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 5, Y: 5}}, nil)
	w.pourSack(5, 6, 0, plainItems([]uint16{0x101}))
	if at, _ := w.DropLanding(1, 5, 5); at != (CellPoint{X: 6, Y: 5}) {
		t.Errorf("south holds a sack: DropLanding = %v, want 6,5", at)
	}
	for _, d := range dropRing {
		if d != [2]int32{0, 1} {
			i, _ := w.cellIndex(5+d[0], 5+d[1])
			w.grid[i] |= blockGround
		}
	}
	if at, _ := w.DropLanding(1, 5, 5); at != (CellPoint{X: 5, Y: 6}) {
		t.Errorf("only the sacked south is open: DropLanding = %v, want 5,6", at)
	}
	i, _ := w.cellIndex(5, 6)
	w.grid[i] |= blockGround
	if at, _ := w.DropLanding(1, 5, 5); at != (CellPoint{X: 5, Y: 5}) {
		t.Errorf("every neighbour closed: DropLanding = %v, want the dropper's own cell", at)
	}
}

// TestDropLandingRefusesAnAbsentOrOffMapDropper echoes the request.
func TestDropLandingRefusesAnAbsentOrOffMapDropper(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: -1, Y: 0}}, nil)
	for _, id := range []EntityID{1, 9} {
		if at, ok := w.DropLanding(id, 2, 3); ok || at != (CellPoint{X: 2, Y: 3}) {
			t.Errorf("DropLanding(%d) = %v, %v, want the request and false", id, at, ok)
		}
	}
}

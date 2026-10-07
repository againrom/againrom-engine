package sim

// The equipment channel (0124 T2): EquipSlots, the world's own per-entity
// record, the equip move KindEquip drives, and Equipped. Built through
// mustStockedWorld (carry_test.go), on that file's own precedent: nothing
// here needs a second constructor helper.
//
// Reaching the world only through Step and Equipped is deliberate: this file
// never writes w.equipment directly except where AC-4 needs a slot already
// occupied before the command that displaces it runs, which is the one shape
// no command built here can produce on its own.

import "testing"

func TestAFreshWorldEquipsEveryEntityAtEveryEmptySlot(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}}, nil)

	for _, id := range []EntityID{1, 2} {
		got, ok := w.Equipped(id)
		if !ok {
			t.Fatalf("Equipped(%d) answered not-ok for an entity this world holds", id)
		}
		for slot, code := range got {
			if code != 0 {
				t.Errorf("entity %d slot %d is %#x, want 0 — a fresh world equips nothing", id, slot+1, code)
			}
		}
	}

	if _, ok := w.Equipped(99); ok {
		t.Error("Equipped(99) answered ok for an entity this world does not hold")
	}
}

func TestEquippedHandsBackACopy(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{0x101}}})
	Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 1}})

	first, _ := w.Equipped(1)
	first[0] = 0xffff

	second, _ := w.Equipped(1)
	if second[0] != 0x101 {
		t.Errorf("mutating one call's result reached the world: Equipped(1) = %v", second)
	}
}

// TestEquipMovesTheNamedIndexIntoTheNamedSlotAndSwapsBack is AC-3 and AC-4
// — the spec's own two I/O examples, and one test rather than two because
// equip.go's own move is one function and not two arms (plan D-3): an
// entity holding [a, b, c] and an empty weapon slot, given the equip
// command for index 1 into slot 1, holds [a, c] with slot 1 = b (AC-3);
// the same world with slot 1 already holding d holds [a, d, c] with slot 1
// = b (AC-4) — the displaced code lands back at the very index the moved
// one vacated, which is what makes the two edges one function.
func TestEquipMovesTheNamedIndexIntoTheNamedSlotAndSwapsBack(t *testing.T) {
	const a, b, c, d = 0x101, 0x102, 0x103, 0x104

	t.Run("AC-3 empty slot", func(t *testing.T) {
		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
			[]Stock{{ID: 1, Items: []uint16{a, b, c}}})

		Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 1, Y: 1}})

		gotCarried, _ := w.Carried(1)
		if !equalCodes(gotCarried, []uint16{a, c}) {
			t.Errorf("Carried(1) = %#x, want [a c] = %#x", gotCarried, []uint16{a, c})
		}
		gotEquip, _ := w.Equipped(1)
		if gotEquip[0] != b {
			t.Errorf("slot 1 = %#x, want %#x", gotEquip[0], b)
		}
	})

	t.Run("AC-4 occupied slot", func(t *testing.T) {
		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
			[]Stock{{ID: 1, Items: []uint16{a, b, c}}})
		// The one direct write this file makes, and why it is the one shape
		// no command built here can produce on its own: AC-4 needs slot 1
		// already occupied BEFORE the equip that displaces it, and nothing
		// exported writes a slot except the command under test.
		w.equipment[0][0] = PlainItem(d)

		Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 1, Y: 1}})

		gotCarried, _ := w.Carried(1)
		if !equalCodes(gotCarried, []uint16{a, d, c}) {
			t.Errorf("Carried(1) = %#x, want [a d c] = %#x", gotCarried, []uint16{a, d, c})
		}
		gotEquip, _ := w.Equipped(1)
		if gotEquip[0] != b {
			t.Errorf("slot 1 = %#x, want %#x", gotEquip[0], b)
		}
	})
}

func TestEquipTouchesNoOtherSlotAndNoOtherEntity(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}},
		[]Stock{{ID: 1, Items: []uint16{0x101, 0x102}}, {ID: 2, Items: []uint16{0x201}}})

	Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 3}})

	got1, _ := w.Equipped(1)
	for slot, code := range got1 {
		if slot == 2 { // slot 3, index 2
			if code != 0x101 {
				t.Errorf("slot 3 = %#x, want 0x101", code)
			}
			continue
		}
		if code != 0 {
			t.Errorf("entity 1 slot %d is %#x, want 0 — only slot 3 was named", slot+1, code)
		}
	}
	got2, _ := w.Equipped(2)
	for slot, code := range got2 {
		if code != 0 {
			t.Errorf("entity 2 slot %d is %#x, want 0 — entity 2 was never named", slot+1, code)
		}
	}
	gotCarried2, _ := w.Carried(2)
	if !equalCodes(gotCarried2, []uint16{0x201}) {
		t.Errorf("Carried(2) = %#x, want unchanged [0x201]", gotCarried2)
	}
}

// eqNoopWorld is the fixture every AC-5 refusal is measured on: one entity
// holding three codes and an empty weapon slot, built through
// mustStockedWorld — a comparison over a world with nothing to move would
// agree for reasons that have nothing to do with the refusal (damage_test.go's
// dmgNoopWorld, restated for this command).
func eqNoopWorld(t *testing.T) *World {
	t.Helper()
	return mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{0x101, 0x102, 0x103}}})
}

func TestEquipRefusesAnAbsentEntityAnOutOfRangeIndexAndAnOutOfRangeSlot(t *testing.T) {
	cases := []struct {
		name string
		cmd  Command
	}{
		{"an entity the world does not hold", Command{Kind: KindEquip, Entity: 99, X: 0, Y: 1}},
		{"an index past the container", Command{Kind: KindEquip, Entity: 1, X: 3, Y: 1}},
		{"a negative index", Command{Kind: KindEquip, Entity: 1, X: -1, Y: 1}},
		{"a slot of 0", Command{Kind: KindEquip, Entity: 1, X: 0, Y: 0}},
		{"a slot past EquipSlots", Command{Kind: KindEquip, Entity: 1, X: 0, Y: EquipSlots + 1}},
		{"a negative slot", Command{Kind: KindEquip, Entity: 1, X: 0, Y: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := eqNoopWorld(t)
			Step(w, []Command{tc.cmd})

			quiet := eqNoopWorld(t)
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
	// above are not passing because Step ignores KindEquip outright
	// (damage_test.go's own control, restated for this command).
	live := eqNoopWorld(t)
	Step(live, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 1}})
	quiet := eqNoopWorld(t)
	Step(quiet, nil)
	if live.Hash() == quiet.Hash() {
		t.Errorf("an equip that lands leaves the same digest %#016x as a quiet tick — "+
			"the cases above then witness nothing", quiet.Hash())
	}
}

// ---------------------------------------------------------------- AC-6

// TestTwoWorldsDifferingOnlyInOneEquipmentSlotHashDifferently is AC-6's own
// clause on equipment: canonical, hashed state, so two worlds alike in
// everything else but one entity's one slot are two worlds, in the digest
// and in the bytes both.
func TestTwoWorldsDifferingOnlyInOneEquipmentSlotHashDifferently(t *testing.T) {
	a := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}}, nil)
	b := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}}, nil)
	b.equipment[0][6] = PlainItem(0x201)

	if a.Hash() == b.Hash() {
		t.Errorf("two worlds differing in one equipment slot both hash %#016x", a.Hash())
	}
	formA, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	formB, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if string(formA) == string(formB) {
		t.Error("two worlds differing in one equipment slot marshal to the same bytes")
	}
}

// TestAWorldHoldingEquipmentRoundTripsByteIdentically is AC-6's round-trip
// half: equipment an equip command wrote is ordinary canonical state, so a
// world holding it marshals, unmarshals and re-marshals to the same bytes
// and hashes the same before and after — TakeSack's own round-trip witness
// (carry_test.go), restated for this command.
func TestAWorldHoldingEquipmentRoundTripsByteIdentically(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{0x101, 0x102, 0x103}}})
	Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 1, Y: 1}})

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
		t.Fatal("a world holding equipment does not round-trip through its byte form")
	}
	if back.Hash() != w.Hash() {
		t.Fatal("a round-tripped world holding equipment hashes differently from the original")
	}
}

// ---------------------------------------------------------------- 0151, defect 4 (unequip)

// TestUnequipMovesTheSlotsCodeIntoTheContainer is the move's own two shapes:
// the code lands as a fresh element at the container's tail when the
// container holds nothing of that code yet, and merges into the existing
// element when it does — foldContainer's own two outcomes, exercised
// through the command rather than asserted on the helper directly.
func TestUnequipMovesTheSlotsCodeIntoTheContainer(t *testing.T) {
	const a, b = 0x101, 0x102

	t.Run("fresh element", func(t *testing.T) {
		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
			[]Stock{{ID: 1, Items: []uint16{a}}})
		w.equipment[0][0] = PlainItem(b) // slot 1 holds b; the command below names slot 1

		Step(w, []Command{{Kind: KindUnequip, Entity: 1, X: 1}})

		gotEquip, _ := w.Equipped(1)
		if gotEquip[0] != 0 {
			t.Errorf("slot 1 = %#x, want 0 (empty)", gotEquip[0])
		}
		gotCarried, _ := w.Carried(1)
		if !equalCodes(gotCarried, []uint16{a, b}) {
			t.Errorf("Carried(1) = %#x, want [a b] = %#x", gotCarried, []uint16{a, b})
		}
	})

	t.Run("merges into an existing element", func(t *testing.T) {
		w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
			[]Stock{{ID: 1, Items: []uint16{a}}})
		w.equipment[0][0] = PlainItem(a) // the worn code already has an element in the container

		Step(w, []Command{{Kind: KindUnequip, Entity: 1, X: 1}})

		gotEquip, _ := w.Equipped(1)
		if gotEquip[0] != 0 {
			t.Errorf("slot 1 = %#x, want 0 (empty)", gotEquip[0])
		}
		stacks, _ := w.CarriedStacks(1)
		if len(stacks) != 1 || stacks[0].Code != a || stacks[0].Count != 2 {
			t.Errorf("CarriedStacks(1) = %+v, want one element {%#x 2} — the code folded rather than "+
				"taking a second place", stacks, a)
		}
	})
}

func TestUnequippingWeaponLeavesTheShieldWorn(t *testing.T) {
	weapon := ItemInstance{Code: 0x0101, Kind: 2, Price: 411,
		Effects: []ItemEffect{{Kind: 12, Operand: 3}}}
	shield := ItemInstance{Code: 0x0201, Kind: 1, Price: 722,
		Effects: []ItemEffect{{Kind: 15, Operand: 9}}}
	var worn [EquipSlots]ItemInstance
	worn[0], worn[1] = weapon, shield
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20}},
		[]Stock{{ID: 1, EquippedItems: worn}})

	Step(w, []Command{{Kind: KindUnequip, Entity: 1, X: 1}})

	gotWorn, _ := w.EquippedItems(1)
	gotPack, _ := w.CarriedItems(1)
	if !gotWorn[0].Empty() || !ItemEqual(gotWorn[1], shield) || gotWorn[1].Price != shield.Price ||
		len(gotPack) != 1 || !ItemEqual(gotPack[0], weapon) || gotPack[0].Price != weapon.Price {
		t.Fatalf("weapon unequip left worn=%+v pack=%+v, want the complete shield worn and the complete weapon in the pack", gotWorn, gotPack)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	backWorn, _ := back.EquippedItems(1)
	if back.Hash() != w.Hash() || !ItemEqual(backWorn[1], shield) || !backWorn[0].Empty() {
		t.Fatalf("round trip lost the shield worn alone: worn=%+v", backWorn)
	}
}

func TestEquippingAShieldWithNoWeaponWearsItAlone(t *testing.T) {
	shield := ItemInstance{Code: 0x0201, Kind: 1, Price: 701,
		Effects: []ItemEffect{{Kind: 15, Operand: 4}}}
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20}},
		[]Stock{{ID: 1, ItemInstances: []ItemInstance{shield}}})

	Step(w, []Command{Equip(1, 0, 2)})

	gotWorn, _ := w.EquippedItems(1)
	gotPack, _ := w.CarriedItems(1)
	if !gotWorn[0].Empty() || !ItemEqual(gotWorn[1], shield) || gotWorn[1].Price != shield.Price || len(gotPack) != 0 {
		t.Fatalf("shield equip left worn=%+v pack=%+v, want the complete shield worn alone and an empty pack", gotWorn, gotPack)
	}
}

func TestUnequipTouchesNoOtherSlotAndNoOtherEntity(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}},
		[]Stock{{ID: 1, Items: nil}, {ID: 2, Items: []uint16{0x201}}})
	w.equipment[0][2] = PlainItem(0x301) // entity 1, slot 3
	w.equipment[0][5] = PlainItem(0x302) // entity 1, slot 6 — must survive untouched

	Step(w, []Command{{Kind: KindUnequip, Entity: 1, X: 3}})

	got1, _ := w.Equipped(1)
	for slot, code := range got1 {
		switch slot {
		case 2: // slot 3, just cleared
			if code != 0 {
				t.Errorf("slot 3 = %#x, want 0", code)
			}
		case 5: // slot 6, untouched
			if code != 0x302 {
				t.Errorf("slot 6 = %#x, want 0x302 (untouched)", code)
			}
		default:
			if code != 0 {
				t.Errorf("entity 1 slot %d is %#x, want 0 — only slot 3 was named", slot+1, code)
			}
		}
	}
	gotCarried, _ := w.Carried(1)
	if !equalCodes(gotCarried, []uint16{0x301}) {
		t.Errorf("Carried(1) = %#x, want [0x301]", gotCarried)
	}
	got2, _ := w.Equipped(2)
	for slot, code := range got2 {
		if code != 0 {
			t.Errorf("entity 2 slot %d is %#x, want 0 — entity 2 was never named", slot+1, code)
		}
	}
	gotCarried2, _ := w.Carried(2)
	if !equalCodes(gotCarried2, []uint16{0x201}) {
		t.Errorf("Carried(2) = %#x, want unchanged [0x201]", gotCarried2)
	}
}

// unNoopWorld is the fixture every unequip refusal is measured on: one
// entity carrying one code and wearing one item at slot 1, built through
// mustStockedWorld — eqNoopWorld's own shape (above), widened with an
// occupied slot so "the slot named was already empty" has a real slot to
// contrast against.
func unNoopWorld(t *testing.T) *World {
	t.Helper()
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}},
		[]Stock{{ID: 1, Items: []uint16{0x101}}})
	w.equipment[0][0] = PlainItem(0x201) // slot 1 occupied; slot 2 and past EquipSlots stay empty
	return w
}

// TestUnequipRefusesAnAbsentEntityAnOutOfRangeSlotAndAnAlreadyEmptySlot is
// equip's own AC-5, restated for its inverse: each case leaves the world
// byte-identical to a tick carrying no command at all, over the WHOLE byte
// form — carried and equipment are not among snap()'s pinned fields
// (world_test.go), so only the marshalled bytes see either move.
func TestUnequipRefusesAnAbsentEntityAnOutOfRangeSlotAndAnAlreadyEmptySlot(t *testing.T) {
	cases := []struct {
		name string
		cmd  Command
	}{
		{"an entity the world does not hold", Command{Kind: KindUnequip, Entity: 99, X: 1}},
		{"a slot of 0", Command{Kind: KindUnequip, Entity: 1, X: 0}},
		{"a slot past EquipSlots", Command{Kind: KindUnequip, Entity: 1, X: EquipSlots + 1}},
		{"a negative slot", Command{Kind: KindUnequip, Entity: 1, X: -1}},
		{"a slot already empty", Command{Kind: KindUnequip, Entity: 1, X: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := unNoopWorld(t)
			Step(w, []Command{tc.cmd})

			quiet := unNoopWorld(t)
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
	// above are not passing because Step ignores KindUnequip outright.
	live := unNoopWorld(t)
	Step(live, []Command{{Kind: KindUnequip, Entity: 1, X: 1}})
	quiet := unNoopWorld(t)
	Step(quiet, nil)
	if live.Hash() == quiet.Hash() {
		t.Errorf("an unequip that lands leaves the same digest %#016x as a quiet tick — "+
			"the cases above then witness nothing", quiet.Hash())
	}
}

// TestAWorldHoldingAnUnequipRoundTripsByteIdentically is the round-trip half
// (equip's own TestAWorldHoldingEquipmentRoundTripsByteIdentically,
// restated for the inverse move): the slot cleared and the code folded back
// into the container are ordinary canonical state, so a world holding the
// result marshals, unmarshals and re-marshals to the same bytes and hashes
// the same before and after.
func TestAWorldHoldingAnUnequipRoundTripsByteIdentically(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}}, nil)
	w.equipment[0][0] = PlainItem(0x101)
	Step(w, []Command{{Kind: KindUnequip, Entity: 1, X: 1}})

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
		t.Fatal("a world holding an unequip's result does not round-trip through its byte form")
	}
	if back.Hash() != w.Hash() {
		t.Fatal("a round-tripped world holding an unequip's result hashes differently from the original")
	}
}

func TestEquipAtomicallyDisplacesTheOtherHeldSlotWithCompleteInstances(t *testing.T) {
	shield := ItemInstance{Code: 0x0201, Kind: 1, Price: 701,
		Effects: []ItemEffect{{Kind: 15, Operand: 4}}}
	weapon := ItemInstance{Code: 0x0101, Kind: 2, Price: 902,
		Effects: []ItemEffect{{Kind: 17, Operand: 3}}}
	var worn [EquipSlots]ItemInstance
	worn[0] = weapon
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20}},
		[]Stock{{ID: 1, ItemInstances: []ItemInstance{shield}, EquippedItems: worn}})

	Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 2, Spell: 1}})

	gotWorn, _ := w.EquippedItems(1)
	if !gotWorn[0].Empty() || !ItemEqual(gotWorn[1], shield) || gotWorn[1].Price != shield.Price {
		t.Fatalf("equipped = %+v, want empty weapon and complete shield %+v", gotWorn, shield)
	}
	gotPack, _ := w.CarriedItems(1)
	if len(gotPack) != 1 || !ItemEqual(gotPack[0], weapon) || gotPack[0].Price != weapon.Price {
		t.Fatalf("pack = %+v, want complete displaced weapon %+v", gotPack, weapon)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Fatalf("round-trip hash = %#x, want %#x", back.Hash(), w.Hash())
	}
	backWorn, _ := back.EquippedItems(1)
	backPack, _ := back.CarriedItems(1)
	if !ItemEqual(backWorn[1], shield) || len(backPack) != 1 || !ItemEqual(backPack[0], weapon) {
		t.Fatalf("round trip lost instances: worn=%+v pack=%+v", backWorn, backPack)
	}
}

func TestEquipRefusesAnInvalidSecondSlotWithoutMovingEitherItem(t *testing.T) {
	shield := ItemInstance{Code: 0x0201, Kind: 1, Price: 701,
		Effects: []ItemEffect{{Kind: 15, Operand: 4}}}
	weapon := ItemInstance{Code: 0x0101, Kind: 2, Price: 902,
		Effects: []ItemEffect{{Kind: 17, Operand: 3}}}
	build := func() *World {
		var worn [EquipSlots]ItemInstance
		worn[0] = weapon
		return mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20}},
			[]Stock{{ID: 1, ItemInstances: []ItemInstance{shield}, EquippedItems: worn}})
	}
	live, quiet := build(), build()
	Step(live, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 2, Spell: 2}}) // target and displacement coincide
	Step(quiet, nil)
	if live.Hash() != quiet.Hash() {
		t.Fatalf("invalid atomic destination changed hash %#x -> %#x", quiet.Hash(), live.Hash())
	}
	liveForm, _ := live.MarshalBinary()
	quietForm, _ := quiet.MarshalBinary()
	if string(liveForm) != string(quietForm) {
		t.Fatal("invalid atomic destination changed the canonical byte form")
	}
}

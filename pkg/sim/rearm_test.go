package sim

import "testing"

// rcEnt is one entity for these tests: an id, a cell, a health pair and every
// combat field named at a distinct value, so a bug that copies the wrong
// field or the wrong entity shows up as a wrong number rather than a
// coincidental match.
func rcEnt(id EntityID, x, y int32, dmgBase, dmgSpread, toHit, def, absorb, charge, relax int32, always bool, reach uint8) Entity {
	return Entity{
		ID: id, X: x, Y: y, HP: 40, MaxHP: 80, Speed: 3, Reach: reach,
		DamageBase: dmgBase, DamageSpread: dmgSpread, ToHit: toHit,
		Defence: def, Absorption: absorb,
		AttackCharge: charge, AttackRelax: relax, AlwaysHits: always,
	}
}

// rcWorld builds a world over an open 4x4 grid, failing the test rather than
// returning an error, so a case reads as its own statement.
func rcWorld(t *testing.T, ents ...Entity) *World {
	t.Helper()
	w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

// rcAt is the world's entry for id, failing when the world holds none.
func rcAt(t *testing.T, w *World, id EntityID) Entity {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		t.Fatalf("world holds no entity %d", id)
	}
	return w.entities[i]
}

// TestSetCombatWitnessesAC12 is AC-12's own witness: writing a recompute's
// result onto a live entity changes the complete combat block, including its
// resistance family, and changes it on that entity only. Entity 1 is the one SetCombat is
// called on; entity 2 stands beside it, untouched, in the same world; id 99
// names no entity in either world built here.
func TestSetCombatWitnessesAC12(t *testing.T) {
	t.Run("the named entity's ten fields move to the block's values, and nothing else on it does", func(t *testing.T) {
		e1 := rcEnt(1, 1, 1, 1, 2, 3, 4, 5, 6, 7, false, 3)
		e1.XPSlot = 1
		e1.Resistance = [5]uint8{1, 2, 3, 4, 5}
		e2 := rcEnt(2, 2, 2, 11, 12, 13, 14, 15, 16, 17, true, 4)
		w := rcWorld(t, e1, e2)

		before1 := rcAt(t, w, 1)

		block := CombatBlock{
			DamageBase: 100, DamageSpread: 101,
			ToHit: 102, Defence: 103, Absorption: 104,
			AttackCharge: 105, AttackRelax: 106,
			AlwaysHits: true, Reach: 9,
			// A DIFFERENT SLOT FROM THE ONE THE ENTITY CARRIES (1), so this
			// case discriminates a setter that writes the tenth field from
			// one that leaves it alone — the whole of the hotfix that added
			// it. With both at the same value the comparison below passes
			// either way.
			XPSlot:     3,
			Resistance: [5]uint8{10, 20, 30, 40, 255},
		}
		if ok := w.SetCombat(1, block); !ok {
			t.Fatal("SetCombat(1, block) = false, want true")
		}

		after1 := rcAt(t, w, 1)
		want1 := before1
		want1.DamageBase, want1.DamageSpread = block.DamageBase, block.DamageSpread
		want1.ToHit, want1.Defence, want1.Absorption = block.ToHit, block.Defence, block.Absorption
		want1.AttackCharge, want1.AttackRelax = block.AttackCharge, block.AttackRelax
		want1.AlwaysHits = block.AlwaysHits
		want1.Reach = block.Reach
		want1.XPSlot = block.XPSlot
		want1.Resistance = block.Resistance
		if after1 != want1 {
			t.Errorf("entity 1 after SetCombat = %+v, want %+v", after1, want1)
		}
		// Named on its own as well as inside the struct compare: the compare
		// says "some field is wrong", this says WHICH VALUE the credited slot
		// should hold.
		if after1.XPSlot != 3 {
			t.Errorf("XPSlot = %d, want 3 — the block's own slot, not the entity's starting 1", after1.XPSlot)
		}
	})

	// The guard, and it is ALL-OR-NOTHING rather than "the slot is dropped":
	// a block whose credited slot names no integer in SkillXP writes NONE of
	// the ten, so the entity keeps the whole combat block it was fighting
	// with. skillSlots is 6, so 6 is the first refused value and 255 is what
	// a negative attack type narrows to through uint8 — the shape the shipped
	// Weapons table's own attack-type -1 row produces.
	//
	// TO CONFIRM THIS TEST WITNESSES THE GUARD, delete the experienceFault
	// check in SetCombat (rearm.go) and rerun: DamageBase reddens at 999 and
	// XPSlot at the refused value, and payExperience would then index
	// SkillXP[255].
	t.Run("a credited slot outside 0..5 writes nothing at all and answers false", func(t *testing.T) {
		for _, slot := range []uint8{6, 255} {
			e1 := rcEnt(1, 1, 1, 1, 2, 3, 4, 5, 6, 7, false, 3)
			e1.XPSlot = 2
			e2 := rcEnt(2, 2, 2, 11, 12, 13, 14, 15, 16, 17, true, 4)
			w := rcWorld(t, e1, e2)

			before1 := rcAt(t, w, 1)
			hashBefore := w.Hash()

			if ok := w.SetCombat(1, CombatBlock{DamageBase: 999, Reach: 9, XPSlot: slot,
				Resistance: [5]uint8{9, 9, 9, 9, 9}}); ok {
				t.Errorf("SetCombat(1, {XPSlot: %d}) = true, want false — only 0..5 name a slot", slot)
			}

			after1 := rcAt(t, w, 1)
			if after1 != before1 {
				t.Errorf("XPSlot %d: entity 1 changed: before %+v, after %+v", slot, before1, after1)
			}
			if after1.XPSlot != 2 {
				t.Errorf("XPSlot %d: credited slot = %d, want 2 — the one the entity already held", slot, after1.XPSlot)
			}
			if after1.DamageBase != 1 {
				t.Errorf("XPSlot %d: DamageBase = %d, want 1 — a refused block writes none of the ten, not nine of them",
					slot, after1.DamageBase)
			}
			if got := w.Hash(); got != hashBefore {
				t.Errorf("XPSlot %d: hash moved from %d to %d on a refused block", slot, hashBefore, got)
			}
		}
	})

	// The window's own edges, asserted rather than assumed: 0 and 5 are legal
	// and are written, so the guard above refuses exactly the values outside
	// the six and no more.
	t.Run("slots 0 and 5 are written", func(t *testing.T) {
		for _, slot := range []uint8{0, 5} {
			e1 := rcEnt(1, 1, 1, 1, 2, 3, 4, 5, 6, 7, false, 3)
			e1.XPSlot = 3
			w := rcWorld(t, e1)
			if ok := w.SetCombat(1, CombatBlock{DamageBase: 8, Reach: 1, XPSlot: slot}); !ok {
				t.Fatalf("SetCombat(1, {XPSlot: %d}) = false, want true", slot)
			}
			if got := rcAt(t, w, 1).XPSlot; got != slot {
				t.Errorf("XPSlot = %d, want %d", got, slot)
			}
		}
	})

	t.Run("a second entity in the same world is untouched, field for field", func(t *testing.T) {
		e1 := rcEnt(1, 1, 1, 1, 2, 3, 4, 5, 6, 7, false, 3)
		e2 := rcEnt(2, 2, 2, 11, 12, 13, 14, 15, 16, 17, true, 4)
		w := rcWorld(t, e1, e2)

		before2 := rcAt(t, w, 2)
		if ok := w.SetCombat(1, CombatBlock{DamageBase: 100, Reach: 9}); !ok {
			t.Fatal("SetCombat(1, ...) = false, want true")
		}
		after2 := rcAt(t, w, 2)
		if after2 != before2 {
			t.Errorf("entity 2 changed: before %+v, after %+v", before2, after2)
		}
	})

	t.Run("an unknown id answers false and leaves the world's hash unchanged", func(t *testing.T) {
		e1 := rcEnt(1, 1, 1, 1, 2, 3, 4, 5, 6, 7, false, 3)
		e2 := rcEnt(2, 2, 2, 11, 12, 13, 14, 15, 16, 17, true, 4)
		w := rcWorld(t, e1, e2)

		before1, before2 := rcAt(t, w, 1), rcAt(t, w, 2)
		hashBefore := w.Hash()

		if ok := w.SetCombat(99, CombatBlock{DamageBase: 999, Reach: 255}); ok {
			t.Error("SetCombat(99, ...) = true, want false — the world holds no id 99")
		}

		hashAfter := w.Hash()
		if hashAfter != hashBefore {
			t.Errorf("hash moved from %d to %d after a call on an unknown id", hashBefore, hashAfter)
		}
		if got := rcAt(t, w, 1); got != before1 {
			t.Errorf("entity 1 changed on a call naming id 99: before %+v, after %+v", before1, got)
		}
		if got := rcAt(t, w, 2); got != before2 {
			t.Errorf("entity 2 changed on a call naming id 99: before %+v, after %+v", before2, got)
		}
	})
}

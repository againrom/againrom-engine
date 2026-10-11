package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// swapUnits are two party members, 7 and 8, each standing on its own cell
// holding its own worn pair and pack.
func swapWorld(t *testing.T, worn [2][sim.EquipSlots]sim.ItemInstance, pack [2][]sim.ItemInstance) *sim.World {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 20, MaxHP: 20}, {ID: 8, X: 5, Y: 3, HP: 20, MaxHP: 20}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, ItemInstances: pack[0], EquippedItems: worn[0]}, {ID: 8, ItemInstances: pack[1], EquippedItems: worn[1]}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

// swapMission opens the two-member mission; 7 is the inventory subject.
func swapMission(t *testing.T, w *sim.World, table *mapload.Table) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{Hero: eqHero()}, {Hero: eqHero()}},
		Start: mapload.Start{IDs: []sim.EntityID{7, 8}}}
	return openMission(ms, table, nil, v, missionSource{}, nil, nil)
}

// swapFrames runs n frames of the production frame path.
func swapFrames(mw *mapWorld, n int) {
	for i := 0; i < n; i++ {
		mw.deterministicFrame()
	}
}

// swapEquip wears pack element idx of the subject through the inventory's own
// equip and runs the frames that apply it.
func swapEquip(t *testing.T, mw *mapWorld, idx int) {
	t.Helper()
	mw.enqueueEquip(idx)
	if len(mw.pending) == 0 {
		t.Fatalf("the inventory queued no equip for element %d", idx)
	}
	swapFrames(mw, 2)
}

func swapHeld(t *testing.T, w *sim.World, id sim.EntityID) (weapon, shield uint16) {
	t.Helper()
	worn, ok := w.EquippedItems(id)
	if !ok {
		t.Fatalf("entity %d is gone", id)
	}
	return worn[0].Code, worn[1].Code
}

func swapPackCodes(w *sim.World, id sim.EntityID) map[uint16]int {
	stacks, _ := w.CarriedStacks(id)
	out := map[uint16]int{}
	for _, st := range stacks {
		out[st.Code] += int(st.Count)
	}
	return out
}

// The swap returns the previous weapon with the shield that went with it and
// sends the current pair to the pack, both ways. Rows: a two-handed weapon and
// a bow, each displacing the shield when worn; a one-handed weapon with no
// shield before it. The loss control is the first press, made before any
// weapon change, which queues nothing.
func TestWeaponSwapRestoresThePreviousPair(t *testing.T) {
	table := shieldWearTable(t)
	sword, shield := shieldWearWeapon(eqSwordCode), shieldWearShield()
	for _, tc := range []struct {
		name       string
		other      uint16
		withShield bool
	}{
		{"sword and shield, then a two-handed sword", swGreatswordCode, true},
		{"sword and shield, then a bow", swLongbowCode, true},
		{"sword alone, then a two-handed sword", swGreatswordCode, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var worn [2][sim.EquipSlots]sim.ItemInstance
			worn[0][0] = sword
			wantShield := uint16(0)
			if tc.withShield {
				worn[0][1], wantShield = shield, eqShieldCode
			}
			w := swapWorld(t, worn, [2][]sim.ItemInstance{{shieldWearWeapon(tc.other)}, nil})
			mw := swapMission(t, w, table)
			swapFrames(mw, 1)

			mw.swapWeapons([]uint32{7})
			if len(mw.pending) != 0 {
				t.Fatalf("control: a swap before any weapon change queued %+v", mw.pending)
			}

			swapEquip(t, mw, 0)
			if wp, sh := swapHeld(t, w, 7); wp != tc.other || sh != 0 {
				t.Fatalf("setup: holds %#x/%#x after the inventory equip, want %#x/0", wp, sh, tc.other)
			}

			mw.swapWeapons([]uint32{7})
			swapFrames(mw, 4)
			if wp, sh := swapHeld(t, w, 7); wp != eqSwordCode || sh != wantShield {
				t.Fatalf("after the swap holds %#x/%#x, want %#x/%#x", wp, sh, eqSwordCode, wantShield)
			}
			if p := swapPackCodes(w, 7); p[tc.other] != 1 || p[eqShieldCode] != 0 || p[eqSwordCode] != 0 {
				t.Fatalf("after the swap the pack holds %v, want only %#x", p, tc.other)
			}

			mw.swapWeapons([]uint32{7})
			swapFrames(mw, 4)
			if wp, sh := swapHeld(t, w, 7); wp != tc.other || sh != 0 {
				t.Fatalf("after the swap back holds %#x/%#x, want %#x/0", wp, sh, tc.other)
			}
			if p := swapPackCodes(w, 7); p[eqSwordCode] != 1 || p[eqShieldCode] != map[bool]int{true: 1}[tc.withShield] {
				t.Fatalf("after the swap back the pack holds %v", p)
			}
		})
	}
}

// A previous weapon that left the pack is not taken back, and the current
// shield stays; the control is the same unit with the weapon still packed.
func TestWeaponSwapNeedsThePreviousWeaponInThePack(t *testing.T) {
	table := shieldWearTable(t)
	for _, dropped := range []bool{false, true} {
		var worn [2][sim.EquipSlots]sim.ItemInstance
		worn[0][0] = shieldWearWeapon(swGreatswordCode)
		w := swapWorld(t, worn, [2][]sim.ItemInstance{{shieldWearWeapon(eqSwordCode)}, nil})
		mw := swapMission(t, w, table)
		swapFrames(mw, 1)
		swapEquip(t, mw, 0)
		if dropped {
			mw.pending = append(mw.pending, sim.DropCarried(7, 0, sim.CellPoint{X: 3, Y: 4}))
			swapFrames(mw, 1)
			if p := swapPackCodes(w, 7); p[swGreatswordCode] != 0 {
				t.Fatalf("setup: the drop left %v in the pack", p)
			}
		}
		mw.swapWeapons([]uint32{7})
		if got := len(mw.pending) != 0; got == dropped {
			t.Errorf("dropped=%t: the swap queued %+v", dropped, mw.pending)
		}
	}
}

// With two units selected each swaps its own pair. The unit that is not the
// inventory subject ends in the same world, hash for hash, as when it is the
// subject and the player wears the same weapon from its inventory, and as when
// the same orders are queued bare or by its inventory before the selection
// moves. The loss control steps the same orders on the World alone, with no
// re-derivation, which leaves a different world.
func TestWeaponSwapOfSeveralUnitsMatchesTheInventoryEquip(t *testing.T) {
	table := shieldWearTable(t)
	start := func() (*sim.World, *mapWorld) {
		var worn [2][sim.EquipSlots]sim.ItemInstance
		worn[0][0] = shieldWearWeapon(eqSwordCode)
		worn[1][0], worn[1][1] = shieldWearWeapon(eqSwordCode), shieldWearShield()
		w := swapWorld(t, worn, [2][]sim.ItemInstance{{shieldWearWeapon(swGreatswordCode)}, {shieldWearWeapon(swGreatswordCode)}})
		mw := swapMission(t, w, table)
		swapFrames(mw, 1)
		swapEquip(t, mw, 0)
		mw.switchInventorySubject(8)
		swapEquip(t, mw, 0)
		return w, mw
	}

	swapped, mw := start()
	mw.switchInventorySubject(7)
	mw.swapWeapons([]uint32{7, 8})
	swapFrames(mw, 6)
	for id, want := range map[sim.EntityID][2]uint16{7: {eqSwordCode, 0}, 8: {eqSwordCode, eqShieldCode}} {
		if wp, sh := swapHeld(t, swapped, id); wp != want[0] || sh != want[1] {
			t.Fatalf("unit %d holds %#x/%#x after the swap, want %#x/%#x", id, wp, sh, want[0], want[1])
		}
	}
	if mw.invSubject.ID != 7 {
		t.Errorf("the swap left unit %d the inventory subject, want 7", mw.invSubject.ID)
	}

	// The same two commands for unit 8 on the same two ticks: by the subject's
	// own inventory, then queued bare with unit 7 the subject.
	equipBy := func(mw *mapWorld, w *sim.World, code uint16, slot sim.EquipSlot, inventory bool) {
		stacks, _ := w.CarriedStacks(8)
		for i, st := range stacks {
			if st.Code != code {
				continue
			}
			if inventory {
				mw.enqueueEquip(i)
			} else {
				mw.pending = append(mw.pending, sim.EquipDisplacing(8, sim.ItemSlot(i), slot, 0))
			}
			return
		}
		t.Fatalf("unit 8's pack holds no %#x", code)
	}
	byHand, hw := start()
	hw.switchInventorySubject(7)
	hw.swapWeapons([]uint32{7})
	hw.switchInventorySubject(8)
	equipBy(hw, byHand, eqSwordCode, 1, true)
	swapFrames(hw, 1)
	equipBy(hw, byHand, eqShieldCode, 2, true)
	swapFrames(hw, 1)
	hw.switchInventorySubject(7)
	swapFrames(hw, 4)
	if swapped.Hash() != byHand.Hash() {
		t.Errorf("swap world %#x differs from the inventory equip world %#x", swapped.Hash(), byHand.Hash())
	}
	a8, _ := swapped.Entity(8)
	b8, _ := byHand.Entity(8)
	if !reflect.DeepEqual(a8, b8) {
		t.Errorf("unit 8 after the swap = %+v, after the inventory equip = %+v", a8, b8)
	}

	bare, bw := start()
	bw.switchInventorySubject(7)
	bw.swapWeapons([]uint32{7})
	equipBy(bw, bare, eqSwordCode, 1, false)
	swapFrames(bw, 1)
	equipBy(bw, bare, eqShieldCode, 2, false)
	swapFrames(bw, 5)
	if bare.Hash() != byHand.Hash() {
		t.Errorf("bare queued orders world %#x differs from the inventory equip world %#x", bare.Hash(), byHand.Hash())
	}

	// The inventory's own equip, queued while unit 8 is the subject, then the
	// selection moves to unit 7 before the tick applies it.
	moved, mvw := start()
	mvw.switchInventorySubject(7)
	mvw.swapWeapons([]uint32{7})
	mvw.switchInventorySubject(8)
	equipBy(mvw, moved, eqSwordCode, 1, true)
	mvw.switchInventorySubject(7)
	swapFrames(mvw, 1)
	mvw.switchInventorySubject(8)
	equipBy(mvw, moved, eqShieldCode, 2, true)
	mvw.switchInventorySubject(7)
	swapFrames(mvw, 5)
	if moved.Hash() != byHand.Hash() {
		t.Errorf("inventory orders applied after the selection moved: world %#x, want %#x", moved.Hash(), byHand.Hash())
	}

	raw, rw := start()
	rw.switchInventorySubject(7)
	rw.swapWeapons([]uint32{7})
	swapFrames(rw, 1)
	for _, c := range []struct {
		code uint16
		slot sim.EquipSlot
	}{{eqSwordCode, 1}, {eqShieldCode, 2}} {
		stacks, _ := raw.CarriedStacks(8)
		i := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == c.code })
		sim.Step(raw, []sim.Command{sim.EquipDisplacing(8, sim.ItemSlot(i), c.slot, 0)})
	}
	swapFrames(rw, 4)
	if wp, sh := swapHeld(t, raw, 8); wp != eqSwordCode || sh != eqShieldCode {
		t.Fatalf("control: the World alone left unit 8 holding %#x/%#x", wp, sh)
	}
	if raw.Hash() == byHand.Hash() {
		t.Error("control: orders stepped without the re-derivation left the same world, so the comparison above separates nothing")
	}
}

// A SAV written while a swap's order of a unit that is not the inventory
// subject is still queued holds that order, and the reopened mission applies
// it to the same world, hash for hash: the re-derivation follows the pending
// orders, which the SAV holds, and no App state. The loss control steps the
// reopened World with the same order and no re-derivation.
func TestWeaponSwapQueuedBeforeASaveAppliesAfterTheLoad(t *testing.T) {
	table := shieldWearTable(t)
	var worn [2][sim.EquipSlots]sim.ItemInstance
	worn[1][0] = shieldWearWeapon(eqSwordCode)
	mw := swapMission(t, swapWorld(t, worn, [2][]sim.ItemInstance{nil, {shieldWearWeapon(swGreatswordCode)}}), table)
	swapFrames(mw, 1)
	mw.switchInventorySubject(8)
	swapEquip(t, mw, 0)
	mw.switchInventorySubject(7)
	mw.swapWeapons([]uint32{8})
	if len(mw.pending) != 1 || mw.pending[0].Entity != 8 {
		t.Fatalf("setup: W queued %+v, want one order of unit 8", mw.pending)
	}
	raw := marshalWorld(t, mw.world)
	reopen := func() *mapWorld {
		w := new(sim.World)
		if err := w.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		cold := swapMission(t, w, table)
		cold.applyResidue(mw.residue())
		return cold
	}
	cold := reopen()
	if len(cold.pending) != 1 || len(cold.weaponSwap.plans) != 0 {
		t.Fatalf("setup: the reopened mission holds orders %+v and plans %+v", cold.pending, cold.weaponSwap.plans)
	}
	control := reopen()
	swapFrames(mw, 1)
	swapFrames(cold, 1)
	sim.Step(control.world, control.pending)
	a, _ := mw.world.Entity(8)
	b, _ := cold.world.Entity(8)
	c, _ := control.world.Entity(8)
	if mw.world.Hash() != cold.world.Hash() || !reflect.DeepEqual(a, b) {
		t.Errorf("after the LOAD the order gives world %#x damage %d/%d, uninterrupted %#x damage %d/%d",
			cold.world.Hash(), b.DamageBase, b.DamageSpread, mw.world.Hash(), a.DamageBase, a.DamageSpread)
	}
	if control.world.Hash() == mw.world.Hash() {
		t.Error("control: the order without the re-derivation gave the same world, so the comparison separates nothing")
	}
	t.Logf("world %#x, damage %d/%d; control world %#x, damage %d/%d", mw.world.Hash(), a.DamageBase, a.DamageSpread,
		control.world.Hash(), c.DamageBase, c.DamageSpread)
}

// W pressed after a queued drop of the remembered weapon waits for the drop
// and then finds the weapon gone, so it queues nothing and slot 1 keeps what it
// held, as with no drop. The control queues a drop of another pack element:
// W still takes the remembered weapon back once the drop has applied.
func TestWeaponSwapWaitsForAQueuedDrop(t *testing.T) {
	table := shieldWearTable(t)
	for _, tc := range []struct {
		name       string
		dropWanted bool
		want       uint16
	}{
		{"the remembered weapon dropped", true, eqSwordCode},
		{"control: another element dropped", false, swGreatswordCode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var worn [2][sim.EquipSlots]sim.ItemInstance
			worn[0][0] = shieldWearWeapon(swGreatswordCode)
			w := swapWorld(t, worn, [2][]sim.ItemInstance{{shieldWearWeapon(eqSwordCode), shieldWearShield()}, nil})
			mw := swapMission(t, w, table)
			swapFrames(mw, 1)
			swapEquip(t, mw, 0)
			prev := mw.packIndex(7, mw.weaponSwap.previous[7].weapon)
			if prev != 0 {
				t.Fatalf("setup: the remembered weapon is pack element %d, want 0", prev)
			}
			drop := 1
			if tc.dropWanted {
				drop = prev
			}
			mw.pending = append(mw.pending, sim.DropCarried(7, sim.ItemSlot(drop), sim.CellPoint{X: 3, Y: 4}))
			mw.swapWeapons([]uint32{7})
			if len(mw.pending) != 1 {
				t.Fatalf("W queued %+v behind the pending drop, want nothing until it applies", mw.pending)
			}
			swapFrames(mw, 4)
			if wp, _ := swapHeld(t, w, 7); wp != tc.want {
				t.Errorf("slot 1 holds %#x, want %#x; pack %v", wp, tc.want, swapPackCodes(w, 7))
			}
		})
	}
}

// The memory is App state: a mission opened again over the same world, as a
// LOAD opens one, has no previous weapon to take back.
func TestWeaponSwapMemoryIsNotCarriedIntoANewMission(t *testing.T) {
	table := shieldWearTable(t)
	var worn [2][sim.EquipSlots]sim.ItemInstance
	worn[0][0] = shieldWearWeapon(eqSwordCode)
	w := swapWorld(t, worn, [2][]sim.ItemInstance{{shieldWearWeapon(swGreatswordCode)}, nil})
	mw := swapMission(t, w, table)
	swapFrames(mw, 1)
	swapEquip(t, mw, 0)
	mw.swapWeapons([]uint32{7})
	if len(mw.pending) == 0 {
		t.Fatal("control: the running mission queued no swap")
	}
	again := swapMission(t, w, table)
	swapFrames(again, 1)
	again.swapWeapons([]uint32{7})
	if len(again.pending) != 0 {
		t.Errorf("a reopened mission queued %+v", again.pending)
	}
}

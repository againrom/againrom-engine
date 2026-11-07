package game

import (
	"bytes"
	"testing"

	"againrom/pkg/sim"
)

// A second-game mission SAVE after a weapon swap writes the hero's
// weapon-arm modifier words as his held weapon's own record forms them:
// to-hit at UD4 +0x12, damage base and spread at +0x20/+0x21, each the
// weapon record's attack block (DIV-3359). The original's removal subtracts
// that record's block from these words, so words left from the previous
// weapon wrap its damage to about 243 after the next swap.
func TestSecondMissionSaveWritesTheSwappedWeaponArm(t *testing.T) {
	sword, mace := sim.PlainItem(eqSwordCode), sim.PlainItem(eqMaceCode)
	f, app, screen, _ := secondBagFrontOf(t, []sim.ItemInstance{mace}, secondGuardMission())
	defs := eqDefsTable(t)
	f.Table.Shapes, f.Table.Materials = actorItemModeScale("Common"), actorItemModeScale("Iron")
	f.Table.Weapons = defs.Weapons
	f.Table.Shields, f.Table.Armors = defs.Shields, defs.Armors
	f.Carried[0].WornItems[0], f.Carried[0].Worn[0] = sword, sword.Code
	f.Carried[0].WornItems[2], f.Carried[0].Worn[2] = sim.ItemInstance{}, 0
	secondBagEnter(t, f, app, screen)
	w := f.live.world
	var hero sim.Entity
	found := false
	for _, e := range w.Entities() {
		if worn, ok := w.EquippedItems(e.ID); ok && e.Owner == sim.SelfSlot && worn[0].Code == sword.Code {
			hero, found = e, true
			break
		}
	}
	if !found {
		t.Fatal("the mission holds no hero with the sword")
	}
	stacks, _ := w.CarriedStacks(hero.ID)
	index := -1
	for k, s := range stacks {
		if s.Code == mace.Code {
			index = k
		}
	}
	if index < 0 {
		t.Fatalf("the hero's bag holds no %#04x: %+v", mace.Code, stacks)
	}
	sim.Step(w, []sim.Command{sim.Equip(hero.ID, sim.ItemSlot(index), 1)})
	if worn, _ := w.EquippedItems(hero.ID); worn[0].Code != mace.Code {
		t.Fatalf("the hero holds %#04x, want %#04x", worn[0].Code, mace.Code)
	}
	doc := checkSecondMissionFile(t, secondPageSave(t, f, app), 10)
	checked := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		refs, _ := savedObjectRefs(r, "HeldWeapon")
		if len(refs) != 1 || refs[0] == 0 {
			continue
		}
		weapon := &doc.Objects[refs[0]-1]
		if code, _ := savedStructureValue(weapon, "F40"); code != uint32(mace.Code) {
			continue
		}
		block, words := secondRecordRaw(weapon, "W52"), secondRecordRaw(r, "UD4")
		want := []byte{block[0], block[1], block[14], block[15]}
		if got := []byte{words[0x12], words[0x13], words[0x20], words[0x21]}; !bytes.Equal(got, want) || want[2] == 0 {
			t.Fatalf("the hero's weapon-arm words %x, want his weapon's %x", got, want)
		}
		checked++
	}
	if checked != 1 {
		t.Fatalf("%d Humans hold the swapped weapon, want the hero", checked)
	}
}

package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Item codes on the installed definition rows, shared by EN and RU: the first
// shield, Two Handed Sword, Short Bow and Short Sword.
const (
	shieldWitnessShield     = uint16(0x0201)
	shieldWitnessGreatsword = uint16(0x1106)
	shieldWitnessBow        = uint16(0x8134)
	shieldWitnessSword      = uint16(0x0103)
)

type shieldWitnessCase struct {
	name   string
	weapon uint16 // worn in slot 1 at the start; zero for none
	// packed is whether wearing the shield sends the weapon to the pack.
	packed bool
}

var shieldWitnessCases = []shieldWitnessCase{
	{"shield alone", 0, false},
	{"over a two-handed sword", shieldWitnessGreatsword, true},
	{"over a bow", shieldWitnessBow, true},
	{"beside a one-handed sword", shieldWitnessSword, false},
}

// shieldWitnessHold is what a hero wears and carries: the codes in his twelve
// worn slots and the sorted codes of his pack, one per unit.
type shieldWitnessHold struct {
	worn [sim.EquipSlots]uint16
	pack []uint16
}

func shieldWitnessRead(w *sim.World, id sim.EntityID) shieldWitnessHold {
	var h shieldWitnessHold
	h.worn, _ = w.Equipped(id)
	stacks, _ := w.CarriedStacks(id)
	for _, stack := range stacks {
		for n := uint32(0); n < stack.Count; n++ {
			h.pack = append(h.pack, stack.Code)
		}
	}
	slices.Sort(h.pack)
	return h
}

// shieldWitnessOpen opens mission 10 with a hero who wears tc.weapon in slot 1
// and carries the shield.
func shieldWitnessOpen(t *testing.T, tc shieldWitnessCase) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if slot, ok := EquipTarget(data.ItemCode(shieldWitnessShield), f.Table); !ok || slot != 2 {
		t.Fatalf("the installed rows put the shield %#x in slot %d (%v), want 2", shieldWitnessShield, slot, ok)
	}
	if tc.weapon != 0 {
		if slot, ok := EquipTarget(data.ItemCode(tc.weapon), f.Table); !ok || slot != 1 {
			t.Fatalf("the installed rows put the weapon %#x in slot %d (%v), want 1", tc.weapon, slot, ok)
		}
		if got := mapload.WeaponBlocksShield(data.ItemCode(tc.weapon), f.Table); got != tc.packed {
			t.Fatalf("the installed rows say weapon %#x blocks a shield = %v, want %v", tc.weapon, got, tc.packed)
		}
	}
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	member := &party[0]
	worn := mapload.MemberItemEquipment(*member, f.Table)
	carried := mapload.MemberCarriedItems(*member, f.Table)
	worn[0], worn[1] = sim.ItemInstance{}, sim.ItemInstance{}
	if tc.weapon != 0 {
		worn[0] = mapload.ItemInstanceFromCode(tc.weapon, f.Table)
	}
	carried = append(carried, mapload.ItemInstanceFromCode(shieldWitnessShield, f.Table))
	syncMissionPartyStock(member, worn, carried)
	member.Weapon = nil
	f.Carried = party
	app := f.App("shield equip")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	return f, app
}

// shieldWitnessSelect lets the mission settle, brings the hero into view and
// selects him so the pack bar answers pointer input.
func shieldWitnessSelect(t *testing.T, f *FrontEnd, app *ui.App) sim.EntityID {
	t.Helper()
	live := f.live
	id := live.mission.ids[0]
	for n := 0; n < 33; n++ {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
		if n == 31 {
			e, _ := live.entity(id)
			live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
		}
	}
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	return id
}

// shieldWitnessEquip double-clicks the shield's pack cell, the player's equip
// gesture, and lets the command apply.
func shieldWitnessEquip(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID) {
	t.Helper()
	stacks, _ := f.live.world.CarriedStacks(id)
	cell := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == shieldWitnessShield })
	if cell < 0 {
		t.Fatalf("the hero's pack holds no shield: %+v", stacks)
	}
	usePackCell(t, app, cell)
	for n := 0; n < 4; n++ {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
}

// shieldWitnessSave writes the mission through the game menu and returns to the
// map.
func shieldWitnessSave(t *testing.T, f *FrontEnd, app *ui.App) (SaveStore, string) {
	t.Helper()
	store, name, _ := menuSAVE(t, f, app, OriginalStore{})
	if app.Screen() == ui.ScreenGameMenu {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("after the SAVE the screen is %s, want the map", app.Screen())
	}
	return store, name
}

// shieldWitnessRequire checks a hero against the start state: the shield worn
// in slot 2, slot 1 keeping its one-handed weapon or left empty, and the pack
// holding what it held plus a weapon taken off, minus the shield.
func shieldWitnessRequire(t *testing.T, label string, tc shieldWitnessCase, start, got shieldWitnessHold) {
	t.Helper()
	want := shieldWitnessHold{worn: start.worn, pack: slices.Clone(start.pack)}
	want.worn[1] = shieldWitnessShield
	want.worn[0] = tc.weapon
	i := slices.Index(want.pack, shieldWitnessShield)
	if i < 0 {
		t.Fatalf("%s: the start pack %#x holds no shield", label, start.pack)
	}
	want.pack = slices.Delete(want.pack, i, i+1)
	if tc.packed {
		want.worn[0] = 0
		want.pack = append(want.pack, tc.weapon)
	}
	slices.Sort(want.pack)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: worn %#x pack %#x, want worn %#x pack %#x", label, got.worn, got.pack, want.worn, want.pack)
	}
}

// A shield is worn on its own; one put on over a two-handed sword or a bow takes
// the weapon off into the pack; a one-handed sword stays. The player double-
// clicks the shield in the pack. The result is the same for an actor built for
// the mission and for one loaded from a SAV, and it survives SAVE and LOAD.
func TestReleaseShieldWornAloneAndOverTwoHandedWeaponsThroughApp(t *testing.T) {
	for _, tc := range shieldWitnessCases {
		t.Run(tc.name, func(t *testing.T) {
			f, app := shieldWitnessOpen(t, tc)
			id := shieldWitnessSelect(t, f, app)
			start := shieldWitnessRead(f.live.world, id)
			if start.worn[0] != tc.weapon || start.worn[1] != 0 || !slices.Contains(start.pack, shieldWitnessShield) {
				t.Fatalf("start: worn %#x pack %#x, want weapon %#x in slot 1, slot 2 empty and the shield packed", start.worn, start.pack, tc.weapon)
			}
			beforeStore, beforeName := shieldWitnessSave(t, f, app)

			// The route of an actor built for the mission.
			before, _ := f.live.entity(id)
			shieldWitnessEquip(t, f, app, id)
			native := shieldWitnessRead(f.live.world, id)
			shieldWitnessRequire(t, "built actor", tc, start, native)
			after, _ := f.live.entity(id)
			if after.Defence <= before.Defence {
				t.Fatalf("built actor: Defence %d after the shield, %d before", after.Defence, before.Defence)
			}
			store, name := shieldWitnessSave(t, f, app)
			cold, _ := loadSAVWindow(t, store, name)
			shieldWitnessRequire(t, "SAVE and LOAD of the built actor", tc, start, shieldWitnessRead(cold.live.world, cold.live.mission.ids[0]))
			if e, _ := cold.live.entity(cold.live.mission.ids[0]); e.Defence != after.Defence {
				t.Fatalf("SAVE and LOAD of the built actor: Defence %d, want %d", e.Defence, after.Defence)
			}

			// The route of an actor loaded from a SAV written before the equip.
			loaded, loadedApp := loadSAVWindow(t, beforeStore, beforeName)
			loadedID := shieldWitnessSelect(t, loaded, loadedApp)
			if got := shieldWitnessRead(loaded.live.world, loadedID); !reflect.DeepEqual(got, start) {
				t.Fatalf("SAVE and LOAD before the equip: worn %#x pack %#x, want worn %#x pack %#x", got.worn, got.pack, start.worn, start.pack)
			}
			before, _ = loaded.live.entity(loadedID)
			shieldWitnessEquip(t, loaded, loadedApp, loadedID)
			shieldWitnessRequire(t, "loaded actor", tc, start, shieldWitnessRead(loaded.live.world, loadedID))
			after, _ = loaded.live.entity(loadedID)
			if after.Defence <= before.Defence {
				t.Fatalf("loaded actor: Defence %d after the shield, %d before", after.Defence, before.Defence)
			}
			store, name = shieldWitnessSave(t, loaded, loadedApp)
			again, _ := loadSAVWindow(t, store, name)
			shieldWitnessRequire(t, "SAVE and LOAD of the loaded actor", tc, start, shieldWitnessRead(again.live.world, again.live.mission.ids[0]))
			if e, _ := again.live.entity(again.live.mission.ids[0]); e.Defence != after.Defence {
				t.Fatalf("SAVE and LOAD of the loaded actor: Defence %d, want %d", e.Defence, after.Defence)
			}
		})
	}
}

// townShieldHeld checks the town hero's slots 1 and 2 and his pack.
func townShieldHeld(t *testing.T, f *FrontEnd, label string, weapon, shield uint16, inPack, notInPack []uint16) {
	t.Helper()
	worn, pack := currentTownMember(t, f, "hero")
	if worn[0] != weapon || worn[1] != shield {
		t.Fatalf("%s: slots 1 and 2 hold %#x and %#x, want %#x and %#x (pack %#x)", label, worn[0], worn[1], weapon, shield, pack)
	}
	for _, code := range inPack {
		if !slices.Contains(pack, code) {
			t.Fatalf("%s: the pack %#x lacks %#x", label, pack, code)
		}
	}
	for _, code := range notInPack {
		if slices.Contains(pack, code) {
			t.Fatalf("%s: the pack %#x still holds %#x", label, pack, code)
		}
	}
}

// townShieldWear drags one pack item onto the doll in the merchant's room.
func townShieldWear(t *testing.T, s *townScreen, code uint16) {
	t.Helper()
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, code)}, ui.ShopControl{Kind: ui.ShopControlDoll})
}

// In the merchant's room the hero wears the shield 0x1222 beside his sword
// 0x103, keeps it when the sword comes off, keeps it worn alone when the Bronze
// Two-Handed Sword 0x1106 goes to the pack for it, and the town SAV holds each
// state. A LOAD of that SAV gives the same result for the next wear.
func TestReleaseTownShieldWornAloneAndOverATwoHandedSwordSavesAsSAV(t *testing.T) {
	const sword, twoHanded, shield = uint16(0x103), uint16(0x1106), uint16(0x1222)
	f := currentTown(t, func(f *FrontEnd, hero sim.EntityID) {
		equipmentReturnPick(t, f, hero, 20, 65) // the Bronze Two-Handed Sword
		equipmentReturnPick(t, f, hero, 12, 50) // the shield
	}, nil)
	s := currentTownShop(f, "hero")
	townShieldHeld(t, f, "start", sword, 0, []uint16{twoHanded, shield}, nil)

	townShieldWear(t, s, shield)
	townShieldHeld(t, f, "shield beside the sword", sword, shield, []uint16{twoHanded}, []uint16{shield})

	click(s, ui.ShopControlDoll, 0)
	townShieldHeld(t, f, "sword taken off", 0, shield, []uint16{twoHanded, sword}, []uint16{shield})

	townShieldWear(t, s, twoHanded)
	townShieldHeld(t, f, "two-handed sword over the shield", twoHanded, 0, []uint16{sword, shield}, nil)

	townShieldWear(t, s, shield)
	townShieldHeld(t, f, "shield over the two-handed sword", 0, shield, []uint16{twoHanded, sword}, []uint16{shield})

	g := currentTownReload(t, currentTownSave(t, f))
	townShieldHeld(t, g, "after SAVE and LOAD", 0, shield, []uint16{twoHanded, sword}, []uint16{shield})

	s = currentTownShop(g, "hero")
	townShieldWear(t, s, twoHanded)
	townShieldHeld(t, g, "loaded hero, two-handed sword over the shield", twoHanded, 0, []uint16{sword, shield}, nil)
	townShieldWear(t, s, shield)
	townShieldHeld(t, g, "loaded hero, shield over the two-handed sword", 0, shield, []uint16{twoHanded, sword}, []uint16{shield})

	h := currentTownReload(t, currentTownSave(t, g))
	townShieldHeld(t, h, "after the second SAVE and LOAD", 0, shield, []uint16{twoHanded, sword}, []uint16{shield})
}

package game

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const swingShieldCode = uint16(2)<<8 | 1

// heroSwingMission opens mission 10 with a bare-handed hero who equips codes.
func heroSwingMission(t *testing.T, f *FrontEnd, codes ...uint16) (*mapWorld, sim.EntityID) {
	t.Helper()
	party := f.ChargenParty(ui.ChargenResult{Name: "Swing witness", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	hero := &party[0]
	hero.Worn[0], hero.WornItems[0], hero.Weapon = 0, sim.ItemInstance{}, nil
	_, _, hero.Class, _ = data.HeroAppearance(f.Bodies, equipmentFromSlots(hero.Worn), false, false)
	if hero.Class != 1 && hero.Class != 2 {
		t.Fatalf("bare-handed hero class %d, want an unarmed class", hero.Class)
	}
	for _, code := range codes {
		hero.Carried = append(slices.Clone(hero.Carried), code)
		hero.CarriedItems = append(slices.Clone(hero.CarriedItems), mapload.ItemInstanceFromCode(code, f.Table))
	}

	a := f.App("weapon swing sound")
	t.Cleanup(a.StopAudio)
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	id := equipmentReturnHero(t, f)
	for _, code := range codes {
		slot, ok := data.EquipSlotFor(data.ItemCode(code))
		if !ok {
			t.Fatalf("code %#x names no equipment slot", code)
		}
		carried, _ := mw.world.CarriedItems(id)
		index := slices.IndexFunc(carried, func(item sim.ItemInstance) bool { return item.Code == code })
		if index < 0 {
			t.Fatalf("hero carries no %#x", code)
		}
		sim.Run(mw.world, [][]sim.Command{{sim.Equip(id, sim.ItemSlot(index), sim.EquipSlot(slot))}}, 3)
		if worn, _ := mw.world.EquippedItems(id); worn[slot-1].Code != code {
			t.Fatalf("slot %d holds %#x after equipping %#x", slot, worn[slot-1].Code, code)
		}
	}
	return mw, id
}

// swingSlotsAt returns the sound slots a swing at an adjacent person requests.
func swingSlotsAt(t *testing.T, f *FrontEnd, mw *mapWorld, id sim.EntityID) []int {
	t.Helper()
	var victim sim.EntityID
	found := false
	for _, e := range mw.world.Entities() {
		if e.ID != id && e.Owner != sim.SelfSlot && e.Alive() && !e.OffMap {
			victim, found = e.ID, true
			break
		}
	}
	if !found {
		t.Fatal("the mission holds no other person to strike")
	}
	hv, _ := mw.entity(id)
	if err := mw.world.HeadlessPlace(victim, hv.X+1, hv.Y); err != nil {
		t.Fatal(err)
	}
	heard := observeReleaseAudio(t, f)
	mw.strike(uint32(id), uint32(victim))
	for tick := 0; tick < 12; tick++ {
		mw.tick()
	}
	var slots []int
	for i, request := range heard.requests {
		if request.Source == "unit-swing" && releaseAudioFromActor(t, heard, i, id) {
			slot, err := strconv.Atoi(strings.TrimPrefix(request.Selector, "registry:"))
			if err != nil {
				t.Fatal(err)
			}
			slots = append(slots, slot)
		}
	}
	return slots
}

func weaponSwingSlots(t *testing.T, f *FrontEnd, codes ...uint16) []int {
	t.Helper()
	mw, id := heroSwingMission(t, f, codes...)
	return swingSlotsAt(t, f, mw, id)
}

func TestReleaseEquippedWeaponSwingSound(t *testing.T) {
	f := releaseFront(t)
	sword := data.ItemCode(0x0103)
	dagger := data.ItemCode(0x0102)
	swordSlots := weaponSwingSlots(t, f, uint16(sword))
	daggerSlots := weaponSwingSlots(t, releaseFront(t), uint16(dagger))
	want := []int{int(f.SoundClasses[3].Slots[0])}
	if bare := weaponSwingSlots(t, releaseFront(t)); len(bare) != 0 {
		t.Fatalf("bare-handed hero requested swing slots %v, want none", bare)
	}
	if want[0] != 100 || !slices.Equal(swordSlots, want) || !slices.Equal(daggerSlots, want) {
		t.Fatalf("swing slots: short sword %v, dagger %v, want %v", swordSlots, daggerSlots, want)
	}
}

// A dagger and shield hero is class 4 (slot 100), also after SAVE and LOAD.
func TestReleaseHeroDaggerAndShieldSwingClassSurvivesSaveLoad(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	mw, id := heroSwingMission(t, f, 0x0102, swingShieldCode)
	e, _ := mw.entity(id)
	if got := mw.spellClientClass(id, e.Class); got != 4 || f.SoundClasses[4].Slots[0] != 100 {
		t.Fatalf("dagger and shield hero class %d with Sound[0] %v, want class 4 with 100", got, f.SoundClasses[4].Slots)
	}
	path, _ := writeOrdinarySAV(t, f, "dagger-shield.sav")
	if got := swingSlotsAt(t, f, mw, id); !slices.Equal(got, []int{100}) {
		t.Fatalf("dagger and shield swing slots %v, want [100]", got)
	}

	g, _ := lancerLoad(t, filepath.Dir(path), filepath.Base(path))
	cold := g.live
	le, ok := cold.entity(id)
	if !ok {
		t.Fatalf("hero %d absent after LOAD", id)
	}
	if got := cold.spellClientClass(id, le.Class); got != 4 {
		t.Fatalf("after LOAD the dagger and shield hero class is %d, want 4", got)
	}
	if got := swingSlotsAt(t, g, cold, id); !slices.Equal(got, []int{100}) {
		t.Fatalf("after LOAD the dagger and shield swing slots %v, want [100]", got)
	}
}

// A hero's class follows his equipment: 4, then 3, then unarmed class 1.
func TestReleaseHeroSwingClassRederivesOnEquipmentChange(t *testing.T) {
	f := releaseFront(t)
	mw, id := heroSwingMission(t, f, 0x0102, swingShieldCode)
	class := func() int32 {
		e, _ := mw.entity(id)
		return mw.spellClientClass(id, e.Class)
	}
	if got := class(); got != 4 {
		t.Fatalf("dagger and shield class %d, want 4", got)
	}
	sim.Run(mw.world, [][]sim.Command{{sim.DropWorn(id, 2, sim.CellPoint{X: 3, Y: 3})}}, 3)
	if got := class(); got != 3 {
		t.Fatalf("dagger alone class %d, want 3", got)
	}
	sim.Run(mw.world, [][]sim.Command{{sim.DropWorn(id, 1, sim.CellPoint{X: 3, Y: 3})}}, 3)
	if got := class(); got != 1 || f.SoundClasses[1].Slots[0] != 0 {
		t.Fatalf("bare-handed class %d with Sound[0] %v, want class 1 and 0", got, f.SoundClasses[1].Slots)
	}
}

// Mission 20's three guards (row 58) keep class 10, slot 120, whatever they
// wear and across SAVE and LOAD; Sarindar stays the silent class 23.
func TestReleaseMission20MapAuthoredHumansKeepTheirRowClass(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("map-authored class")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	var guards []sim.EntityID
	for id := range mw.mission.state.Start.Roster {
		if e, ok := mw.entity(id); ok && e.TypeID == 10 {
			guards = append(guards, id)
		}
	}
	slices.Sort(guards)
	if len(guards) != 3 {
		t.Fatalf("mission 20 holds %d roster humans of type 10, want the 3 guards", len(guards))
	}
	if f.SoundClasses[10].Slots[0] != 120 {
		t.Fatalf("class 10 Sound[0] %v, want 120", f.SoundClasses[10].Slots)
	}
	check := func(live *mapWorld, when string) {
		t.Helper()
		for _, id := range guards {
			e, _ := live.entity(id)
			if got := live.spellClientClass(id, e.Class); got != 10 {
				t.Errorf("%s: guard %d swing class %d, want its row class 10", when, id, got)
			}
			if art := live.art[id]; art != live.units.Classes[10] {
				t.Errorf("%s: guard %d draws another class than 10", when, id)
			}
		}
		const sarindar sim.EntityID = 48
		e, _ := live.entity(sarindar)
		if got := live.spellClientClass(sarindar, e.Class); got != 23 || live.sounds[23].Slots[0] != 0 {
			t.Errorf("%s: Sarindar swing class %d, want the silent class 23", when, got)
		}
	}
	check(mw, "fresh")
	for _, id := range guards {
		if slots, _ := mw.world.Equipped(id); slots[0] != 0 {
			sim.Run(mw.world, [][]sim.Command{{sim.DropWorn(id, 1, sim.CellPoint{X: 1, Y: 1})}}, 3)
			if after, _ := mw.world.Equipped(id); after[0] != 0 {
				t.Fatalf("guard %d still holds %#x after DropWorn", id, after[0])
			}
		}
	}
	check(mw, "after the guards drop their weapons")

	path, _ := writeOrdinarySAV(t, f, "map-authored.sav")
	g, _ := lancerLoad(t, filepath.Dir(path), filepath.Base(path))
	check(g.live, "cold LOAD")
}

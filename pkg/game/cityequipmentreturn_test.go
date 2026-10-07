package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// equipmentReturnHero is the live starting hero of the open mission.
func equipmentReturnHero(t *testing.T, f *FrontEnd) sim.EntityID {
	t.Helper()
	for i, p := range f.live.mission.party {
		if p.StartingHero {
			return f.live.mission.ids[i]
		}
	}
	t.Fatal("no starting hero in the open mission")
	return 0
}

// equipmentReturnPick takes the ground sack at (x, y) into the hero's pack.
func equipmentReturnPick(t *testing.T, f *FrontEnd, hero sim.EntityID, x, y int32) {
	t.Helper()
	liveTakeAt(t, f.live, hero, x, y)
}

// equipmentReturnEquip equips the hero's first pack item of code into slot
// through the ordinary command path.
func equipmentReturnEquip(t *testing.T, f *FrontEnd, hero sim.EntityID, code uint16, slot int) {
	t.Helper()
	carried, _ := f.live.world.CarriedItems(hero)
	index := slices.IndexFunc(carried, func(item sim.ItemInstance) bool { return item.Code == code })
	if index < 0 {
		t.Fatalf("hero carries no %#x to equip", code)
	}
	sim.Run(f.live.world, [][]sim.Command{{sim.Equip(hero, sim.ItemSlot(index), sim.EquipSlot(slot))}}, 3)
	worn, _ := f.live.world.EquippedItems(hero)
	if worn[slot-1].Code != code {
		t.Fatalf("hero slot %d holds %#x after equipping %#x", slot, worn[slot-1].Code, code)
	}
}

// equipmentReturnTownSave finishes the open mission, opens a town room, whose
// faces re-derive every member's body from what he wears, and makes the
// ordinary town SAVE. A SAVE that falls back to .ags fails with the export's
// own refusal.
func equipmentReturnTownSave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish the mission: %v", err)
	}
	town := f.TownScreen().(*townScreen)
	town.room = roomTavern
	town.composeShopFaces()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("town SAVE: %v", err)
	}
	if !IsOriginal(name) {
		snapshot, label, err := f.Snapshot(false)
		if err == nil {
			_, err = f.ExportOriginalSave(snapshot, label)
		}
		t.Fatalf("town SAVE wrote %q, want an original-format SAV; export refusal: %v", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// equipmentReturnReload loads a town SAV in a fresh front end and returns the
// starting hero's equipment and pack codes.
func equipmentReturnReload(t *testing.T, raw []byte) (worn [sim.EquipSlots]uint16, pack []uint16) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	_, _, load := agsSaveSeams(g, SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	if _, town, err := load("town.sav"); err != nil || !town {
		t.Fatalf("town LOAD of the written SAV: town=%v err=%v", town, err)
	}
	for _, p := range g.Carried {
		if !p.StartingHero {
			continue
		}
		for i, item := range mapload.MemberItemEquipment(p, nil) {
			worn[i] = item.Code
		}
		for _, item := range mapload.MemberCarriedItems(p, nil) {
			pack = append(pack, item.Code)
		}
		return worn, pack
	}
	t.Fatal("the reloaded town has no starting hero")
	return worn, pack
}

// loadedMission20 is the player's route to a mid-mission LOAD: a fresh
// chargen plays mission 10 (m10 runs there), opens mission 20 and makes the
// mission SAVE, then a fresh front end LOADs that SAV and reopens mission 20.
func loadedMission20(t *testing.T, m10 func(f *FrontEnd, hero sim.EntityID)) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Equipment return", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("equipment return")
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	if m10 != nil {
		m10(f, equipmentReturnHero(t, f))
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 10: %v", err)
	}
	if err := app.OpenMission(f.MissionOpenerWith(20, f.NextParty())); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE in mission 20 wrote %q: %v", name, err)
	}
	g := releaseFront(t)
	_, _, load := agsSaveSeams(g, store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("LOAD of the mission SAVE: town=%v err=%v", town, err)
	}
	if err := g.App("equipment return loaded").OpenMission(open); err != nil {
		t.Fatalf("reopen the loaded mission 20: %v", err)
	}
	return g
}

// TestReleaseLoadedMission20WeaponSwapSavesTheTownAsSAV is the owner's own
// item pair on the player's route: the Bronze Two-Handed Sword 0x1106 is
// carried in from mission 10's sack at (20,65), and after the mid-mission
// LOAD it replaces the Iron Short Sword 0x103. The hero's drawn class moves
// from the one-handed swordsman to the two-handed one, which the retained
// party still records as the mission-entry value.
func TestReleaseLoadedMission20WeaponSwapSavesTheTownAsSAV(t *testing.T) {
	f := loadedMission20(t, func(f *FrontEnd, hero sim.EntityID) { equipmentReturnPick(t, f, hero, 20, 65) })
	equipmentReturnEquip(t, f, equipmentReturnHero(t, f), 0x1106, 1)
	worn, pack := equipmentReturnReload(t, equipmentReturnTownSave(t, f))
	if worn[0] != 0x1106 || !slices.Contains(pack, 0x103) {
		t.Fatalf("reloaded hero weapon %#x, pack %#x; want 0x1106 held and 0x103 in the pack", worn[0], pack)
	}
}

// TestReleaseLoadedMission20PickedUpArmourWornSavesTheTownAsSAV wears an
// armour first picked up in mission 20, 0xf72d from the sack at (24,39), in
// slot 7. Its owner Reference names the ground sack it came from, which the
// town document does not hold.
func TestReleaseLoadedMission20PickedUpArmourWornSavesTheTownAsSAV(t *testing.T) {
	f := loadedMission20(t, nil)
	hero := equipmentReturnHero(t, f)
	equipmentReturnPick(t, f, hero, 24, 39)
	equipmentReturnEquip(t, f, hero, 0xf72d, 7)
	worn, _ := equipmentReturnReload(t, equipmentReturnTownSave(t, f))
	if worn[6] != 0xf72d {
		t.Fatalf("reloaded hero slot 7 holds %#x, want the picked-up 0xf72d", worn[6])
	}
}

// TestReleaseLoadedMission20ShieldAddedSavesTheTownAsSAV adds the shield
// 0x1222, carried in from mission 10's sack at (12,50), beside the hero's
// one-handed sword after the LOAD. The shield moves his drawn class too.
func TestReleaseLoadedMission20ShieldAddedSavesTheTownAsSAV(t *testing.T) {
	f := loadedMission20(t, func(f *FrontEnd, hero sim.EntityID) { equipmentReturnPick(t, f, hero, 12, 50) })
	equipmentReturnEquip(t, f, equipmentReturnHero(t, f), 0x1222, 2)
	worn, _ := equipmentReturnReload(t, equipmentReturnTownSave(t, f))
	if worn[0] != 0x103 || worn[1] != 0x1222 {
		t.Fatalf("reloaded hero weapon %#x, shield %#x; want 0x103 and 0x1222", worn[0], worn[1])
	}
}

// TestReleaseImportedTownBowEquipSavesTheTownAsSAV is the town-opened route:
// an original town SAV is loaded, mission 30 is played from it, and the hero
// equips the bow 0x8134 from his pack in place of his sword 0x106. The
// Human's modifier block and own weight move with that change. The hero's
// unchanged worn armour keeps the owner Reference its source file gave it,
// the Player, which still resolves in the written town.
func TestReleaseImportedTownBowEquipSavesTheTownAsSAV(t *testing.T) {
	f, _ := townReturnImported1168(t)
	if err := f.App("bow equip return").OpenMission(f.MissionOpenerWith(30, f.NextParty())); err != nil {
		t.Fatalf("open mission 30: %v", err)
	}
	equipmentReturnEquip(t, f, equipmentReturnHero(t, f), 0x8134, 1)
	raw := equipmentReturnTownSave(t, f)
	worn, pack := equipmentReturnReload(t, raw)
	if worn[0] != 0x8134 || !slices.Contains(pack, 0x106) {
		t.Fatalf("reloaded hero weapon %#x, pack %#x; want 0x8134 held and 0x106 in the pack", worn[0], pack)
	}
	d, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	identities := make(map[uint32]string)
	for i := range d.Objects {
		field := "Identity"
		if d.Objects[i].Class == "Player" {
			field = "This"
		}
		if v, err := savedStructureValue(&d.Objects[i], field); err == nil && v != 0 {
			identities[v] = d.Objects[i].Class
		}
	}
	kept := 0
	for i := range d.Objects {
		r := &d.Objects[i]
		if r.Class != "Armor" {
			continue
		}
		code, _ := savedStructureValue(r, "F40")
		if code != 0x162a && code != 0x1833 && code != 0x916 {
			continue
		}
		reference, err := savedStructureValue(r, "Reference")
		if err != nil || reference == 0 || identities[reference] != "Player" {
			t.Fatalf("hero armour %#x Reference %#x resolves to %q, want the Player", code, reference, identities[reference])
		}
		kept++
	}
	if kept != 3 {
		t.Fatalf("found %d of the hero's three unchanged Player-owned armours", kept)
	}
}

package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// townEntrySaves names the directory holding the tester's town SAVs
// cherepaha.sav (written by this game) and what.sav (the original's resave of
// it). No byte of either is in the repository.
func townEntrySaves(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("AGAINROM_TOWN_ENTRY_SAVS")
	if dir == "" {
		t.Skip("AGAINROM_TOWN_ENTRY_SAVS is not set")
	}
	return dir
}

// townEntryOpen cold-LOADs one named town SAV through the LOAD window.
func townEntryOpen(t *testing.T, name string) (*FrontEnd, *townScreen) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(townEntrySaves(t), name))
	if err != nil {
		t.Skip("no ", name, ": ", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	openLocalTownSAV(t, f, dir, name)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("LOAD left no town screen")
	}
	return f, s
}

// townEntryTravel takes the world-map route to the first enabled mission
// record and runs the opener its arrival returns, the call the town flow makes.
// A refused opener sends the party home; the error is what the player reads.
func townEntryTravel(t *testing.T, f *FrontEnd, s *townScreen) (int, error) {
	t.Helper()
	s.room = roomGates
	s.enterWorldMap()
	for i, m := range s.worldMap.missions {
		if !m.Enabled || m.Kind == placeTown {
			continue
		}
		s.selectWorldMission(i)
		s.WorldMapChoose()
		for k := 0; k < 4096; k++ {
			act := s.WorldMapTick()
			if act.Open == nil {
				continue
			}
			_, _, _, _, _, _, _, _, _, _, err := act.Open()
			if err == nil && f.liveMission != m.Number {
				t.Fatalf("mission %d opened, live mission %d", m.Number, f.liveMission)
			}
			return m.Number, err
		}
		t.Fatalf("mission %d: travel never arrived", m.Number)
	}
	t.Fatal("no enabled mission record")
	return 0, nil
}

// bindWornSpellWeapons constructs every carried member's worn castSpell
// Weapon that holds no constructor operands, as an equip does.
func bindWornSpellWeapons(f *FrontEnd) []string {
	var out []string
	for i := range f.Carried {
		worn := f.Carried[i].WornItems[:]
		if f.Carried[i].Carry != nil {
			worn = f.Carried[i].Carry.EquippedItems[:]
		}
		for slot := range worn {
			if _, _, cast := worn[slot].CastSpell(); cast && worn[slot].SourceEquipment.Class == 0 {
				worn[slot] = mapload.SourceEquippedItem(worn[slot], f.Table)
				out = append(out, fmt.Sprintf("member %d slot %d code %d", i, slot+1, worn[slot].Code))
			}
		}
	}
	return out
}

// Both files hold the same campaign and party. cherepaha.sav's current-state
// carrier restores two worn castSpell Weapons, the hero's staff and a
// companion's pike, without constructor operands; what.sav restores them
// constructed. The bound control changes only those two values in the loaded
// session and leaves the carrier's offer labels as loaded.
func TestReleaseTownSAVWithAnUnconstructedWornSpellWeaponEntersAMission(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, name := range []string{"what.sav", "cherepaha.sav"} {
		t.Run(name, func(t *testing.T) {
			f, s := townEntryOpen(t, name)
			if n, err := townEntryTravel(t, f, s); err != nil {
				t.Errorf("mission %d refused: %v", n, err)
			}
		})
	}
	t.Run("cherepaha.sav bound staff", func(t *testing.T) {
		f, s := townEntryOpen(t, "cherepaha.sav")
		bound := bindWornSpellWeapons(f)
		t.Logf("control: bound %v", bound)
		if len(bound) == 0 {
			t.Fatal("control: the file holds no unconstructed worn spell weapon")
		}
		if n, err := townEntryTravel(t, f, s); err != nil {
			t.Fatalf("control: mission %d refused with the staff constructed: %v", n, err)
		}
	})
}

func townEntryMageShop(t *testing.T) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{}
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Mage", Choices: []int{0, 1, 0}, Stats: []int{25, 25, 25, 25}})
	app := f.App("bought spell weapon")
	if err := app.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	documentsObjective(t, f, app, 10)
	if err := app.HeadlessActivate("notice"); err != nil || f.liveMission != 20 {
		t.Fatal("mission 10 victory did not enter mission 20", err, f.liveMission)
	}
	documentsObjective(t, f, app, 20)
	campaignReturn(t, f, app)
	f.Town.gold = 1 << 30
	s := f.TownScreen().(*townScreen)
	s.room = roomShop
	return f, app, s
}

func townEntryBuySpellWeapon(t *testing.T, s *townScreen, route string) uint16 {
	t.Helper()
	for ri, room := range s.rack() {
		if !room.stocked {
			continue
		}
		for k, it := range s.deal().Shelf(room.shelf) {
			if _, _, cast := it.Instance().CastSpell(); !cast || it.Code.B() != 1 {
				continue
			}
			s.shopChosen, s.shelfBase = ri, 0
			if route == "pack" || route == "table" {
				s.shopShelf = room.shelf
				s.shopFromShelf(k, false)
				if len(s.deal().Table()) != 1 {
					t.Fatal("shelf item did not reach the trade table")
				}
				if route == "table" {
					s.shopEquipFromTable(0)
				} else {
					s.shopBuy()
					found := false
					for index, stack := range s.shopPackStacks() {
						if stack.Code == uint16(it.Code) {
							s.packBase = index
							s.shopEquipFromPack(1)
							found = true
							break
						}
					}
					if !found {
						t.Fatal("purchase did not reach the pack")
					}
				}
			} else {
				s.shopEquipFromShelf(k)
			}
			return uint16(it.Code)
		}
	}
	t.Fatal("the town-30 shop offers no castSpell Weapon")
	return 0
}

func TestReleaseTownBoughtSpellWeaponEntersTheNextMission(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, route := range []string{"shelf", "pack", "table", "legacy class zero", "constructed without Spell"} {
		t.Run(route, func(t *testing.T) {
			f, _, s := townEntryMageShop(t)
			bought := townEntryBuySpellWeapon(t, s, route)
			worn := &f.Carried[0].Carry.EquippedItems[0]
			if worn.Code != bought || worn.SourceEquipment.Class != sim.SourceWeapon || !worn.SourceEquipment.Spell.Present {
				t.Fatalf("worn %d: constructor class %d, Spell %v; want a constructed Weapon with its Spell", worn.Code, worn.SourceEquipment.Class, worn.SourceEquipment.Spell.Present)
			}
			if route == "legacy class zero" {
				worn.SourceEquipment = sim.SourceEquipment{}
				worn.Weight, worn.WeightPresent = 0, false
			} else if route == "constructed without Spell" {
				worn.SourceEquipment.Spell = sim.SourceItemSpell{}
			}
			takeCampaignOffer(t, f, 30)
			raw, stage, err := writerCensusSave(f, false, t.TempDir())
			if err != nil {
				t.Fatal(stage, err)
			}
			g := releaseFront(t)
			if _, town, err := g.RestoreOriginal(raw); err != nil || !town {
				t.Fatal("cold town LOAD", town, err)
			}
			loaded := mapload.MemberItemEquipment(g.Carried[0], g.Table)[0]
			if !sim.ItemEqual(loaded, *worn) {
				t.Fatal("town SAVE and LOAD changed the held weapon")
			}
			if n, err := townEntryTravel(t, g, g.TownScreen().(*townScreen)); err != nil {
				t.Fatalf("mission %d refused: %v", n, err)
			}
			items, _ := g.live.world.EquippedItems(g.live.mission.ids[0])
			if items[0].Code != bought || items[0].SourceEquipment.Class != sim.SourceWeapon || items[0].SourceEquipment.Spell.Present != (route != "constructed without Spell") {
				t.Fatalf("mission worn weapon: %+v", items[0])
			}
		})
	}
}

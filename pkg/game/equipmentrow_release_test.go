package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// requireEquipmentDefinitionRows fails on a written equipment row that is 0,
// outside its table, or not its code's row (SAV-1088).
func requireEquipmentDefinitionRows(t *testing.T, f *FrontEnd, point string, raw []byte) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatalf("%s: decode: %v", point, err)
	}
	sizes := map[string]int{"Weapon": f.Table.Weapons.Len(), "Armor": f.Table.Armors.Len(), "Shield": f.Table.Shields.Len()}
	seen := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if !equipmentRecordClass(r.Class) {
			continue
		}
		seen++
		row, err := savedStructureValue(r, "T0C")
		if err != nil {
			t.Fatalf("%s: object %d: %v", point, i+1, err)
		}
		code, err := savedStructureValue(r, "F40")
		if err != nil {
			t.Fatalf("%s: object %d: %v", point, i+1, err)
		}
		if row == 0 || int(row) >= sizes[r.Class] || row != code&0x1f {
			t.Errorf("%s: object %d %s code %#x names definition row %d; want %d of %d", point, i+1, r.Class, code, row, code&0x1f, sizes[r.Class])
		}
	}
	if seen == 0 {
		t.Fatalf("%s: the SAV holds no Weapon, Armor or Shield", point)
	}
}

func readMissionSave(t *testing.T, f *FrontEnd, app *ui.App) []byte {
	t.Helper()
	raw, err := os.ReadFile(generatedMissionSave(t, f, app))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Save points: a fresh mission, the first town, a purchase carried into the
// next mission, and a drop and pick-up there.
func TestReleaseWrittenEquipmentNamesItsCodeDefinitionRow(t *testing.T) {
	fresh := releaseFront(t)
	party := fresh.ChargenParty(ui.ChargenResult{Name: "Row witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := fresh.App("row witness")
	if err := app.OpenMission(fresh.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	requireEquipmentDefinitionRows(t, fresh, "fresh mission", readMissionSave(t, fresh, app))

	f := releaseFront(t)
	party = f.ChargenParty(ui.ChargenResult{Name: "Row witness town", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app = f.App("row witness town")
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 10: %v", err)
	}
	if err := app.OpenMission(f.MissionOpenerWith(20, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 20: %v", err)
	}
	requireEquipmentDefinitionRows(t, f, "town", currentTownSave(t, f))

	currentTownBuy(t, currentTownShop(f, "hero"), roomWeapons, 0x1102)
	if err := app.OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	requireEquipmentDefinitionRows(t, f, "purchase carried into a mission", readMissionSave(t, f, app))

	hero := equipmentReturnHero(t, f)
	pack, _ := f.live.world.CarriedStacks(hero)
	index := -1
	for i, item := range pack {
		if item.Code == 0x1102 {
			index = i
		}
	}
	if index < 0 {
		t.Fatal("the bought 0x1102 is not in the hero's mission pack")
	}
	var actor sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.ID == hero {
			actor = e
		}
	}
	f.live.pending = append(f.live.pending, sim.DropCarried(hero, sim.ItemSlot(index), sim.CellPoint{X: actor.X, Y: actor.Y}))
	f.live.tick()
	if groundAt(f.live.world.Sacks(), actor.X, actor.Y) == nil {
		t.Fatal("the drop left no ground holdings")
	}
	equipmentReturnPick(t, f, hero, actor.X, actor.Y)
	requireEquipmentDefinitionRows(t, f, "drop and pick-up", readMissionSave(t, f, app))
}

// A loaded row-0 Weapon, as this engine once wrote, resaves derived.
func TestReleaseLoadedZeroDefinitionRowResavesDerived(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Row reload", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("row reload")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(readMissionSave(t, f, app))
	if err != nil {
		t.Fatal(err)
	}
	zeroed := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Weapon" {
			continue
		}
		for j := range r.Values {
			if r.Values[j].Name == "T0C" {
				r.Values[j].Value = 0
				zeroed++
			}
		}
	}
	if zeroed == 0 {
		t.Fatal("the fresh mission SAV holds no Weapon")
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zero.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	open, town, err := load("zero.sav")
	if err != nil || town {
		t.Fatalf("LOAD of the row-0 SAV: town=%v err=%v", town, err)
	}
	reopened := g.App("row reload loaded")
	if err := reopened.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	requireEquipmentDefinitionRows(t, g, "resave of a loaded row-0 Weapon", readMissionSave(t, g, reopened))
}

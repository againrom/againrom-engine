package game

import (
	"slices"
	"testing"

	"againrom/pkg/sim"
)

// elvenSavTable gives f's table MagicItems row 2, a SuperScroll (the second
// game's Elven Scroll) whose string casts spell 1 at 100.
func elvenSavTable(f *FrontEnd) {
	rows := make(dbCollection, 29)
	rows[2] = dbEntry{name: "SuperScroll of spell", params: []int32{1000}, strings: []string{"castSpell=spell:100"}}
	f.Table.MagicItems = rows
}

// elvenSavOld is the Elven Scroll as an engine before the SuperScroll kind
// saved it: kind 0, no effects. It is also every engine's bag form of the
// row, which holds no kind.
var elvenSavOld = sim.ItemInstance{Code: 0x0e02, Price: 1000}

// A town file whose bag holds an Elven Scroll LOADs, and the mission the
// hero enters holds a scroll that reads its spell at its row's power.
func TestAnElvenScrollOfATownFileBagEntersTheMissionAsAScroll(t *testing.T) {
	f, _, _, _ := secondBagFront(t, []sim.ItemInstance{elvenSavOld})
	elvenSavTable(f)
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatalf("town SAVE: %v", err)
	}
	cold, app, _, _ := secondBagFront(t, nil)
	elvenSavTable(cold)
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold town LOAD", town, err)
	}
	if got := bagLabels(*cold.Town.second.heroBag()); !slices.Equal(got, bagLabels([]sim.ItemInstance{elvenSavOld})) {
		t.Fatalf("cold town LOAD bag %v, want the Elven Scroll as saved", got)
	}
	screen := cold.TownScreen().(*townScreen)
	app.SetTown(screen)
	secondBagEnter(t, cold, app, screen)
	items, _ := cold.live.world.CarriedItems(cold.live.mission.ids[0])
	i := slices.IndexFunc(items, func(it sim.ItemInstance) bool { return it.Code == elvenSavOld.Code })
	if i < 0 {
		t.Fatalf("mission hero pack %v holds no Elven Scroll", bagLabels(items))
	}
	if id, power, ok := sim.ScrollSpell(items[i]); !ok || id != 1 || power != 100 || items[i].Price != elvenSavOld.Price {
		t.Fatalf("mission Elven Scroll %v: spell %d power %d scroll %v, want spell 1 at 100", bagLabels(items[i:i+1]), id, power, ok)
	}
}

// A mission file an engine before the SuperScroll kind wrote holds its
// Elven Scroll bound to a saved object with kind 0 and no effects. It LOADs
// with the scroll as saved, which reads no spell in that mission (DIV-4684),
// and after the victory the scroll comes home to the bag in its bag form,
// from which the next mission builds the scroll.
func TestAnElvenScrollOfAnOldMissionFileLoadsAsSavedAndComesHome(t *testing.T) {
	f, app, screen, dir := secondBagFront(t, nil)
	elvenSavTable(f)
	secondBagOldScrollPack(t, f, elvenSavOld)
	secondBagEnter(t, f, app, screen)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := secondSaveFiles(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("mission SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	cold := secondMissionFrontOf(t, secondMissionContent{side: 128})
	elvenSavTable(cold)
	coldApp := cold.App("second-mission")
	coldApp.Layout(1024, 768)
	save, list, load := cold.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	coldApp.SetSaveSeams(save, list, load)
	groundAppLoad(t, coldApp, list, entries[0].Name)
	if cold.live == nil {
		t.Fatal("the cold LOAD opened no mission")
	}
	if packs, _ := secondBagHeld(cold.live.world, cold.live.mission.ids); !slices.Equal(packs[0], bagLabels([]sim.ItemInstance{elvenSavOld})) {
		t.Fatalf("cold LOAD hero pack %v, want the scroll as saved %v", packs[0], bagLabels([]sim.ItemInstance{elvenSavOld}))
	}
	secondBagArrivesHome(t, cold, elvenSavOld)
}

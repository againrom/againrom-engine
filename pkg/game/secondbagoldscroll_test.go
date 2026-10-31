package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondBagScrollTable gives f's table MagicItems rows 2 and 28, scrolls
// whose string casts spell 1 at 50, so the engine's item decoder gives
// 0x0e02 and the starting 0x0e1c a castspell effect.
func secondBagScrollTable(f *FrontEnd) {
	rows := make(dbCollection, 29)
	rows[2] = dbEntry{name: "Scroll of spell", params: []int32{100}, strings: []string{"castSpell=spell:50"}}
	rows[28] = dbEntry{name: "Scroll of spell", params: []int32{50}, strings: []string{"castSpell=spell:50"}}
	f.Table.MagicItems = rows
}

// secondBagOldScrollPack puts scroll, as an engine before the decoder gave a
// scroll its spell wrote it (type 4, no effects), as the hero's only pack
// unit.
func secondBagOldScrollPack(t *testing.T, f *FrontEnd, scroll sim.ItemInstance) {
	t.Helper()
	hero, ok := memberWithPack(f.Carried[0], []sim.ItemInstance{scroll}, f.Table, true)
	if !ok {
		t.Fatal("the hero's pack refused the scroll")
	}
	f.Carried[0] = hero
}

// secondBagArrivesHome wins mission 10 on f, arrives at town 2 and fails
// unless the bag holds want and the hero's pack nothing.
func secondBagArrivesHome(t *testing.T, f *FrontEnd, want sim.ItemInstance) {
	t.Helper()
	secondBagWin(t, f, 10)
	screen := f.TownScreen().(*townScreen)
	if at := (&destinationTownMap{screen}).enterLocation(partyPlace{kind: placeTown, id: 2}); at.Msg != "" {
		t.Fatalf("the arrival at town 2 reports %q", at.Msg)
	}
	if got := bagLabels(*f.Town.second.heroBag()); !slices.Equal(got, bagLabels([]sim.ItemInstance{want})) {
		t.Fatalf("bag at town 2 %v, want %v", got, bagLabels([]sim.ItemInstance{want}))
	}
	if left := mapload.MemberCarriedItems(f.Carried[secondHeroIndex(f.Carried)], f.Table); len(left) != 0 {
		t.Fatalf("the hero's pack kept %v", bagLabels(left))
	}
}

// A mission file holding a bought scroll with no effects, as an engine
// before the decoder gave a scroll its spell wrote it, LOADs with the scroll
// as saved, and after the victory the scroll comes home to the bag in its
// bag form at town 2. Fails when the arrival asks the saved scroll for the
// decoder's castspell effect and leaves it in the hero's hidden pack.
func TestSecondBagEffectlessScrollOfAMissionFileComesHome(t *testing.T) {
	old := sim.ItemInstance{Code: 0x0e02, Kind: 4, Price: 100}
	f, app, screen, dir := secondBagFront(t, nil)
	secondBagScrollTable(f)
	secondBagOldScrollPack(t, f, old)
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
	cold, _ := secondMissionLoadOf(t, secondMissionContent{side: 128}, dir, entries[0].Name)
	if cold.live == nil {
		t.Fatal("the cold LOAD opened no mission")
	}
	secondBagScrollTable(cold)
	if packs, _ := secondBagHeld(cold.live.world, cold.live.mission.ids); !slices.Equal(packs[0], bagLabels([]sim.ItemInstance{old})) {
		t.Fatalf("cold LOAD hero pack %v, want the scroll as saved %v", packs[0], bagLabels([]sim.ItemInstance{old}))
	}
	secondBagArrivesHome(t, cold, sim.ItemInstance{Code: old.Code, Price: old.Price})
}

// A town file whose hero pack holds the starting scroll 0x0e1c with no
// effects, as an engine before the decoder gave a scroll its spell wrote it,
// LOADs with the scroll as saved; the hero takes it into the next mission
// as saved, and after the victory it comes home to the bag in its bag form.
// Fails when the arrival leaves the saved scroll in the hero's hidden pack.
func TestSecondBagEffectlessStartingScrollOfATownFileComesHome(t *testing.T) {
	old := sim.ItemInstance{Code: 0x0e1c, Kind: 4, Price: 50}
	f, _, _, _ := secondBagFront(t, nil)
	secondBagScrollTable(f)
	secondBagOldScrollPack(t, f, old)
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatalf("town SAVE: %v", err)
	}
	cold, app, _, _ := secondBagFront(t, nil)
	secondBagScrollTable(cold)
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold town LOAD", town, err)
	}
	if got := bagLabels(mapload.MemberCarriedItems(cold.Carried[secondHeroIndex(cold.Carried)], cold.Table)); !slices.Equal(got, bagLabels([]sim.ItemInstance{old})) {
		t.Fatalf("cold town LOAD hero pack %v, want the scroll as saved %v", got, bagLabels([]sim.ItemInstance{old}))
	}
	screen := cold.TownScreen().(*townScreen)
	app.SetTown(screen)
	secondBagEnter(t, cold, app, screen)
	if packs, _ := secondBagHeld(cold.live.world, cold.live.mission.ids); !slices.Equal(packs[0], bagLabels([]sim.ItemInstance{old})) {
		t.Fatalf("mission hero pack %v, want the scroll as saved %v", packs[0], bagLabels([]sim.ItemInstance{old}))
	}
	secondBagArrivesHome(t, cold, sim.ItemInstance{Code: old.Code, Price: old.Price})
}

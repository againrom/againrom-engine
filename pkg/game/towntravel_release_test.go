package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// twoTownFront is a release front end whose campaign holds the travel
// fixture's two towns, East at the install's third map object.
func twoTownFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	c := f.Campaign.Value()
	c.Towns = travelBook(t, travelCatalog)
	f.Campaign = resolved(c, nil)
	f.Town = NewTown(c)
	return f
}

// twoTownReload is a cold LOAD of a town SAV in a fresh two-town front end.
func twoTownReload(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := twoTownFront(t)
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	if _, town, err := load("town.sav"); err != nil || !town {
		t.Fatalf("town LOAD of the written SAV: town=%v err=%v", town, err)
	}
	return g
}

// A town SAVE in East, the second town of a two-town catalog, and a cold LOAD
// land in East with West's hire kept; the screen composes from East's
// description. Loss control: the same file without its engine-state leaves
// lands in West, with West's hire gone.
func TestReleaseTwoTownSaveLoadsIntoTheSecondTown(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := twoTownFront(t)
	reachabilityWalkArrive(t, f, "Travel Hero")
	hired := 0
	for typ := 1; typ < len(f.Town.mercPool) && hired == 0; typ++ {
		if _, ok := f.Town.mercenaryHire(typ, 0); ok {
			hired = typ
		}
	}
	if hired == 0 {
		t.Fatal("West hires no mercenary")
	}
	if !f.Town.travelTo(2) {
		t.Fatalf("the party cannot travel to East: open %v, towns %v", f.Town.Open(), f.Town.travelTowns())
	}
	if f.Town.MercenaryHired(hired) {
		t.Fatal("East shows West's hire")
	}
	raw := currentTownSave(t, f)

	g := twoTownReload(t, raw)
	if g.Town.townID() != 2 {
		t.Fatalf("LOAD lands in town %d, want East", g.Town.townID())
	}
	if g.Town.MercenaryHired(hired) {
		t.Fatal("East shows West's hire after LOAD")
	}
	east := g.TownScreen().(*townScreen).townDescription()
	if east == nil || east != g.Campaign.Value().Towns.descriptions["rom1-east"] {
		t.Fatal("the loaded screen does not compose from East's description")
	}
	if !g.Town.travelTo(1) || !g.Town.MercenaryHired(hired) {
		t.Fatal("West's hire is lost across LOAD")
	}

	lost := twoTownReload(t, withoutEngineState(t, raw))
	if lost.Town.townID() != 1 || lost.Town.MercenaryHired(hired) {
		t.Fatalf("loss control: the leafless file lands in town %d, West's hire %v", lost.Town.townID(), lost.Town.MercenaryHired(hired))
	}
}

// savedTowns is the town record of a SAV's session leaf.
func savedTowns(t *testing.T, raw []byte) *currentTowns {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if a == nil || a.Session == nil {
		t.Fatal("the SAV holds no session leaf")
	}
	return a.Session.Towns
}

// missionSave makes the mission SAVE of f's running mission.
func missionSave(t *testing.T, f *FrontEnd, store SaveStore) (string, []byte) {
	t.Helper()
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE = %q: %v", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	return name, raw
}

// The shipped first game holds one town, so neither its town SAVE, with a
// squad hired, nor its mission SAVE carries a town record.
func TestReleaseShippedFirstGameSavesCarryNoTownRecord(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Record Hero")
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 5_000_000
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("fixture assumption broke: the tavern hire of type 14 is refused")
	}
	if r := savedTowns(t, currentTownSave(t, f)); r != nil {
		t.Fatalf("the shipped town SAVE writes a town record %+v", r)
	}
	m := releaseFront(t)
	m.Carried = m.ChargenParty(ui.ChargenResult{Name: "Record Hero", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}})
	if _, _, _, _, _, _, _, _, _, _, err := m.MissionOpenerWith(10, m.Carried)(); err != nil {
		t.Fatal(err)
	}
	m.LiveAdvance(3)
	if _, raw := missionSave(t, m, SaveStore{Dir: t.TempDir()}); savedTowns(t, raw) != nil {
		t.Fatalf("the shipped mission SAVE writes a town record %+v", savedTowns(t, raw))
	}
}

// A squad hired in West goes on a mission from East. East never offers its
// type meanwhile; a mission SAVE and a cold LOAD land in East with West's
// hire; the mission end clears that hire and sets West's pool to the
// squad's survivors, while East keeps its own pool.
func TestReleaseTwoTownHireGoesHomeAcrossAMissionSave(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	const typ = 14
	f := twoTownFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Squad Hero")
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 5_000_000
	eastPool := f.Town.others[2].mercPool[typ]
	if _, ok := screen.toggleMercenary(typ); !ok || eastPool <= 0 {
		t.Fatalf("fixture assumption broke: West's hire of type %d refused or East holds no squad (%d)", typ, eastPool)
	}
	squad := countTownMercenaries(f.Carried, typ)
	if !f.Town.travelTo(2) {
		t.Fatal("the party cannot travel to East")
	}
	east := f.TownScreen().(*townScreen)
	if slices.ContainsFunc(east.tavernMercenaries(), func(o TavernOffer) bool { return o.Type == typ }) {
		t.Fatal("East's tavern offers the type West's squad holds")
	}
	if _, ok := east.toggleMercenary(typ); ok {
		t.Fatal("East hires a second squad of the type West's squad holds")
	}
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(31, f.Carried)(); err != nil {
		t.Fatal(err)
	}
	f.LiveAdvance(3)
	store := SaveStore{Dir: t.TempDir()}
	name, raw := missionSave(t, f, store)
	if r := savedTowns(t, raw); r == nil || r.Town != 2 {
		t.Fatalf("the mission SAVE from East writes town record %+v", r)
	}

	g := twoTownFront(t)
	g.SetDeterministicFrames(true)
	_, _, load := g.SaveSeams(store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("mission LOAD: town=%v err=%v", town, err)
	}
	if err := g.App("two-town mission").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if g.Town.townID() != 2 || g.Town.MercenaryHired(typ) || !g.Town.mercenaryHiredElsewhere(typ) {
		t.Fatalf("mission LOAD: town %d, East hired %v, hired elsewhere %v; want East with West's hire",
			g.Town.townID(), g.Town.MercenaryHired(typ), g.Town.mercenaryHiredElsewhere(typ))
	}
	m, w := g.live.mission, g.live.world
	live := liveMercenaries(m.party, w, m.ids)[typ]
	if live != squad {
		t.Fatalf("the loaded mission holds %d of West's %d men standing", live, squad)
	}
	g.FinishMission(31, m.party, w, m.ids)
	tw := g.Town
	if tw.townID() != 2 || tw.mercenaryHiredElsewhere(typ) || tw.MercenaryHired(typ) || tw.MercenaryPool(typ) != eastPool {
		t.Fatalf("East after the mission: town %d, hired elsewhere %v, pool %d; want no hire and East's pool %d",
			tw.townID(), tw.mercenaryHiredElsewhere(typ), tw.MercenaryPool(typ), eastPool)
	}
	if west := tw.others[1]; west.mercHired[typ] || west.mercPool[typ] != live {
		t.Fatalf("West after the mission: hired %v pool %d, want no hire and pool %d", west.mercHired[typ], west.mercPool[typ], live)
	}
}

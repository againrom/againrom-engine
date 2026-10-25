package game

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mod"
	"againrom/pkg/random"
)

// The install's tavern names each NPC record once per chapter, so a row's
// key (town, chapter, npc) names one row of scenario.reg.
func TestReleaseTavernRowKeyIsUnique(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	c := releaseFront(t).Campaign.Value()
	rows := 0
	for _, n := range sortedChapters(c) {
		ch := c.Chapters[n]
		seen := map[int]bool{}
		for i := range min(len(ch.InnNPC), len(ch.Inn)) {
			if seen[ch.InnNPC[i]] {
				t.Errorf("chapter %d names npc %d twice: %v", n, ch.InnNPC[i], ch.InnNPC)
			}
			seen[ch.InnNPC[i]] = true
			rows++
		}
	}
	if _, ok := c.Towns.tavernRows(); ok || rows != 22 {
		t.Fatalf("%d rows; the install's book states rows %v", rows, ok)
	}
}

func sortedChapters(c Campaign) []int {
	var out []int
	for n := range c.Chapters {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

// tavernFixtureMods is a mods folder holding the fixture mods ids, east-town
// with its description written.
func tavernFixtureMods(t *testing.T, ids ...string) string {
	t.Helper()
	dir := eastTownMods(t)
	for _, id := range ids {
		if err := os.CopyFS(filepath.Join(dir, id), os.DirFS(filepath.Join("..", "modrt", "testdata", "mods", id))); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// tavernModFront is a front end over the install built through the
// launcher's steps with the mods ids of dir: their install files, their
// rules and set, and their town layers.
func tavernModFront(t *testing.T, dir string, acceptUnmarked bool, ids ...string) *FrontEnd {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the tavern mods need a lawful install")
	}
	res := runTownModsFor(t, BaseID(InspectInstall(root)), dir, ids...)
	f, err := NewModFrontEnd(root, res.Files)
	if err != nil {
		t.Fatal(err)
	}
	cleanupFrontAudio(t, f)
	f.randomService().SetStreamSeed(random.TownWildlife, 1)
	f.randomService().SetStreamSeed(random.AmbientBirds, 1)
	f.SetDeterministicFrames(true)
	if err := f.SetMods(res.Rules, res.Set, acceptUnmarked); err != nil {
		t.Fatal(err)
	}
	if overrides, err := f.SetModTowns(res.Towns); err != nil || len(overrides) != 0 {
		t.Fatalf("town layer: %v, overrides %v", err, overrides)
	}
	return f
}

// tavernReload is a cold LOAD of a town SAV in front end g.
func tavernReload(t *testing.T, g *FrontEnd, raw []byte) error {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	_, town, err := load("town.sav")
	if err == nil && !town {
		t.Fatal("the town SAV did not open a town")
	}
	return err
}

// tavernChapter makes chapter main the town's live chapter, as the inn
// registration witness does, keeping the loaded party and purse.
func tavernChapter(f *FrontEnd, main int) {
	c := f.Campaign.Value()
	gold := f.Town.Gold()
	f.Town = NewTown(c)
	f.Town.main = main
	f.Town.records = newCampaignRecords(c, main)
	f.Town.Arrive()
	f.Town.gold = gold
	f.Town.takeAddHeroes(main)
}

// A save written with no mod loads without one, is refused with the mod
// unless the player accepts unmarked saves, and with that keeps its live
// offers: LOAD adds no mod row, to the offers, the roster or the talk cells
// of chapter 50, which the mod changes. The chapter entered after the LOAD
// shows the mod's rows.
func TestReleaseTavernModOldSave(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	base := currentTown(t, nil, nil)
	want := base.Town.Offers(TownTavern)
	raw := currentTownSave(t, base)
	mods := tavernFixtureMods(t, "tavern-rows")
	if err := tavernReload(t, releaseFront(t), raw); err != nil {
		t.Fatalf("LOAD without the mod: %v", err)
	}
	if err := tavernReload(t, tavernModFront(t, mods, false, "tavern-rows"), raw); !errors.Is(err, ErrModMark) {
		t.Fatalf("LOAD of an unmarked save with the mod: %v, want a mod-mark refusal", err)
	}
	g := tavernModFront(t, mods, true, "tavern-rows")
	if err := tavernReload(t, g, raw); err != nil {
		t.Fatalf("LOAD of an unmarked save accepting it: %v", err)
	}
	if got := g.Town.Offers(TownTavern); !reflect.DeepEqual(got, want) {
		t.Fatalf("the loaded tavern %+v, want the save's %+v", got, want)
	}
	if m, ok := g.Town.Take(TownTavern, want[0].Index); !ok || m != want[0].Mission {
		t.Fatalf("the next action: took %d %v", m, ok)
	}

	// Chapter 50, which the mod changes: the roster and the talk cells are
	// the save's rows with the mod as without it.
	in50 := currentTown(t, nil, nil)
	tavernChapter(in50, 50)
	raw = currentTownSave(t, in50)
	plain := releaseFront(t)
	if err := tavernReload(t, plain, raw); err != nil {
		t.Fatalf("LOAD of chapter 50 without the mod: %v", err)
	}
	g = tavernModFront(t, mods, true, "tavern-rows")
	if err := tavernReload(t, g, raw); err != nil {
		t.Fatalf("LOAD of chapter 50 accepting it: %v", err)
	}
	saved := plain.Town.Offers(TownTavern)
	for _, c := range []struct {
		what      string
		plain, in []TownOffer
	}{
		{"offers", saved, g.Town.Offers(TownTavern)},
		{"roster", plain.Town.TavernRoster(), g.Town.TavernRoster()},
		{"talk cells", plain.TownScreen().(*townScreen).tavernVisibleOffers(), g.TownScreen().(*townScreen).tavernVisibleOffers()},
	} {
		if g.Town.Chapter() != 50 || !reflect.DeepEqual(c.plain, saved) || !reflect.DeepEqual(c.in, saved) {
			t.Errorf("chapter %d's %s: without the mod %+v, with it %+v, want the save's %+v", g.Town.Chapter(), c.what, c.plain, c.in, saved)
		}
	}

	// Chapter 50 entered after a LOAD in chapter 40 shows the mod's rows.
	in40 := currentTown(t, nil, nil)
	tavernChapter(in40, 40)
	raw = currentTownSave(t, in40)
	g = tavernModFront(t, mods, true, "tavern-rows")
	if err := tavernReload(t, g, raw); err != nil {
		t.Fatalf("LOAD of chapter 40 accepting it: %v", err)
	}
	if _, ok := g.Town.Won(40); !ok || g.Town.Chapter() != 50 {
		t.Fatalf("winning 40 after the LOAD: won %v, chapter %d", ok, g.Town.Chapter())
	}
	pairs := func(offers []TownOffer) [][2]int {
		var out [][2]int
		for _, o := range offers {
			out = append(out, [2]int{o.NPC, o.Mission})
		}
		return out
	}
	modRows := [][2]int{{64, 50}, {32, 31}, {41, 0}, {52, 51}}
	offers := g.Town.Offers(TownTavern)
	if !reflect.DeepEqual(pairs(offers), modRows) || !reflect.DeepEqual(g.Town.TavernRoster(), offers) ||
		!reflect.DeepEqual(g.TownScreen().(*townScreen).tavernVisibleOffers(), offers) {
		t.Fatalf("chapter 50 entered after the LOAD: offers %+v roster %+v cells %+v, want the mod's rows %v",
			offers, g.Town.TavernRoster(), g.TownScreen().(*townScreen).tavernVisibleOffers(), modRows)
	}
}

// The fixture mod on chapter 50: the tavern shows the added talk and quest
// and the changed quest, not the switched-off talk; each speaks the mod's
// text. A SAVE holds the rows and the mod mark; after a cold LOAD the talk
// is heard and the quest accepted, and after a second SAVE and LOAD the
// quest's mission opens. A LOAD without the mod is refused naming it.
func TestReleaseTavernModRowsSaveAndLoad(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	mods := tavernFixtureMods(t, "tavern-rows")
	raw := currentTownSave(t, currentTown(t, nil, nil))
	f := tavernModFront(t, mods, true, "tavern-rows")
	if err := tavernReload(t, f, raw); err != nil {
		t.Fatal(err)
	}
	tavernChapter(f, 50)
	want := []TownOffer{{Index: 0, Mission: 50, NPC: 64}, {Index: 1, Mission: 31, NPC: 32}, {Index: 2, NPC: 41}, {Index: 3, Mission: 51, NPC: 52}}
	if got := f.Town.Offers(TownTavern); !reflect.DeepEqual(got, want) {
		t.Fatalf("chapter 50's tavern %+v, want %+v", got, want)
	}
	marked := currentTownSave(t, f)
	p := autoGetCampaign(t, marked)
	if !slices.Equal(p.InnNPC, []uint16{64, 32, 41, 52}) || !slices.Equal(p.InnMission, []uint16{50, 31, 0, 51}) {
		t.Fatalf("SAVE holds npc %v missions %v", p.InnNPC, p.InnMission)
	}
	doc, err := sav.DecodeDocumentData(marked)
	if err != nil {
		t.Fatal(err)
	}
	if mark, ok, err := readModMark(doc); err != nil || !ok || !slices.ContainsFunc(mark.Mods, func(m mod.EntryRecord) bool { return m.ID == "tavern-rows" }) {
		t.Fatalf("the SAV's mod mark %+v (present %v): %v", mark.SetRecord, ok, err)
	}

	g := tavernModFront(t, mods, false, "tavern-rows")
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "town.sav"), marked, 0o644); err != nil {
		t.Fatal(err)
	}
	app, s := releaseTavernApp(t, g, store)
	if got := g.Town.Offers(TownTavern); !reflect.DeepEqual(got, want) {
		t.Fatalf("the loaded tavern %+v, want %+v", got, want)
	}
	for _, o := range want[1:3] {
		text := o.Mission
		if text == 0 {
			text = 50
		}
		path, _ := s.townTextPathFor(TownTavern, o, o.NPC)
		if wantPath, _ := TownNPCTextPath(o.NPC, text); path != wantPath || len(s.townPayloadFor(TownTavern, o, o.NPC)) == 0 {
			t.Fatalf("npc %d speaks from %s, want the mod's %s", o.NPC, path, wantPath)
		}
	}
	hearInnOfferIndex(t, app, s, 0, 2)
	hearInnOfferIndex(t, app, s, 51, 3)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	after := []TownOffer{{Index: 0, Mission: 50, NPC: 64}, {Index: 1, Mission: 31, NPC: 32}, {Index: 2, NPC: 41}}
	if got := g.Town.Offers(TownTavern); !reflect.DeepEqual(got, after) || g.Town.selectedMission() != 51 {
		t.Fatalf("after hearing: %+v selected %d, want %+v and 51", got, g.Town.selectedMission(), after)
	}
	accepted := saveInnApp(t, g, app)
	p = autoGetCampaign(t, accepted)
	announced := slices.ContainsFunc(p.Children, func(r sav.CampaignRecord) bool { return r.Mission == 51 && r.Announced })
	if !slices.Equal(p.InnNPC, []uint16{64, 32, 41}) || !slices.Equal(p.InnMission, []uint16{50, 31, 0}) || p.SelectedMission != 51 || !announced {
		t.Fatalf("SAVE after accepting: npc %v missions %v selected %d announced %v", p.InnNPC, p.InnMission, p.SelectedMission, announced)
	}

	h := tavernModFront(t, mods, false, "tavern-rows")
	if err := tavernReload(t, h, accepted); err != nil {
		t.Fatal(err)
	}
	if got := h.Town.Offers(TownTavern); !reflect.DeepEqual(got, after) || !slices.Contains(h.Town.Available(), 51) {
		t.Fatalf("the reloaded tavern %+v, available %v", got, h.Town.Available())
	}
	if err := h.App("tavern quest").OpenMission(h.MissionOpenerWith(51, h.NextParty())); err != nil {
		t.Fatalf("the accepted mod quest's mission does not open: %v", err)
	}
	if err := tavernReload(t, releaseFront(t), accepted); !errors.Is(err, ErrModMark) || !strings.Contains(err.Error(), "tavern-rows") {
		t.Fatalf("LOAD without the mod: %v, want a mod-mark refusal naming tavern-rows", err)
	}
}

// East, the fixture town, with its own tavern rows: East's tavern shows
// them and West's does not. A SAVE in East keeps them over a cold LOAD with
// both mods; without the tavern mod the LOAD is refused naming it.
func TestReleaseTavernModTownRows(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	mods := tavernFixtureMods(t, "east-tavern")
	f := tavernModFront(t, mods, false, "east-town", "east-tavern")
	reachabilityWalkArrive(t, f, "East Hero")
	f.Town.Won(10)
	f.arriveInTown()
	tw := f.Town
	tw.Arrive()
	tw.Won(20)
	west := tw.Offers(TownTavern)
	if slices.ContainsFunc(west, func(o TownOffer) bool { return o.NPC == 30 || o.NPC == 52 }) || tw.Chapter() != 30 {
		t.Fatalf("West's tavern in chapter %d shows East's rows: %+v", tw.Chapter(), west)
	}
	if !tw.travelTo(2) {
		t.Fatal("the party cannot travel to East")
	}
	east := tw.Offers(TownTavern)
	if len(east) != 2 || east[0].NPC != 30 || east[0].Mission != 0 || east[1].NPC != 52 || east[1].Mission != 31 {
		t.Fatalf("East's tavern %+v, want npc 30's talk and npc 52's mission 31", east)
	}
	raw := currentTownSave(t, f)
	g := tavernModFront(t, mods, false, "east-town", "east-tavern")
	if err := tavernReload(t, g, raw); err != nil {
		t.Fatal(err)
	}
	if got := g.Town.Offers(TownTavern); g.Town.townID() != 2 || !reflect.DeepEqual(got, east) {
		t.Fatalf("the loaded East's tavern %+v, want %+v", got, east)
	}
	if m, ok := g.Town.Take(TownTavern, east[1].Index); !ok || m != 31 {
		t.Fatalf("East's quest: took %d %v", m, ok)
	}
	if err := tavernReload(t, townModFront(t, mods, "east-town"), raw); !errors.Is(err, ErrModMark) || !strings.Contains(err.Error(), "east-tavern") {
		t.Fatalf("LOAD without east-tavern: %v, want a mod-mark refusal naming it", err)
	}
}

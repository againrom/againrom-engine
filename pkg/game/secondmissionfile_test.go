package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// checkSecondMissionFile: a second-game mission SAVE is a Bsg& file in the
// second game's records with a world half, the mission in its head,
// CurrentState/InBattle 1, an empty SpellBook/Shortcuts, the campaign tail
// standing at the mission, no engine leaf but the engine's own Asg&
// document, and the original's reader opens its world half.
func checkSecondMissionFile(t *testing.T, raw []byte, mission int) sav.DocumentData {
	t.Helper()
	if !bytes.HasPrefix(raw, []byte("Bsg&")) {
		t.Fatalf("the mission file opens with %q", raw[:4])
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Game != "rom2" || doc.World == nil || doc.Head.Mission != uint32(mission) {
		t.Fatalf("game %q world %v mission %d, want rom2 with a world half in mission %d", doc.Game, doc.World != nil, doc.Head.Mission, mission)
	}
	if v, found, err := doc.State.ReadLeaf("/CurrentState/InBattle"); err != nil || !found || v.Int32 != 1 {
		t.Fatalf("CurrentState/InBattle %+v found %v: %v", v, found, err)
	}
	if v, found, err := doc.State.ReadLeaf("/SpellBook/Shortcuts"); err != nil || !found || len(v.Bytes) != 0 {
		t.Fatalf("SpellBook/Shortcuts %+v found %v: %v", v, found, err)
	}
	if _, found, _ := doc.State.ReadLeaf(sav.NativeActionsPath); found {
		t.Fatal("the second game's records carry the engine's action leaf")
	}
	tail := doc.SecondTail
	if tail == nil || int(tail.Current) >= len(tail.Locations) {
		t.Fatalf("tail %+v", tail)
	}
	if at := tail.Locations[tail.Current]; at.Kind != uint32(placeMission) || at.ID != uint32(mission) {
		t.Fatalf("the tail stands at kind %d place %d, want mission %d", at.Kind, at.ID, mission)
	}
	for _, r := range doc.Objects {
		if r.Class == "Diary" {
			t.Fatal("the second game's records hold a Diary")
		}
	}
	engine, err := sav.EngineDocument(raw)
	if err != nil || !bytes.HasPrefix(engine, []byte("Asg&")) {
		t.Fatalf("engine document: %v", err)
	}
	sf, err := sav.Open(raw)
	if err != nil || !sf.SecondGame() || sf.World == nil {
		t.Fatalf("the original's reader: %v", err)
	}
	return doc
}

// secondMissionPosition is what a second-game mission LOAD must give back:
// the World, the mission, the explored cells, the party and the campaign's
// place, destinations, bank and purse.
type secondMissionPosition struct {
	hash               uint64
	mission            int
	explored, cells    int
	party              []string
	current            partyPlace
	available          []partyPlace
	bank               [1024]int32
	gold               int
	entities, holdings int
}

func secondMissionPositionOf(t *testing.T, f *FrontEnd) secondMissionPosition {
	t.Helper()
	if f.live == nil || f.Town == nil || f.Town.second == nil {
		t.Fatal("no live second-game mission")
	}
	w := f.live.world
	p := secondMissionPosition{hash: w.Hash(), mission: f.liveMission, current: f.Town.second.current,
		available: slices.Clone(f.Town.second.available), bank: f.Town.second.bank, gold: f.Town.gold}
	p.explored, p.cells = f.LiveExplored()
	for _, m := range f.LiveParty() {
		p.party = append(p.party, m.ID+" "+m.Name)
	}
	for _, e := range w.Entities() {
		p.entities++
		carried, _ := w.Carried(e.ID)
		p.holdings += len(carried)
	}
	return p
}

// secondMissionLoadBytes LOADs raw through the second game's load page into
// a fresh front over the 128x128 mission 10 map.
func secondMissionLoadBytes(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	return secondMissionLoadBytesOf(t, secondMissionContent{side: 128}, raw)
}

// secondMissionLoadBytesOf is secondMissionLoadBytes over a mission holding c.
func secondMissionLoadBytesOf(t *testing.T, c secondMissionContent, raw []byte) *FrontEnd {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, _ := secondMissionLoadOf(t, c, dir, "game0001.sav")
	return f
}

func checkSecondMissionPosition(t *testing.T, name string, got, want secondMissionPosition) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: the LOAD stands at %+v, want %+v", name, got, want)
	}
}

// Every second-game mission save point of a mission the engine started
// from its own town departure writes the second game's mission file: the
// save page, quick save and the timed autosave. Each LOADs back to the same
// World, mission, fog, party and campaign place; the file's records alone
// LOAD the mission with the hero. Fails while a mission SAVE writes the
// engine's Asg& document.
func TestSecondMissionSavePointsWriteTheSecondGameFile(t *testing.T) {
	f, app, screen, _ := secondBagFront(t, secondBagSample())
	dir := t.TempDir()
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	now := time.Unix(100, 0)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return now })
	secondBagEnter(t, f, app, screen)
	hero := f.live.mission.ids[0]
	e, _ := f.live.world.Entity(hero)
	f.live.enqueue(uint32(hero), int(e.X)+4, int(e.Y))
	for range 24 {
		f.live.tick()
	}
	if moved, _ := f.live.world.Entity(hero); moved.X == e.X {
		t.Fatal("the hero did not move")
	}
	want := secondMissionPositionOf(t, f)
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != want.hash {
		want = secondMissionPositionOf(t, f)
	}
	prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
		OnMap: true, Directory: dir, Name: "page", Label: "page", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepared.Commit(true); err != nil {
		t.Fatal(err)
	}
	points := map[string]string{"quick-save-1.sav": "quick save", "timed-autosave-1.sav": "timed autosave",
		"game0000.sav": "mission start autosave", "page.sav": "save page"}
	entries, err := secondSaveFiles(dir)
	if err != nil || len(entries) != len(points) {
		t.Fatalf("the save points wrote %v: %v", entries, err)
	}
	for file, point := range points {
		raw, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatalf("%s: %v", point, err)
		}
		checkSecondMissionFile(t, raw, 10)
		if point == "save page" {
			// A LOAD records the runtime IDs the document names, which the
			// uninterrupted World does not carry: the World hash is that of
			// a LOAD of the engine's own document.
			cold := secondMissionLoadBytes(t, raw)
			got := secondMissionPositionOf(t, cold)
			own, err := sav.EngineDocument(raw)
			if err != nil {
				t.Fatal(err)
			}
			engine := secondMissionLoadBytes(t, own)
			if got.hash != engine.live.world.Hash() {
				t.Fatalf("%s: the World hash is %x, the engine document's LOAD %x", point, got.hash, engine.live.world.Hash())
			}
			got.hash, want.hash = 0, 0
			checkSecondMissionPosition(t, point, got, want)
			bare := secondMissionLoadBytes(t, withoutEngineDocument(t, raw))
			if bare.live == nil || bare.liveMission != 10 || len(bare.LiveParty()) == 0 {
				t.Fatal("the file's records alone do not LOAD mission 10 with the hero")
			}
			continue
		}
		if got := secondMissionPositionOf(t, secondMissionLoadBytes(t, raw)); got.mission != 10 || !slices.Equal(got.party, want.party) {
			t.Fatalf("%s: the LOAD stands in mission %d with %v", point, got.mission, got.party)
		}
	}
}

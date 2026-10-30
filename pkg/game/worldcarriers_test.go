package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// worldCarrierSave writes our SAVE of f's live mission into dir and returns
// its file name and bytes.
func worldCarrierSave(t *testing.T, f *FrontEnd, dir string) (string, []byte) {
	t.Helper()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, filepath.Base(name)))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Base(name), raw
}

// The SAV LOAD writes the World's loaded-record carrier: a mission file whose
// document does not decode, then our SAVE of it, whose document does.
func TestTheLoadWritesTheWorldsLoadedRecord(t *testing.T) {
	f, _, dir := secondMissionLoadFile(t, secondMissionFile(secondMissionFixtureHero(), secondMissionStore(4, 0, -1)))
	if got := f.live.world.LoadedRecord(); got != sim.LoadedWithoutDocument {
		t.Fatalf("the mission file's World holds loaded record %d", got)
	}
	name, _ := worldCarrierSave(t, f, dir)
	cold, _ := secondMissionLoad(t, dir, name)
	if got := cold.live.world.LoadedRecord(); got != sim.LoadedDocument {
		t.Fatalf("our SAVE's cold World holds loaded record %d", got)
	}
}

// A SAVE leaves the unbound effects out of its World copy by what the World
// holds, not by the record the mission last loaded: a World whose carrier
// names a LOAD without a document drops them under a record that has one,
// and a World whose carrier names a document keeps them under a record that
// has none.
func TestTheSavesAbsenceReadsTheWorldNotTheLoadedRecord(t *testing.T) {
	store := sim.SavedProjectiles{FreeIndex: 8, IDs: []uint16{7}, Items: []sim.SavedProjectile{{ID: 7}}}
	for _, c := range []struct {
		name   string
		world  sim.LoadedRecord
		record *SnapshotSAVDocument
		kept   int
	}{
		{"no document in the World, a document in the record", sim.LoadedWithoutDocument, &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &sav.DocumentData{}}, 0},
		{"a document in the World, none in the record", sim.LoadedDocument, &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Unavailable: "complete SAV document unavailable"}, 1},
		{"no record", sim.NoLoadedRecord, nil, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			live := secondGameScriptWorld(t, nil, nil)
			live.SetSavedProjectiles(store)
			live.SetLoadedRecord(c.world)
			before := live.Hash()
			var s Snapshot
			s.SavedDocument = c.record
			captureWorldCarriers(live, &s)
			raw, err := live.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var w sim.World
			if err := w.UnmarshalBinary(raw); err != nil {
				t.Fatal(err)
			}
			decodeWorldCarriers(s, &w)
			applied := leaveAbsentCarriersOut(&w)
			if got := len(w.SavedProjectiles().Items); got != c.kept || applied != (c.kept == 0) {
				t.Fatalf("the SAVE's World copy keeps %d projectiles, applied %v; want %d", got, applied, c.kept)
			}
			if live.Hash() != before || len(live.SavedProjectiles().Items) != 1 {
				t.Fatal("the SAVE's absence changed the live World")
			}
		})
	}
}

// The two invariants on the second game's mission file fixture: a SAVE never
// changes the live World, and its next tick equals the next tick of a twin
// that did not SAVE; our SAV of a World with a document, loaded and saved
// again with no tick between, writes equal bytes.
func TestASaveKeepsTheLiveWorldAndAResaveItsBytes(t *testing.T) {
	file := secondMissionFile(secondMissionFixtureHero(), secondMissionStore(4, 0, -1))
	f, _, dir := secondMissionLoadFile(t, file)
	twin, _, _ := secondMissionLoadFile(t, file)
	for _, w := range []*sim.World{f.live.world, twin.live.world} {
		w.SetSavedProjectiles(sim.SavedProjectiles{FreeIndex: 8, IDs: []uint16{7}, Items: []sim.SavedProjectile{{ID: 7}}})
	}
	before := f.live.world.Hash()
	if twin.live.world.Hash() != before {
		t.Fatal("the twin LOAD differs")
	}
	name, first := worldCarrierSave(t, f, dir)
	if f.live.world.Hash() != before || f.live.world.LoadedRecord() != sim.LoadedWithoutDocument {
		t.Fatal("the SAVE changed the live World")
	}
	f.live.tick()
	twin.live.tick()
	if f.live.world.Hash() != twin.live.world.Hash() {
		t.Fatal("the tick after the SAVE differs from the twin's")
	}
	if len(first) == 0 {
		t.Fatal("the SAVE wrote nothing")
	}
	// The first resave of a mission file's SAVE is not compared: its town
	// chapter in the offer labels reads 0 after the cold LOAD (open debt of
	// the campaign projection, not of a World carrier).
	cold, _ := secondMissionLoad(t, dir, name)
	resaveDir := t.TempDir()
	resaved, second := worldCarrierSave(t, cold, resaveDir)
	again, _ := secondMissionLoad(t, resaveDir, resaved)
	loaded := again.live.world.Hash()
	_, third := worldCarrierSave(t, again, t.TempDir())
	if again.live.world.Hash() != loaded {
		t.Fatal("the resave changed the cold World")
	}
	if !bytes.Equal(second, third) {
		t.Fatalf("our SAV loaded and saved again writes %d bytes, not the %d it loaded", len(third), len(second))
	}
}

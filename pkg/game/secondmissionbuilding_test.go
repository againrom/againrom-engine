package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// secondBuildingMission LOADs the fixture mission file holding p in the
// place of its Pointer.
func secondBuildingMission(t *testing.T, p secondSavedPointer) (*FrontEnd, []byte) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), pointerSaveFileOf(p), 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := secondMissionLoadOf(t, pointerSaveMission(), dir, "game0001.sav")
	if f.live == nil || f.live.world == nil {
		t.Fatalf("the fixture file does not LOAD: %s", app.HeadlessMessage())
	}
	return f, secondPageSave(t, f, app)
}

// checkSecondStructure: w holds one saved structure, of class and with the
// Token row and Pointer word given.
func checkSecondStructure(t *testing.T, name string, w *sim.World, class sim.SavedStructureClass, row uint16, word uint32) {
	t.Helper()
	source, _, _ := w.SavedStructures()
	if len(source) != 1 || source[0].Class != class || source[0].Token0C != row || source[0].Pointer6C != word {
		t.Fatalf("%s: the World's structures are %+v, want one of class %d row %d word %#x", name, source, class, row, word)
	}
}

// secondFileStructure is the record the mission file holds for the
// fixture's structure, Token key 0x0b000001.
func secondFileStructure(t *testing.T, name string, raw []byte) sav.DocumentRecordData {
	t.Helper()
	doc := checkSecondMissionFile(t, raw, 10)
	var found []sav.DocumentRecordData
	for _, r := range doc.Objects {
		for _, v := range r.Values {
			if v.Name == "Identity" && v.Value == pointerSaveRecord().key {
				found = append(found, r)
			}
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: %d records hold the structure's key, want 1", name, len(found))
	}
	return found[0]
}

func secondRecordValue(r sav.DocumentRecordData, name string) (uint32, bool) {
	for _, v := range r.Values {
		if v.Name == name {
			return v.Value, true
		}
	}
	return 0, false
}

// A second-game mission LOADed from a file whose Building is row 330 holds
// the row; its SAVE writes the row in the Building's +0xc, and the file's
// LOAD holds it again with the World hash of the play that saved it. Its
// records alone LOAD the row too. Fails while a structure keeps one byte of
// the row (330 > 74).
func TestSecondMissionKeepsABuildingRowAbove255(t *testing.T) {
	p := pointerSaveRecord()
	p.building, p.token0C, p.word = true, 330, 0
	f, raw := secondBuildingMission(t, p)
	checkSecondStructure(t, "the LOAD", f.live.world, sim.SavedBuilding, 330, 0)
	r := secondFileStructure(t, "the SAVE", raw)
	if row, _ := secondRecordValue(r, "T0C"); r.Class != "Building" || row != 330 {
		t.Fatalf("the SAVE writes a %s of row %d, want a Building of row 330", r.Class, row)
	}
	cold := secondMissionLoadBytesOf(t, pointerSaveMission(), raw)
	if cold.live == nil {
		t.Fatal("the SAVE does not LOAD")
	}
	checkSecondStructure(t, "the SAVE's LOAD", cold.live.world, sim.SavedBuilding, 330, 0)
	if cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatalf("the SAVE's LOAD has World hash %x, the play that saved it %x", cold.live.world.Hash(), f.live.world.Hash())
	}
	bare := secondMissionLoadBytesOf(t, pointerSaveMission(), withoutEngineDocument(t, raw))
	if bare.live == nil {
		t.Fatal("the SAVE's records alone do not LOAD")
	}
	checkSecondStructure(t, "the records' LOAD", bare.live.world, sim.SavedBuilding, 330, 0)
}

// A Pointer the loaded file held (a Building and its +0x6c word,
// R2-ENGINE-603) SAVEs as a Pointer record with the same word, and the
// file's LOAD holds the Pointer with the World hash of the play that saved
// it. A LOAD of its records alone SAVEs the Pointer again. Fails while a
// SAVE writes each Pointer as a Building (DIV-4172).
func TestSecondMissionWritesAPointerBackAsAPointer(t *testing.T) {
	p := pointerSaveRecord()
	p.word = 0x01020304
	f, raw := secondBuildingMission(t, p)
	checkSecondStructure(t, "the LOAD", f.live.world, sim.SavedPointer, 0, p.word)
	check := func(name string, raw []byte) {
		t.Helper()
		r := secondFileStructure(t, name, raw)
		if word, ok := secondRecordValue(r, "P6C"); r.Class != "Pointer" || !ok || word != p.word {
			t.Fatalf("%s writes a %s with word %#x (%v), want a Pointer with %#x", name, r.Class, word, ok, p.word)
		}
	}
	check("the SAVE", raw)
	cold := secondMissionLoadBytesOf(t, pointerSaveMission(), raw)
	if cold.live == nil {
		t.Fatal("the SAVE does not LOAD")
	}
	checkSecondStructure(t, "the SAVE's LOAD", cold.live.world, sim.SavedPointer, 0, p.word)
	if cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatalf("the SAVE's LOAD has World hash %x, the play that saved it %x", cold.live.world.Hash(), f.live.world.Hash())
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), withoutEngineDocument(t, raw), 0o600); err != nil {
		t.Fatal(err)
	}
	bare, bareApp := secondMissionLoadOf(t, pointerSaveMission(), dir, "game0001.sav")
	if bare.live == nil {
		t.Fatal("the SAVE's records alone do not LOAD")
	}
	check("the records' resave", secondPageSave(t, bare, bareApp))
}

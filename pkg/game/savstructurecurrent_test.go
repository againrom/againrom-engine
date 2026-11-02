package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestUnregisteredStructureWritesCurrentGeometryAndRetainsUnknownBytes(t *testing.T) {
	document, _, _ := structureProjectionFixture(t)
	before, err := sav.CloneDocumentData(document)
	if err != nil {
		t.Fatal(err)
	}
	shared := document
	world, err := sim.NewWorld(0, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	world.DeclareStructures([]sim.Structure{{ID: 9000, Kind: 1, Col: 10, Row: 10,
		Width: 3, Height: 4, Blocking: 0x12345678, Attach: 0x87654321, Field42: 0x2345, MaxHealth: 0x3456}})
	if err := projectSavedStructures(&document, world); err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(document)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	buildings, _, err := f.Buildings()
	if err != nil {
		t.Fatal(err)
	}
	b := buildings[0]
	if b.Width != 3 || b.Height != 4 || b.Blocking != 0x12345678 || b.Attach != 0x87654321 || b.Health != 0x2345 || b.MaxHealth != 0x3456 {
		t.Fatalf("current native geometry: %+v", b)
	}
	oldFile, err := sav.Open(mustEncodeCurrentDocument(t, before))
	if err != nil {
		t.Fatal(err)
	}
	old, _, _ := oldFile.Buildings()
	// The raw record is not the geometry fields; it keeps the loaded bytes (SAV-1385).
	if !bytes.Equal(b.Base52[:], old[0].Base52[:]) || !reflect.DeepEqual(shared, before) {
		t.Fatal("unknown bytes or retained owner changed")
	}
}

// A LOAD keeps a Building's raw record, which is not its width, height and
// blocking fields, and the next SAVE writes the bytes the LOAD read.
func TestSaveAfterLoadWritesTheLoadedBuildingRecord(t *testing.T) {
	f := structureFixtureFront(t, 51, 52, 53)
	ms, err := StartMission(f.Archives.Containers, 10, f.Table, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := structureFixtureRows()
	rows[0].Base52 = [22]byte{14: 9, 15: 8, 18: 0x77, 21: 0x66}
	rows[1].Base52 = [22]byte{3: 1, 14: 1, 15: 1, 18: 1}
	payload := structureFixtureSave(rows)
	sf, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	loaded, _, err := sf.Buildings()
	if err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if loaded[i].Base52 != rows[i].Base52 {
			t.Fatalf("LOAD rewrote Building %d raw record: %x", i, loaded[i].Base52)
		}
	}
	var report OriginalSaveResume
	if err := applyOriginalStructures(ms, loaded, true, f.Table, &report, sf); err != nil {
		t.Fatal(err)
	}
	complete := completeCurrentDeadDocument(t, &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}}, payload)
	document, err := sav.DecodeDocumentData(complete)
	if err != nil {
		t.Fatal(err)
	}
	if err := projectSavedStructures(&document, ms.World); err != nil {
		t.Fatal(err)
	}
	out, err := sav.Open(mustEncodeCurrentDocument(t, document))
	if err != nil {
		t.Fatal(err)
	}
	written, _, err := out.Buildings()
	if err != nil || len(written) != len(rows) {
		t.Fatal(len(written), err)
	}
	for i := range rows {
		if written[i].Base52 != rows[i].Base52 || written[i].Width != 1 || written[i].Height != 1 || written[i].Blocking != 1 {
			t.Errorf("SAVE wrote Building %d raw record %x, fields %d %d %d", i, written[i].Base52, written[i].Width, written[i].Height, written[i].Blocking)
		}
	}
}

// A Building the World constructs has the zero record its constructor
// writes (SAV-1385); the width, height and blocking are its own fields.
func TestConstructedBuildingKeepsTheConstructorRecord(t *testing.T) {
	f := structureFixtureFront(t, 51, 52, 53)
	ms, err := StartMission(f.Archives.Containers, 10, f.Table, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := constructCurrentBuildings(ms, f.Archives.Containers, f.Table.ModTerrain()); err != nil {
		t.Fatal(err)
	}
	sources, _, present := ms.World.SavedStructures()
	if !present || len(sources) == 0 {
		t.Fatal("no constructed Building", present)
	}
	for i, s := range sources {
		if s.Base52 != [22]byte{} || s.Blocking != ms.World.Structures()[i].Blocking {
			t.Errorf("constructed Building %d record %x blocking %d", i, s.Base52, s.Blocking)
		}
	}
}

func mustEncodeCurrentDocument(t *testing.T, doc sav.DocumentData) []byte {
	t.Helper()
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

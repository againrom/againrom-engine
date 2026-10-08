package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
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
	if b.Base52[14] != 3 || b.Base52[15] != 4 || binary.LittleEndian.Uint32(b.Base52[18:]) != 0x12345678 {
		t.Fatal("overlapping geometry carrier differs", b.Base52)
	}
	oldFile, err := sav.Open(mustEncodeCurrentDocument(t, before))
	if err != nil {
		t.Fatal(err)
	}
	old, _, _ := oldFile.Buildings()
	if !bytes.Equal(b.Base52[:14], old[0].Base52[:14]) || !bytes.Equal(b.Base52[16:18], old[0].Base52[16:18]) || !reflect.DeepEqual(shared, before) {
		t.Fatal("unknown bytes or retained owner changed")
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

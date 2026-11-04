package mapedit_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// A new Vertical Wooden Bridge carries a 3x3 extension, so it saves and loads
// as the rectangle an installed bridge is; any other structure keeps its
// plain record.
func TestNewWoodenBridgeSpansARectangle(t *testing.T) {
	e := freshRecords(t)
	for _, kind := range []uint32{0x21, 0x25, 5} {
		rec, err := e.NewStructureRecord(20*256, 20*256, kind, 1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.Place(mapedit.KindStructure, rec); err != nil {
			t.Fatal(err)
		}
	}
	m, err := alm.Open(e.Bytes())
	if err != nil || len(m.Objects) != 3 {
		t.Fatal(err, len(m.Objects))
	}
	if o := m.Objects[0]; len(o.Ext) != alm.ObjectExtSize || o.Ext[0] != 3 || o.Ext[4] != 3 {
		t.Fatalf("bridge extension %v, want width 3 height 3", o.Ext)
	}
	if len(m.Objects[1].Ext) != 0 || len(m.Objects[2].Ext) != 0 {
		t.Fatal("a structure other than the vertical bridge carries an extension")
	}
}

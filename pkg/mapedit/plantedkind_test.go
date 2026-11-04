package mapedit_test

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// A map that already holds a record of the Horizontal Wooden Bridge (kind 37)
// loads it as a plain record and saves it byte for byte, whatever the
// editor's palettes offer.
func TestPlantedHorizontalBridgeRecordSavesUnchanged(t *testing.T) {
	e := freshRecords(t)
	rec, err := e.NewStructureRecord(20*256, 20*256, 37, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Place(mapedit.KindStructure, rec); err != nil {
		t.Fatal(err)
	}
	planted := e.Bytes()
	m, err := alm.Open(planted)
	if err != nil || len(m.Objects) != 1 || m.Objects[0].Kind != 37 || len(m.Objects[0].Ext) != 0 {
		t.Fatalf("planted record did not load as a plain kind 37: %v %+v", err, m.Objects)
	}
	again, err := mapedit.New(planted)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := again.SaveBytes()
	if err != nil || !bytes.Equal(saved, planted) {
		t.Fatalf("a map holding kind 37 does not save byte for byte (err %v)", err)
	}
}

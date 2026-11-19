package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/scriptprofile"
)

func TestFirstGameMapOpensForEditingWhateverTheCaller(t *testing.T) {
	raw := brushMap(0x14, 20)
	if alm.FormatOf(raw) != alm.FirstFormat {
		t.Fatal("first-game header read as the second format")
	}
	e, err := mapedit.New(raw)
	if err != nil || e.ViewOnly() {
		t.Fatalf("first-game map not editable: %v", err)
	}
	if _, err := e.SaveBytes(); err != nil {
		t.Fatalf("save: %v", err)
	}
}

func TestSecondMapFormatSurvivesCloneAndRefusesSave(t *testing.T) {
	le := binary.LittleEndian
	raw := make([]byte, 20)
	copy(raw, "M7R\x00")
	le.PutUint32(raw[4:], 20)
	le.PutUint32(raw[12:], 3)
	le.PutUint32(raw[16:], 1300)
	meta := make([]byte, 660)
	le.PutUint32(meta, 16)
	le.PutUint32(meta[4:], 16)
	for id, payload := range [][]byte{meta, make([]byte, 512), make([]byte, 256)} {
		header := make([]byte, 20)
		le.PutUint32(header, 7)
		le.PutUint32(header[4:], 20)
		le.PutUint32(header[8:], uint32(len(payload)))
		le.PutUint32(header[12:], uint32(id))
		if id == 0 {
			le.PutUint32(header[8:], 644)
		}
		raw = append(append(raw, header...), payload...)
	}
	if alm.FormatOf(raw) != alm.SecondFormat {
		t.Fatal("header version 1300 is not the second format")
	}
	e, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range []*mapedit.Editor{e, e.Clone()} {
		m, err := model.Map()
		if err != nil || m.Width != 16 || m.Height != 16 || !model.ViewOnly() {
			t.Fatalf("format lost: %v", err)
		}
		if b, err := model.SaveBytes(); err == nil || b != nil || !bytes.Equal(raw, model.Bytes()) || model.CanUndo() {
			t.Fatal("view-only save exposed bytes or changed history")
		}
		if model.Profile() != scriptprofile.Second || model.AddArm(mapedit.ArmWinUnit, 1) == nil || model.CanUndo() {
			t.Fatal("second-game script profile missing or an arm was written")
		}
	}
}

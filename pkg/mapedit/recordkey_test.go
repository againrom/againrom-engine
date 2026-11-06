package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
)

func TestRecordIdentityFollowsSplicesAndHistory(t *testing.T) {
	m, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: make([]byte, 12), Units: []synth.ALMUnit{{UnitID: 11}, {UnitID: 12}}}))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = m.Place(mapedit.KindLoot, m.GroundSack(256, 256, 1))
	_, _ = m.AddUnitStock(0)
	_, _ = m.AddUnitStock(1)
	for _, tag := range []uint32{10, 20} {
		b := make([]byte, 26)
		binary.LittleEndian.PutUint32(b, tag)
		_, _ = m.Place(mapedit.KindEnchantment, b)
	}
	unit, stock, spell := m.RecordKey(mapedit.KindUnit, 1), m.RecordKey(mapedit.KindLoot, 2), m.RecordKey(mapedit.KindEnchantment, 1)
	before := m.Bytes()
	if err := m.Delete(mapedit.KindUnit, 0); err != nil {
		t.Fatal(err)
	}
	if m.RecordKey(mapedit.KindUnit, 0) != unit || m.RecordKey(mapedit.KindLoot, 1) != stock || m.RecordKey(mapedit.KindEnchantment, 1) != spell {
		t.Fatal("unit cascade lost a surviving byte anchor")
	}
	if !m.Undo() || !bytes.Equal(before, m.Bytes()) || m.RecordKey(mapedit.KindUnit, 1) != unit || m.RecordKey(mapedit.KindLoot, 2) != stock {
		t.Fatal("Undo lost keys or bytes")
	}
	if !m.Redo() || m.RecordKey(mapedit.KindLoot, 1) != stock {
		t.Fatal("Redo lost stock key")
	}
	mark := m.Mark()
	if err := m.Delete(mapedit.KindEnchantment, 0); err != nil {
		t.Fatal(err)
	}
	if err := m.SetField(mapedit.KindLoot, 1, "additem", 1); err != nil {
		t.Fatal(err)
	}
	m.Squash(mark)
	if m.RecordKey(mapedit.KindEnchantment, 0) != spell || m.RecordKey(mapedit.KindLoot, 1) != stock {
		t.Fatal("record resize or squash lost a key")
	}
	if !m.Undo() || m.RecordKey(mapedit.KindEnchantment, 1) != spell || !m.Redo() || m.RecordKey(mapedit.KindEnchantment, 0) != spell {
		t.Fatal("squashed key history differs")
	}
}

func TestRecordIdentityReorderAndDiscardedGeneration(t *testing.T) {
	m, err := mapedit.New(scriptFixture())
	if err != nil {
		t.Fatal(err)
	}
	a, b := m.RecordKey(mapedit.KindAction, 0), m.RecordKey(mapedit.KindAction, 1)
	if err := m.Reorder(mapedit.KindAction, 0, 1); err != nil {
		t.Fatal(err)
	}
	if m.RecordKey(mapedit.KindAction, 0) != b || m.RecordKey(mapedit.KindAction, 1) != a {
		t.Fatal("reorder changed record identity")
	}
	if !m.Undo() || m.RecordKey(mapedit.KindAction, 0) != a {
		t.Fatal("Undo reorder lost identity")
	}
	copy := m.Clone()
	if err := copy.Delete(mapedit.KindAction, 0); err != nil {
		t.Fatal(err)
	}
	if m.RecordKey(mapedit.KindAction, 0) != a {
		t.Fatal("transaction changed its source's key table")
	}
	record, _ := copy.Record(mapedit.KindAction, 0)
	index, err := copy.Place(mapedit.KindAction, record)
	if err != nil {
		t.Fatal(err)
	}
	first := copy.RecordKey(mapedit.KindAction, index)
	if !copy.Undo() {
		t.Fatal("missing Undo")
	}
	index, err = copy.Place(mapedit.KindAction, record)
	if err != nil {
		t.Fatal(err)
	}
	second := copy.RecordKey(mapedit.KindAction, index)
	if first == second || first == a || second == a || copy.CanRedo() {
		t.Fatal("a new edit reused a discarded generation")
	}
}

func TestRecordIdentitySceneryAndPhysicalSections(t *testing.T) {
	m, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 16, Height: 16}))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetCell(mapedit.GridRaw, mapedit.GridOverlay, 2, 2, 1); err != nil {
		t.Fatal(err)
	}
	cell := m.SceneryKey(34)
	section := m.SectionKey(0)
	if cell == 0 || section == 0 || cell == section {
		t.Fatal("keys missing or aliased")
	}
	if err := m.SetCell(mapedit.GridRaw, mapedit.GridOverlay, 2, 2, 0); err != nil {
		t.Fatal(err)
	}
	if m.SceneryKey(34) != 0 {
		t.Fatal("cleared cell kept its object")
	}
	if !m.Undo() || m.SceneryKey(34) != cell || m.SectionKey(0) != section {
		t.Fatal("Undo lost scenery or physical section identity")
	}
	if !m.Redo() {
		t.Fatal("missing Redo")
	}
	if err := m.SetCell(mapedit.GridRaw, mapedit.GridOverlay, 2, 2, 1); err != nil {
		t.Fatal(err)
	}
	if m.SceneryKey(34) == cell {
		t.Fatal("replaced scenery reused the deleted key")
	}
	if m.RecordKey(mapedit.KindUnit, -1) != 0 || m.SectionKey(-1) != 0 || m.SceneryKey(-1) != 0 {
		t.Fatal("invalid address produced a key")
	}
}

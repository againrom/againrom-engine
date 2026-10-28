package alm

import (
	"bytes"
	"testing"
)

func TestStructureActionsReadThe11cRecordsAfter11aAnd11b(t *testing.T) {
	m := &Map{}
	if _, ok := m.StructureActions(); ok {
		t.Fatal("a map without type-11 data answered records")
	}
	m.Meta.Count11 = [3]uint32{1, 1, 2}
	m.Extension[1] = append(append(bytes.Repeat([]byte{0xa1}, rom2Type11A), bytes.Repeat([]byte{0xb1}, rom2Type11B)...),
		append(musicRecord(42, 1, 25), musicRecord(28, 0, 7)...)...)
	got, ok := m.StructureActions()
	want := []StructureAction{{ID: 42, Gate: 1, Position: 25}, {ID: 28, Gate: 0, Position: 7}}
	if !ok || len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("records %+v ok %v, want %+v", got, ok, want)
	}
	m.Extension[1] = m.Extension[1][:len(m.Extension[1])-4]
	if _, ok := m.StructureActions(); ok {
		t.Fatal("a payload shorter than its counts decoded")
	}
}

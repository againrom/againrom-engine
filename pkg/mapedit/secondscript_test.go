package mapedit_test

import (
	"bytes"
	"errors"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// secondScriptTally is the type-0 word +0x28 of a map built by
// secondPlacedMap, the sum of the three type-7 counts (R2-ASSET-255): the
// file header and the type-0 header come first.
func secondScriptTally(e *mapedit.Editor) uint32 {
	return secondLE.Uint32(e.Bytes()[2*alm.RecordHeaderSize+0x28:])
}

// secondTextNode is a node whose ten text fields hold text, as every
// installed second-game node's do (R2-ASSET-364).
func secondTextNode(t *testing.T, label string, opcode, id uint32, params ...mapedit.ScriptParam) []byte {
	t.Helper()
	rec, err := mapedit.NewScriptNode(label, opcode, id, params...)
	if err != nil {
		t.Fatal(err)
	}
	for f := 0; f < 10; f++ {
		copy(rec[0x9c+64*f:], "text field")
	}
	secondLE.PutUint32(rec[0x48:], 7)
	return rec
}

func TestSecondScriptAddsCopiesAndDeletesKeepWord28AtTheSumOfTheCounts(t *testing.T) {
	e, _ := secondOpen(t, secondPlacedMap())
	trigger, err := mapedit.NewScriptTrigger(alm.ScriptTrigger{Name: "Start", Left: []uint32{1}, Right: []uint32{1}, Cmp: []uint32{0}, Acts: []uint32{1}, Once: 1})
	if err != nil {
		t.Fatal(err)
	}
	action := secondTextNode(t, "Send", 2, 1, mapedit.ScriptParam{Type: 1, Value: 5})
	condition := secondTextNode(t, "Register", 19, 1, mapedit.ScriptParam{Type: 1, Value: 3})
	if _, err := e.PlaceAll([]mapedit.Placement{{Kind: mapedit.KindAction, Record: action}, {Kind: mapedit.KindCondition, Record: condition}, {Kind: mapedit.KindTrigger, Record: trigger}}); err != nil {
		t.Fatal(err)
	}
	if got := secondScriptTally(e); got != 3 {
		t.Fatalf("after three records the word +0x28 is %d, want 3", got)
	}
	at, err := e.CopyTrigger(0, "Start copy", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := secondScriptTally(e); got != 6 {
		t.Fatalf("after the copy the word +0x28 is %d, want 6", got)
	}
	for _, c := range []struct {
		k   mapedit.Kind
		src []byte
	}{{mapedit.KindAction, action}, {mapedit.KindCondition, condition}} {
		got, err := e.Record(c.k, 1)
		if err != nil {
			t.Fatal(err)
		}
		want := bytes.Clone(c.src)
		secondLE.PutUint32(want[alm.ScriptNodeID:], 2)
		if !bytes.Equal(got, want) {
			t.Fatalf("the copied %v node is not the source record under id 2", c.k)
		}
	}
	m := secondReload(t, e)
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	if tr := sc.Triggers[at]; tr.Name != "Start copy" || tr.Left[0] != 2 || tr.Acts[0] != 2 {
		t.Fatalf("the copy reads %+v", tr)
	}
	if err := e.Delete(mapedit.KindAction, 0); err != nil {
		t.Fatal(err)
	}
	if got := secondScriptTally(e); got != 5 {
		t.Fatalf("after a deletion the word +0x28 is %d, want 5", got)
	}
	e.Undo()
	if got := secondScriptTally(e); got != 6 {
		t.Fatalf("an undone deletion leaves the word +0x28 at %d, want 6", got)
	}
}

// A second-game action 65538 is a compiler parameter, not the first game's
// drop zone: a copy of its trigger is refused in the second game's word.
func TestSecondScriptCopyOfACompilerParameterIsRefusedInItsOwnWord(t *testing.T) {
	e, _ := secondOpen(t, secondPlacedMap())
	trigger, err := mapedit.NewScriptTrigger(alm.ScriptTrigger{Name: "Start", Acts: []uint32{1}, Once: 1})
	if err != nil {
		t.Fatal(err)
	}
	action := secondTextNode(t, "Param", 0x10002, 1, mapedit.ScriptParam{Type: 5, Value: 11}, mapedit.ScriptParam{Type: 6, Value: 41})
	if _, err := e.PlaceAll([]mapedit.Placement{{Kind: mapedit.KindAction, Record: action}, {Kind: mapedit.KindTrigger, Record: trigger}}); err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(e.Bytes())
	_, err = e.CopyTrigger(0, "Start copy", nil)
	var r *mapedit.CopyRefusalError
	if !errors.As(err, &r) || r.Word != "editor.second.copycompilerparameter" || r.ID != 1 {
		t.Fatalf("the copy of a compiler parameter's trigger answers %v (%+v), want the second game's refusal of action 1", err, r)
	}
	if !bytes.Equal(e.Bytes(), before) {
		t.Fatal("a refused copy changed the map")
	}
}

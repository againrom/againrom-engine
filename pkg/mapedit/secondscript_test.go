package mapedit_test

import (
	"bytes"
	"slices"
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

// A second-game action 65538 adds its two bytes to the compiler's list
// whatever trigger names it (R2-ENGINE-1727): a copy of its trigger names
// the same node, so the map holds one such node, and the other action is
// copied under a new id.
func TestSecondScriptCopyNamesTheCompilerParameterItself(t *testing.T) {
	e, _ := secondOpen(t, secondPlacedMap())
	trigger, err := mapedit.NewScriptTrigger(alm.ScriptTrigger{Name: "Start", Acts: []uint32{1, 2}, Once: 1})
	if err != nil {
		t.Fatal(err)
	}
	param := secondTextNode(t, "Param", 0x10002, 1, mapedit.ScriptParam{Type: 5, Value: 11}, mapedit.ScriptParam{Type: 6, Value: 41})
	event := secondTextNode(t, "Event", 2, 2, mapedit.ScriptParam{Type: 1, Value: 7})
	if _, err := e.PlaceAll([]mapedit.Placement{{Kind: mapedit.KindAction, Record: param}, {Kind: mapedit.KindAction, Record: event},
		{Kind: mapedit.KindTrigger, Record: trigger}}); err != nil {
		t.Fatal(err)
	}
	at, err := e.CopyTrigger(0, "Start copy", nil)
	if err != nil {
		t.Fatalf("the copy of a compiler parameter's trigger: %v", err)
	}
	view, err := e.Map()
	if err != nil {
		t.Fatal(err)
	}
	sc, err := view.Script()
	if err != nil {
		t.Fatal(err)
	}
	if got := sc.Triggers[at].Acts; len(got) < 2 || got[0] != 1 || got[1] != 3 || slices.ContainsFunc(got[2:], func(id uint32) bool { return id != 0 }) {
		t.Fatalf("the copy names actions %v, want [1 3]", got)
	}
	params := 0
	for _, n := range sc.Actions {
		if n.Opcode == 0x10002 {
			params++
		}
	}
	if params != 1 || len(sc.Actions) != 3 {
		t.Fatalf("the map holds %d compiler parameters among %d actions, want 1 among 3", params, len(sc.Actions))
	}
	if got := secondScriptTally(e); int(got) != len(sc.Actions)+len(sc.Conditions)+len(sc.Triggers) {
		t.Fatalf("the word +0x28 is %d after the copy", got)
	}
}

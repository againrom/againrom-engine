package mapedit_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

func scriptOf(t *testing.T, e *mapedit.Editor) alm.Script {
	t.Helper()
	m, err := e.Map()
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

// TestExtendScriptMovesTheTriggersAsOneEdit extends a synthetic map: the
// trigger moves into the Againrom record with the hundred registers as its
// variables, the type-7 record keeps the nodes, and one undo restores the
// exact bytes.
func TestExtendScriptMovesTheTriggersAsOneEdit(t *testing.T) {
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: scriptPayload()})
	e := newEditor(t, b)
	if e.Extended() {
		t.Fatal("an original map reads as extended")
	}
	if err := e.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	sc := scriptOf(t, e)
	want := alm.ScriptTrigger{Name: "Win", Left: []uint32{1}, Right: []uint32{2}, Cmp: []uint32{0}, Acts: []uint32{2}, Group: []uint32{0}}
	if !e.Extended() || !sc.Extended || sc.Variables != 100 || len(sc.Triggers) != 1 || len(sc.Conditions) != 2 || len(sc.Actions) != 2 {
		t.Fatalf("extended script %+v", sc)
	}
	if got := sc.Triggers[0]; got.Name != want.Name || !slices.Equal(got.Left, want.Left) || !slices.Equal(got.Right, want.Right) ||
		!slices.Equal(got.Cmp, want.Cmp) || !slices.Equal(got.Acts, want.Acts) || got.Once != 0 {
		t.Fatalf("trigger %+v, want %+v", got, want)
	}
	if e.Count(mapedit.KindTrigger) != 1 || e.ScriptVariables() != 100 {
		t.Fatalf("count %d variables %d", e.Count(mapedit.KindTrigger), e.ScriptVariables())
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("one undo did not restore the original map")
	}
}

// TestExtendedTriggersEditThroughTheSectionMachinery writes an extended
// trigger past the original record, places, reorders and deletes triggers,
// adds an arm across the two payloads and sets the variable count, each
// read back through the map's decoder.
func TestExtendedTriggersEditThroughTheSectionMachinery(t *testing.T) {
	e := newEditor(t, synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: scriptPayload()}))
	if err := e.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	wide := alm.ScriptTrigger{Left: []uint32{1, 1, 2, 1}, Right: []uint32{2, 2, 1, 1}, Cmp: []uint32{0, 1, 2, 3}, Group: []uint32{0, 0, 1, 1},
		Acts: []uint32{2, 2, 2, 2, 2}, Once: 1}
	if err := e.SetTriggerSlots(0, wide); err != nil {
		t.Fatal(err)
	}
	got := scriptOf(t, e).Triggers[0]
	if got.Name != "Win" || !slices.Equal(got.Group, wide.Group) || !slices.Equal(got.Left, wide.Left) || len(got.Acts) != 5 || got.Once != 1 {
		t.Fatalf("wide trigger %+v", got)
	}
	if err := e.SetFields(mapedit.KindTrigger, 0, mapedit.FieldSet{Key: "right3", Value: 2}, mapedit.FieldSet{Key: "action4", Value: 1}); err != nil {
		t.Fatal(err)
	}
	if got := scriptOf(t, e).Triggers[0]; got.Right[3] != 2 || got.Acts[4] != 1 {
		t.Fatalf("field writes %+v", got)
	}
	if err := e.SetFields(mapedit.KindTrigger, 0, mapedit.FieldSet{Key: "group1", Value: 2}); err == nil {
		t.Fatal("a group numbering the record refuses was written")
	}
	rec, err := e.NewTrigger(alm.ScriptTrigger{Name: "Second", Acts: []uint32{1}})
	if err != nil {
		t.Fatal(err)
	}
	if i, err := e.Place(mapedit.KindTrigger, rec); err != nil || i != 1 {
		t.Fatalf("place %d %v", i, err)
	}
	if err := e.Reorder(mapedit.KindTrigger, 1, 0); err != nil {
		t.Fatal(err)
	}
	if sc := scriptOf(t, e); sc.Triggers[0].Name != "Second" || sc.Triggers[1].Name != "Win" {
		t.Fatalf("reordered %q %q", sc.Triggers[0].Name, sc.Triggers[1].Name)
	}
	if err := e.AddArm(mapedit.ArmWinUnit, 7); err != nil {
		t.Fatal(err)
	}
	if sc := scriptOf(t, e); len(sc.Triggers) != 3 || len(sc.Conditions) != 4 || len(sc.Actions) != 3 {
		t.Fatalf("arm: %d triggers, %d conditions, %d actions", len(sc.Triggers), len(sc.Conditions), len(sc.Actions))
	}
	if err := e.Delete(mapedit.KindTrigger, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.SetScriptVariables(150); err != nil {
		t.Fatal(err)
	}
	if sc := scriptOf(t, e); len(sc.Triggers) != 2 || sc.Triggers[0].Name != "Win" || sc.Variables != 150 {
		t.Fatalf("after delete %d triggers, first %q, %d variables", len(sc.Triggers), sc.Triggers[0].Name, sc.Variables)
	}
}

// TestFoldScriptWritesTheOriginalRecords folds an extended map that fits the
// original record again: its triggers are the bytes NewScriptTrigger writes
// and the record is gone. A trigger the original cannot hold, or a variable
// past the hundred registers, refuses the fold with nothing written.
func TestFoldScriptWritesTheOriginalRecords(t *testing.T) {
	e := newEditor(t, synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: scriptPayload()}))
	if err := e.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	wide := alm.ScriptTrigger{Left: []uint32{1, 1, 2, 1}, Right: []uint32{2, 2, 1, 1}, Cmp: []uint32{0, 0, 0, 0}, Acts: []uint32{2}}
	if err := e.SetTriggerSlots(0, wide); err != nil {
		t.Fatal(err)
	}
	held := e.Bytes()
	var limit *mapedit.ScriptLimitError
	if err := e.FoldScript(); !errors.As(err, &limit) || limit.Limit != mapedit.LimitPairs || !bytes.Equal(e.Bytes(), held) {
		t.Fatalf("a fold of four pairs answered %v", err)
	}
	fits := alm.ScriptTrigger{Name: "Win", Left: []uint32{1, 2}, Right: []uint32{2, 1}, Cmp: []uint32{0, 4}, Acts: []uint32{2, 1}, Once: 1}
	if err := e.SetTriggerSlots(0, fits); err != nil {
		t.Fatal(err)
	}
	if err := e.SetScriptVariables(101); err != nil {
		t.Fatal(err)
	}
	if err := e.FoldScript(); !errors.As(err, &limit) || limit.Limit != mapedit.LimitVariables {
		t.Fatalf("a fold of 101 variables answered %v", err)
	}
	if err := e.SetScriptVariables(100); err != nil {
		t.Fatal(err)
	}
	if err := e.FoldScript(); err != nil {
		t.Fatal(err)
	}
	if e.Extended() {
		t.Fatal("the folded map still carries the record")
	}
	want, err := mapedit.NewScriptTrigger(fits)
	if err != nil {
		t.Fatal(err)
	}
	if rec, err := e.Record(mapedit.KindTrigger, 0); err != nil || !bytes.Equal(rec, want) {
		t.Fatalf("folded record % x (%v), want % x", rec, err, want)
	}
}

// TestOriginalTriggerWritesNameTheLimitTheRecordLifts refuses, on an
// original map, a fourth pair, a fifth action and a second group with a
// *ScriptLimitError the extended record lifts.
func TestOriginalTriggerWritesNameTheLimitTheRecordLifts(t *testing.T) {
	e := newEditor(t, synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: scriptPayload()}))
	for name, c := range map[string]struct {
		t     alm.ScriptTrigger
		limit mapedit.ScriptLimit
	}{
		"fourth pair":  {alm.ScriptTrigger{Left: []uint32{1, 1, 1, 1}, Right: []uint32{2, 2, 2, 2}}, mapedit.LimitPairs},
		"fifth action": {alm.ScriptTrigger{Left: []uint32{1}, Right: []uint32{2}, Acts: []uint32{2, 2, 2, 2, 2}}, mapedit.LimitActions},
		"second group": {alm.ScriptTrigger{Left: []uint32{1, 1}, Right: []uint32{2, 2}, Group: []uint32{0, 1}}, mapedit.LimitGroups},
	} {
		var limit *mapedit.ScriptLimitError
		if err := e.SetTriggerSlots(0, c.t); !errors.As(err, &limit) || limit.Limit != c.limit || !limit.Lifted() {
			t.Fatalf("%s answered %v", name, err)
		}
	}
	if e.CanUndo() {
		t.Fatal("a refused write entered the history")
	}
}

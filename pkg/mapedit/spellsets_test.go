package mapedit_test

import (
	"bytes"
	"slices"
	"testing"

	"againrom/pkg/mapedit"
)

func TestTrapStrokeMembershipSurvivesEditsIndicesAndHistory(t *testing.T) {
	e := freshRecords(t)
	first, err := e.AddSpellFrom(mapedit.SpellConstructor(0), 12, 13, 0)
	if err != nil {
		t.Fatal(err)
	}
	before := e.Bytes()
	mark := e.Mark()
	var group []int
	for range 2 {
		i, err := e.AddSpellFrom(mapedit.SpellConstructor(0), 12, 13, 0)
		if err != nil {
			t.Fatal(err)
		}
		group = append(group, i)
	}
	if err := e.GroupSpellRecords(group); err != nil {
		t.Fatal(err)
	}
	e.Squash(mark)
	added := e.Bytes()
	if len(e.SpellSet(first)) != 0 || !slices.Equal(e.SpellSet(group[0]), group) {
		t.Fatal("identical outsider entered stroke")
	}
	if !e.Undo() || !bytes.Equal(before, e.Bytes()) || len(e.SpellSet(first)) != 0 {
		t.Fatal("stroke undo")
	}
	if !e.Redo() || !bytes.Equal(added, e.Bytes()) || !slices.Equal(e.SpellSet(group[1]), group) {
		t.Fatal("stroke redo")
	}
	copy := e.Clone()
	if err := e.Delete(mapedit.KindEnchantment, first); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(e.SpellSet(0), []int{0, 1}) || !slices.Equal(copy.SpellSet(group[0]), group) {
		t.Fatal("delete renumber or clone isolation")
	}
	if err := e.SetFields(mapedit.KindEnchantment, 0, mapedit.FieldSet{Key: "sourcex", Value: 24}, mapedit.FieldSet{Key: "power", Value: 40}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(e.SpellSet(1), []int{0, 1}) {
		t.Fatal("field edit lost membership")
	}
	if _, err := e.Place(mapedit.KindLoot, e.GroundSack(2*256, 3*256, 257)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(e.SpellSet(1), []int{0, 1}) {
		t.Fatal("section splice lost membership")
	}
	reopened, err := mapedit.New(e.Bytes())
	if err != nil || len(reopened.SpellSet(0)) != 0 || !bytes.Equal(e.Bytes(), reopened.Bytes()) {
		t.Fatal("session membership serialized", err)
	}
	if err := e.Delete(mapedit.KindEnchantment, 0); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(e.SpellSet(0), []int{0}) {
		t.Fatal("deleted member did not leave stroke")
	}
	if !e.Undo() || !slices.Equal(e.SpellSet(1), []int{0, 1}) {
		t.Fatal("member undo lost group")
	}
	if !e.Redo() || !slices.Equal(e.SpellSet(0), []int{0}) {
		t.Fatal("member redo lost group")
	}
}

func TestTrapStrokeMembershipDropsRedoBranch(t *testing.T) {
	e := freshRecords(t)
	mark := e.Mark()
	i, err := e.AddSpellFrom(mapedit.SpellConstructor(0), 12, 13, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.GroupSpellRecords([]int{i}); err != nil {
		t.Fatal(err)
	}
	e.Squash(mark)
	if !e.Undo() {
		t.Fatal("undo")
	}
	j, err := e.AddSpellFrom(mapedit.SpellConstructor(0), 12, 13, 0)
	if err != nil || e.CanRedo() || len(e.SpellSet(j)) != 0 {
		t.Fatal("new identical record inherited abandoned group", err)
	}
}

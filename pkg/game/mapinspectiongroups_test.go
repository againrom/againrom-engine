package game

import (
	"bytes"
	"image"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// A group action over two units is one undo step: one Undo restores the
// bytes from before it and Redo the grouped bytes.
func TestMapEditorGroupActionIsOneUndoStep(t *testing.T) {
	refs := []ui.RecordRef{{Kind: "Unit", Index: 0}, {Kind: "Unit", Index: 1}}
	for _, group := range []int{7, 0} {
		s := objectEdits(t, groupEditor(t, groupMap(t, 1)))
		if group == 0 {
			if _, err := s.SetUnitsGroup(refs, 7); err != nil {
				t.Fatal(err)
			}
		}
		before := bytes.Clone(s.model.Bytes())
		next, err := s.SetUnitsGroup(refs, group)
		if err != nil || next == nil {
			t.Fatalf("group %d: %v", group, err)
		}
		after := bytes.Clone(next.SourceBytes)
		undone, err := s.Undo()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(undone.SourceBytes, before) {
			t.Errorf("group %d: one Undo did not restore the bytes from before the action", group)
		}
		redone, err := s.Redo()
		if err != nil || !bytes.Equal(redone.SourceBytes, after) {
			t.Errorf("group %d: Redo did not restore the grouped bytes: %v", group, err)
		}
	}
}

// Ctrl+7 and the selection popup's Remove from group, each undone and redone
// by the keys.
func TestMapEditorGroupKeyAndButtonUndoInOneStep(t *testing.T) {
	raw := groupMap(t, 1)
	e := groupEditor(t, raw)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	if err := e.Key("togroup7"); err != nil {
		t.Fatal(err)
	}
	grouped := bytes.Clone(e.Document().SourceBytes)
	m, err := alm.Open(grouped)
	if err != nil || m.Units[0].GroupID != 7 || m.Units[1].GroupID != 7 {
		t.Fatalf("Ctrl+7 did not group both units: %v", err)
	}
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("one Undo after Ctrl+7 did not restore the original bytes")
	}
	if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, grouped) {
		t.Fatalf("Redo after Ctrl+7 did not restore the grouped bytes: %v", err)
	}
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	c := &editorClicker{t: t}
	r, ok := e.SelectionButtonRect(1)
	c.click(e, r, ok, "Remove from group")
	if bytes.Equal(e.Document().SourceBytes, grouped) {
		t.Fatal("Remove from group wrote nothing")
	}
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e.Document().SourceBytes, grouped) {
		t.Fatal("one Undo after Remove from group did not restore the grouped bytes")
	}
}

package game

import (
	"bytes"
	"testing"

	"againrom/pkg/ui"
)

// The object popup's Apply writes its changed fields through SetFields: one
// history entry with the bytes the field-by-field path writes, and a refused
// field refuses the whole set.
func TestSetFieldsIsOneEditWithTheFieldByFieldBytes(t *testing.T) {
	b := inspectionFixtureBytes(t)
	_, a, _ := openTestMapEditor(t, b)
	_, c, _ := openTestMapEditor(t, b)
	values := []ui.InspectionFieldValue{{Key: "hp", Value: 77}, {Key: "group", Value: 3}}
	set, ok := a.Document().Edits.(ui.InspectionFieldSet)
	if !ok {
		t.Fatal("the map host has no SetFields")
	}
	together, err := set.SetFields("Unit", 0, values)
	if err != nil || together == nil {
		t.Fatal("SetFields", err)
	}
	var single *ui.InspectionDocument
	for _, v := range values {
		if single, err = c.Document().Edits.SetField("Unit", 0, v.Key, v.Value); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(together.SourceBytes, single.SourceBytes) || bytes.Equal(together.SourceBytes, b) {
		t.Fatal("SetFields wrote other bytes than SetField field by field")
	}
	undone, err := a.Document().Edits.Undo()
	if err != nil || !bytes.Equal(undone.SourceBytes, b) || a.Document().Edits.CanUndo() {
		t.Fatal("one Undo did not restore the source bytes", err)
	}
	refused := []ui.InspectionFieldValue{{Key: "hp", Value: 12}, {Key: "owner", Value: 999999}}
	if next, err := set.SetFields("Unit", 0, refused); err == nil || next != nil {
		t.Fatal("a refused field was not refused", err)
	}
	if a.Document().Edits.CanUndo() || a.Document().Edits.Dirty() {
		t.Fatal("a refused set changed the history")
	}
	redone, err := a.Document().Edits.Redo()
	if err != nil || !bytes.Equal(redone.SourceBytes, together.SourceBytes) {
		t.Fatal("the refused set lost the redo entry", err)
	}
}

package game

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/alm"
)

// TestReleaseMapEditorDeleteUsedGroupAsksFirst deletes both members of
// scenario/10.alm group 1, which a script names, by the Delete key and by the
// selection popup's Delete button. The warning comes first and Cancel leaves
// the bytes; Empty it deletes, Undo restores the bytes and Redo the deletion.
func TestReleaseMapEditorDeleteUsedGroupAsksFirst(t *testing.T) {
	x := installedInspector(t)
	for _, route := range []string{"key", "button"} {
		t.Run(route, func(t *testing.T) {
			e := openGroupMap(t, x)
			original := bytes.Clone(e.Document().SourceBytes)
			m, err := alm.Open(original)
			if err != nil {
				t.Fatal(err)
			}
			var members []int
			for i, u := range m.Units {
				if u.GroupID == 1 {
					members = append(members, i)
				}
			}
			if len(members) != 2 || scriptGroupRefs(t, m, 1) == 0 {
				t.Fatalf("group 1 holds units %v with %d script references", members, scriptGroupRefs(t, m, 1))
			}
			selectUnits(t, e, members...)
			c := &editorClicker{t: t}
			press := func() {
				if route == "key" {
					if err := e.Key("delete"); err != nil {
						t.Fatal(err)
					}
					return
				}
				r, ok := e.SelectionButtonRect(2)
				c.click(e, r, ok, "Delete")
			}
			press()
			if len(e.PromptNotes()) == 0 || !bytes.Equal(e.Document().SourceBytes, original) {
				t.Fatalf("Delete did not ask first: notes %v", e.PromptNotes())
			}
			r, ok := e.PromptButtonRect(1)
			c.click(e, r, ok, "Cancel")
			if len(e.PromptNotes()) != 0 || !bytes.Equal(e.Document().SourceBytes, original) {
				t.Fatal("Cancel changed the map or left the question")
			}
			press()
			r, ok = e.PromptButtonRect(0)
			c.click(e, r, ok, "Empty it")
			deleted := bytes.Clone(e.Document().SourceBytes)
			after, err := alm.Open(deleted)
			if err != nil || len(after.Units) != len(m.Units)-2 {
				t.Fatalf("Empty it left %d units of %d: %v", len(after.Units), len(m.Units), err)
			}
			if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, original) {
				t.Fatalf("Undo did not restore the bytes: %v", err)
			}
			if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, deleted) {
				t.Fatalf("Redo did not restore the deletion: %v", err)
			}
		})
	}
}

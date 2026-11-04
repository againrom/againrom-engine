package ui

import (
	"image"
	"testing"
)

// A bar entry opens no branch, but a record of its category selected
// afterwards by another route opens its branch: a press on a trigger's area
// label for Scripts, whose entry selects nothing, a press on the player's
// row for Players.
func TestMapEditorTreeFollowsALaterSelectionOfTheEntrysRecord(t *testing.T) {
	t.Run("scripts", func(t *testing.T) {
		e, _, _ := panelsEditor(t, 1920, 1080, "")
		e.doc.Script.Areas = []InspectionArea{{Trigger: 0, Rect: image.Rect(20, 20, 24, 24)}}
		e.tree.closed = map[string]bool{"scripts": true}
		e.treeSpec()
		chromeClick(e, barPoint(t, e, "scripts"))
		e.treeSpec()
		if e.selected != -1 || !e.tree.closed["scripts"] {
			t.Fatalf("the Scripts entry selected %d, branch closed %v", e.selected, e.tree.closed["scripts"])
		}
		r, ok := e.AreaLabelRect(0)
		if !ok {
			t.Fatal("the trigger's area label is not on the map")
		}
		modesClick(e, r.Min.Add(image.Pt(2, 2)))
		if e.treeSpec(); e.selected != 7 || e.tree.closed["scripts"] {
			t.Fatalf("a press on the label selected %d, branch closed %v", e.selected, e.tree.closed["scripts"])
		}
	})
	t.Run("players", func(t *testing.T) {
		e, _, _ := panelsEditor(t, 1920, 1080, "")
		e.tree.closed = map[string]bool{"players": true}
		e.treeSpec()
		chromeClick(e, barPoint(t, e, "players"))
		e.treeSpec()
		sel := e.selected
		if !e.tree.closed["players"] {
			t.Fatal("the Players entry opened the branch")
		}
		e.Select(sel, false)
		if e.treeSpec(); e.tree.closed["players"] {
			t.Fatalf("selecting player record %d again left the branch closed", sel)
		}
	})
}

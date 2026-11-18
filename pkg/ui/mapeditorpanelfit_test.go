package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
)

// The left panel narrows to leave the map its least width. The tree still
// opens floating group windows and their Delete group button.
func TestMapEditorEveryGroupOpensAndDeletesAtNarrowWindows(t *testing.T) {
	for _, w := range []int{640, 700, 760, 800} {
		e, p, _ := panelsEditor(t, w, 480, "")
		p.units[2].group, p.units[4].group = 30, 30
		if err := e.Open("map0"); err != nil {
			t.Fatal(err)
		}
		e.Layout(w, 480)
		_ = e.Key("objectspanel")
		_ = e.Key("objectspanel")
		if !e.panelOpen(panelLeft) || e.panelsShown()[panelLeft] < sidePanels[panelLeft].least {
			t.Fatalf("width %d: the left panel shows %v", w, e.panelsShown())
		}
		for _, g := range []editorGroup{{player: 1, group: 30}, {player: 2, group: 30}, {player: 1, group: 1}, {player: 2, group: 2}} {
			chromeClick(e, barPoint(t, e, "groups"))
			row := slices.IndexFunc(e.treeRowsNow(), func(r treeRow) bool {
				return r.group.set && r.group.player == g.player && r.group.group == g.group
			})
			if row < 0 {
				t.Fatalf("width %d: the tree has no row for group %d of player %d: %q", w, g.group, g.player, e.TreeLines())
			}
			clickTreeRow(t, e, row)
			if !e.groupSel.set || e.groupSel.player != g.player || e.groupSel.group != g.group {
				t.Fatalf("width %d: the tree row of group %d of player %d chose %+v", w, g.group, g.player, e.groupSel)
			}
			if g.group != 30 {
				continue
			}
			r, ok := e.FindControl(ControlQuery{Role: ControlGroupDelete})
			if !ok || !r.In(image.Rectangle{Max: image.Pt(w, 480)}) || e.chromeHit(center(r)) != popObject {
				t.Fatalf("width %d: Delete group at %v reaches %d", w, r, e.chromeHit(center(r)))
			}
			before := len(p.calls.lines)
			chromeClick(e, center(r))
			unit := 2
			if g.player == 2 {
				unit = 4
			}
			if got := strings.Join(p.calls.lines[before:], "|"); got != fmt.Sprintf("SetUnitsGroup [{Unit %d}] 0", unit) {
				t.Fatalf("width %d: Delete group of group 30 of player %d called %q", w, g.player, got)
			}
		}
	}
}

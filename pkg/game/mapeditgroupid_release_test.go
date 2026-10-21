package game

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// groupIDScenario is the installed map whose script names group 2 of player
// 2 while player 3 owns units.
const groupIDScenario = "scenario/10.alm"

// scriptGroupRefs counts the script nodes of m that name group id.
func scriptGroupRefs(t *testing.T, m *alm.Map, id uint32) int {
	t.Helper()
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, nodes := range [][]alm.ScriptNode{sc.Actions, sc.Conditions} {
		for _, nd := range nodes {
			for p, tag := range nd.Type {
				if tag == 2 && nd.Value[p] == id {
					n++
				}
			}
		}
	}
	return n
}

// openGroupMap opens groupIDScenario in the editor.
func openGroupMap(t *testing.T, x *MapInspector) *ui.MapEditor {
	t.Helper()
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(groupIDScenario); err != nil {
		t.Fatal(err)
	}
	return e
}

// recordOfUnit is the document record of the map's unit index.
func recordOfUnit(t *testing.T, e *ui.MapEditor, index int) int {
	t.Helper()
	for i, r := range e.Document().Records {
		if r.Kind == "Unit" && r.Index == index {
			return i
		}
	}
	t.Fatalf("no record of unit %d", index)
	return -1
}

// selectUnits selects the map's units by index through the pointer: the
// first by a jump to it, each other by a Shift click on its cell. The window
// is made wide so the record popup stands clear of the units.
func selectUnits(t *testing.T, e *ui.MapEditor, units ...int) {
	t.Helper()
	if !trySelectUnits(t, e, units...) {
		t.Fatalf("the pointer selected %v, want units %v", e.SelectedRecords(), units)
	}
}

// trySelectUnits is selectUnits reporting whether the pointer selected
// exactly the units.
func trySelectUnits(t *testing.T, e *ui.MapEditor, units ...int) bool {
	t.Helper()
	now := time.Unix(1, 0)
	e.Layout(2400, 1400)
	e.Select(recordOfUnit(t, e, units[0]), true)
	for _, u := range units[1:] {
		p := e.CellPoint(e.Document().Records[recordOfUnit(t, e, u)].Cell)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true, Shift: true}, now)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, Shift: true}, now)
	}
	var want []ui.RecordRef
	for _, u := range units {
		want = append(want, ui.RecordRef{Kind: "Unit", Index: u})
	}
	return fmt.Sprint(e.SelectedRecords()) == fmt.Sprint(want)
}

// nearUnits are the first unit of owner and the one of owner nearest it.
func nearUnits(t *testing.T, m *alm.Map, owner uint32) (int, int) {
	t.Helper()
	first, near, best := -1, -1, 0
	for i, u := range m.Units {
		if u.Owner != owner {
			continue
		}
		if first < 0 {
			first = i
			continue
		}
		dx, dy := int(u.X>>8)-int(m.Units[first].X>>8), int(u.Y>>8)-int(m.Units[first].Y>>8)
		if d := dx*dx + dy*dy; d > 0 && (near < 0 || d < best) {
			near, best = i, d
		}
	}
	if near < 0 {
		t.Fatalf("player %d owns fewer than two units on separate cells", owner)
	}
	return first, near
}

// TestReleaseMapEditorNewGroupNumberIsMapWide groups the two units of player
// 3 on a map whose script names player 2's group 2 and their own group 1.
// Ctrl+2 skips them, since group 2 is player 2's, and writes nothing; To
// group... > New first warns that group 1, which triggers name, would be left
// empty, then gives them a number no unit carries; the script's group 2 keeps
// its only carriers.
func TestReleaseMapEditorNewGroupNumberIsMapWide(t *testing.T) {
	x := installedInspector(t)
	e := openGroupMap(t, x)
	before, err := objectEdits(t, e).model.Map()
	if err != nil {
		t.Fatal(err)
	}
	carriers := func(m *alm.Map, id uint32) map[uint32]int {
		out := map[uint32]int{}
		for _, u := range m.Units {
			if u.GroupID == id {
				out[u.Owner]++
			}
		}
		return out
	}
	const named = 2
	if c := carriers(before, named); len(c) != 1 || c[2] == 0 {
		t.Fatalf("group %d carriers before: %v, want player 2 only", named, c)
	}
	refs := scriptGroupRefs(t, before, named)
	if refs < 6 {
		t.Fatalf("the script names group %d %d times, want at least 6", named, refs)
	}
	a, b := nearUnits(t, before, 3)
	used := map[uint32]bool{}
	for _, u := range before.Units {
		used[u.GroupID] = true
	}
	selectUnits(t, e, a, b)
	if err := e.Key("togroup2"); err != nil {
		t.Fatal(err)
	}
	if e.Dirty() {
		t.Fatal("Ctrl+2 put player 3's units in player 2's group")
	}
	c := &editorClicker{t: t, now: time.Unix(10, 0)}
	r, ok := e.BarRect("groups")
	c.click(e, r, ok, "the Groups entry")
	r, ok = e.GroupsButtonRect(0)
	c.click(e, r, ok, "To group...")
	r, ok = e.PromptButtonRect(0)
	c.click(e, r, ok, "OK on New")
	// Both units are the whole of group 1, which the script names: the
	// window asks first and names the triggers, and nothing is written yet.
	if notes := strings.Join(e.PromptNotes(), " "); e.Dirty() || !strings.Contains(notes, "1") || len(notes) < 20 {
		t.Fatalf("emptying the script's group 1 did not ask first: dirty %v, notes %q", e.Dirty(), notes)
	}
	r, ok = e.PromptButtonRect(0)
	c.click(e, r, ok, "Empty it")
	after, err := objectEdits(t, e).model.Map()
	if err != nil {
		t.Fatal(err)
	}
	g := after.Units[a].GroupID
	if g == 0 || used[g] {
		t.Fatalf("the new group %d is carried by a unit of the map", g)
	}
	if after.Units[b].GroupID != g {
		t.Fatalf("members carry groups %d and %d", g, after.Units[b].GroupID)
	}
	if c := carriers(after, named); len(c) != 1 || c[2] != carriers(before, named)[2] {
		t.Fatalf("group %d carriers after: %v, want player 2's units only", named, c)
	}
	if got := scriptGroupRefs(t, after, named); got != refs {
		t.Fatalf("the script names group %d %d times after, %d before", named, got, refs)
	}
}

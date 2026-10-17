package game

import (
	"fmt"
	"strings"
	"testing"

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

// openGroupWindow opens the Add window from the Groups list.
func openGroupWindow(t *testing.T, x *MapInspector) *ui.MapEditor {
	t.Helper()
	e := openGroupMap(t, x)
	r, ok := e.BarRect("groups")
	(&editorClicker{t: t}).click(e, r, ok, "the Groups entry")
	if err := e.Key("player"); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := e.AddStep(); !ok {
		t.Fatal("no Add window")
	}
	return e
}

// TestReleaseMapEditorAddWindowGroupNumberIsMapWide adds a group of player 3
// on a map whose script names player 2's group 2: the number offered is one
// no unit carries, a typed number another player carries is refused with
// its reason, and the script's group keeps its only carriers.
func TestReleaseMapEditorAddWindowGroupNumberIsMapWide(t *testing.T) {
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
	var members []int
	for i, u := range before.Units {
		if u.Owner == 3 && len(members) < 2 {
			members = append(members, i)
		}
	}
	if len(members) < 2 {
		t.Fatal("player 3 owns fewer than two units")
	}
	used := map[uint32]bool{}
	for _, u := range before.Units {
		used[u.GroupID] = true
	}

	// The refusal of a typed number another player carries.
	e = openGroupWindow(t, x)
	if !addStepsN(t, e, map[string]string{"owner": "3"}, 1) {
		t.Fatal("the window ended before the group step")
	}
	if key, _, _ := e.AddStep(); key != "group" {
		t.Fatalf("step %q, want group", key)
	}
	_ = e.Key("clear")
	e.TextInput(fmt.Sprint(named))
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	key, note, _ := e.AddStep()
	want := x.inspectionUTF8(fmt.Sprintf(x.word("editor.add.group.other"), named, 2))
	if key != "group" || !strings.Contains(note, want) {
		t.Fatalf("typing %d for player 3: step %q note %q, want a refusal containing %q", named, key, note, want)
	}

	// The offered number and the finished add.
	e = openGroupWindow(t, x)
	addSteps(t, e, map[string]string{"owner": "3", "member": fmt.Sprintf("%d|%d", members[0], members[1])})
	after, err := objectEdits(t, e).model.Map()
	if err != nil {
		t.Fatal(err)
	}
	g := after.Units[members[0]].GroupID
	if g == 0 || used[g] {
		t.Fatalf("the offered group %d is carried by a unit of the map", g)
	}
	if after.Units[members[1]].GroupID != g {
		t.Fatalf("members carry groups %d and %d", g, after.Units[members[1]].GroupID)
	}
	if c := carriers(after, named); len(c) != 1 || c[2] != carriers(before, named)[2] {
		t.Fatalf("group %d carriers after: %v, want player 2's units only", named, c)
	}
	if got := scriptGroupRefs(t, after, named); got != refs {
		t.Fatalf("the script names group %d %d times after, %d before", named, got, refs)
	}
}

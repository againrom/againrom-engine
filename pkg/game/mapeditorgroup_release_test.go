package game

import (
	"fmt"
	"image"
	"sort"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// groupMembersOf lists the unit records of one player's group in the document.
func groupMembersOf(d *ui.InspectionDocument, player, group int) []int {
	var out []int
	for i, r := range d.Records {
		if r.Kind == "Unit" && r.Player == player && r.Group == group {
			out = append(out, i)
		}
	}
	return out
}

// groupRowOf is the Groups list row of one player's group.
func groupRowOf(d *ui.InspectionDocument, player, group int) int {
	type pg struct{ p, g int }
	seen := map[pg]bool{}
	var all []pg
	for _, r := range d.Records {
		if r.Kind == "Unit" && r.Group != 0 && !seen[pg{r.Player, r.Group}] {
			seen[pg{r.Player, r.Group}] = true
			all = append(all, pg{r.Player, r.Group})
		}
	}
	sort.SliceStable(all, func(a, b int) bool {
		if all[a].p != all[b].p {
			return all[a].p < all[b].p
		}
		return all[a].g < all[b].g
	})
	for n, k := range all {
		if k.p == player && k.g == group {
			return n
		}
	}
	return -1
}

func clickRect(e *ui.MapEditor, r image.Rectangle, ok bool, now *time.Time) bool {
	if !ok {
		return false
	}
	p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, *now)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, *now)
	*now = now.Add(time.Second)
	return true
}

// TestReleaseMapEditorGroupPopupFollowsUndoAndRedo deletes unit record 0 of
// the installed map scenario/10.alm, opens the group of player 2, group 2, and
// checks that Undo and Redo leave the popup listing, boxing and opening the
// group's true members.
func TestReleaseMapEditorGroupPopupFollowsUndoAndRedo(t *testing.T) {
	x := installedInspector(t)
	now := time.Unix(1, 0)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	if e.Document().Records[0].Kind != "Unit" {
		t.Fatal("record 0 is not a unit")
	}
	e.Select(0, false)
	records := len(e.Document().Records)
	_ = e.Key("delete")
	if len(e.Document().Records) != records-1 {
		t.Fatal("the deletion did not remove record 0")
	}
	row := groupRowOf(e.Document(), 2, 2)
	if row < 0 {
		t.Fatal("player 2 has no group 2 after the deletion")
	}
	r, ok := e.BarRect("groups")
	clickRect(e, r, ok, &now)
	r, ok = e.ListRowRect(row)
	if !clickRect(e, r, ok, &now) {
		t.Fatalf("group row %d is not shown", row)
	}
	check := func(step string) {
		t.Helper()
		want := fmt.Sprint(groupMembersOf(e.Document(), 2, 2))
		if got := fmt.Sprint(e.GroupMembers()); got != want {
			t.Fatalf("%s: the popup lists %s, the group's members are %s", step, got, want)
		}
		for n, id := range e.GroupMembers() {
			if rec := e.Document().Records[id]; rec.Player != 2 || rec.Group != 2 {
				t.Fatalf("%s: member %d is %s #%d of player %d group %d", step, id, rec.Kind, rec.Index, rec.Player, rec.Group)
			}
			if _, ok := e.GroupMemberRect(n); !ok {
				t.Fatalf("%s: member row %d is not shown", step, n)
			}
		}
	}
	check("after the deletion")
	for _, step := range []string{"undo", "redo", "undo"} {
		_ = e.Key(step)
		check("after " + step)
	}
	// The member row opens the group's unit, whichever record it now is.
	members := e.GroupMembers()
	if len(members) == 0 {
		t.Fatal("the group has no member")
	}
	r, ok = e.GroupMemberRect(0)
	if !clickRect(e, r, ok, &now) || e.Selected() != members[0] {
		t.Fatalf("the member row opened record %d, want %d", e.Selected(), members[0])
	}
}

package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestMapEditorSearchOutsideClickUsesDisplayedMap(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	closed := e.mapArea()
	const unit = 0
	cell := e.doc.Records[unit].Cell
	chromeClick(e, e.CellPoint(cell))
	if e.selected != unit {
		t.Fatalf("control click selected record %d, want %d", e.selected, unit)
	}
	e, _, _ = menuEditor(t, "en")
	if err := e.Key(searchKey); err != nil {
		t.Fatal(err)
	}
	open := e.mapArea()
	at := e.CellPoint(cell)
	if open.Min.Y <= closed.Min.Y || !at.In(open) {
		t.Fatalf("search map %v, closed map %v, unit at %v", open, closed, at)
	}
	input := Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}
	e.Pointer(input, time.Unix(1, 0))
	e.Layout(1920, 1080)
	held := e.mapArea()
	input.PrimaryDown = false
	e.Pointer(input, time.Unix(2, 0))
	if e.selected != unit || held != open || e.mapArea() != closed || e.searchTyping() {
		t.Fatalf("click at %v selected record %d with map %v while held and %v on release, want record %d and maps %v then %v",
			at, e.selected, held, e.mapArea(), unit, open, closed)
	}
}

func TestMapEditorSearchLeavesFloatingDockButtonUsable(t *testing.T) {
	for _, x := range []int{40, 840} {
		e, _, _ := menuEditor(t, "en")
		e.openMaps = []*openMap{{doc: e.doc}, {doc: &InspectionDocument{Name: "Second map"}}}
		e.undockToolbar(image.Pt(x, e.mapTabsRect().Max.Y+popupPad))
		before := e.toolbarLayout()
		at := center(before.Close)
		if e.chromeHit(at) != popToolbar || !at.In(e.searchLayout().Rect) {
			t.Fatalf("toolbar close %v does not exercise overlap with search %v", before.Close, e.searchLayout().Rect)
		}
		withoutSearch := e.ChromeImage()
		if err := e.Key(searchKey); err != nil {
			t.Fatal(err)
		}
		duringSearch := e.ChromeImage()
		if got := e.toolbarLayout().Close; got != before.Close {
			t.Fatalf("search moved the floating dock button from %v to %v", before.Close, got)
		}
		chromeClick(e, at)
		if e.toolbarFloats() || e.searchTyping() {
			t.Fatalf("press at %v left toolbar floating %v and search active %v", at, e.toolbarFloats(), e.searchTyping())
		}
		if !sameRectPixels(withoutSearch, before.Close, duringSearch, before.Close) {
			t.Fatalf("search covered the dock button at %v", before.Close)
		}
	}
}

func TestMapEditorSearchTipWheelReachesQueryTail(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	e.SetBodyFont(wordsFont("en"))
	if err := e.Key(searchKey); err != nil {
		t.Fatal(err)
	}
	query := strings.Repeat("southern bridge guards ", 150) + "query tail"
	e.TextInput(query)
	at := center(e.searchEntry().rect)
	input := Input{CursorX: at.X, CursorY: at.Y}
	now := time.Unix(10, 0)
	e.Pointer(input, now)
	now = now.Add(time.Duration(editorTipDelay) * time.Millisecond)
	e.Pointer(input, now)
	first, ok := e.shownTip()
	if !ok || first.Text != query || first.Rows <= 0 || first.Rows >= len(first.Lines) {
		t.Fatalf("expected a settled, paged tip for %d query bytes, got %d bytes and %d of %d lines",
			len(query), len(first.Text), first.Rows, len(first.Lines))
	}
	input.WheelY = -1
	e.Pointer(input, now.Add(time.Millisecond))
	step, _ := e.shownTip()
	if step.Top <= first.Top {
		t.Fatalf("the wheel left the first visible line at %d", step.Top)
	}
	input.WheelY = -float64(len(first.Lines))
	e.Pointer(input, now.Add(2*time.Millisecond))
	last, ok := e.shownTip()
	lines := last.shown()
	if !ok || last.Top != len(last.Lines)-last.Rows || !strings.HasSuffix(strings.Join(lines, " "), "query tail") {
		t.Fatalf("the end of the tip shows from line %d with %d of %d lines", last.Top, last.Rows, len(last.Lines))
	}
}

// searchView is the map view's place and zoom.
func searchView(e *MapEditor) [3]float64 {
	c := e.doc.Viewer.Camera()
	return [3]float64{c.X, c.Y, c.Zoom}
}

// searchLabels are the open search list's rows.
func searchLabels(e *MapEditor) []string {
	var out []string
	for _, it := range e.menuItems("search") {
		out = append(out, it.label)
	}
	return out
}

// searchText is the text the search field holds while it takes the typed
// text, and whether it does.
func searchText(e *MapEditor) (string, bool) {
	target, _, key := e.inputTarget()
	if key != "search" || target == nil {
		return "", false
	}
	return *target, true
}

// treeHitLabels are the members the tree's search keeps for query, each as
// the search list names it: its category's word, then its tree row's text.
func treeHitLabels(e *MapEditor, query string) []string {
	kept := e.tree.search
	defer func() { e.tree.search = kept; e.treeSpec() }()
	e.tree.search = query
	s, ok := e.treeSpec()
	if !ok {
		return nil
	}
	var out []string
	for i, r := range s.List.Rows() {
		row := e.tree.rows[i]
		if !row.member {
			continue
		}
		c, _ := categoryByKey(row.of)
		out = append(out, e.word(c.word)+": "+r.Text)
	}
	return out
}

// Owner item 6: Ctrl+F opens the search field below File; typed text goes to it and no key it
// binds acts; Esc leaves it and empties it.
func TestMapEditorBarSearchTakesCtrlF(t *testing.T) {
	for _, c := range []struct{ lang, hint string }{{"en", "Search (Ctrl+F)"}, {"ru", "Поиск (Ctrl+F)"}} {
		t.Run(c.lang, func(t *testing.T) {
			e, _, _ := menuEditor(t, c.lang)
			// Owner item 6: the field opens below File, not in the bar.
			if _, ok := e.FindControl(ControlQuery{Role: ControlBar, Key: "search"}); ok {
				t.Fatal("Search remains in the bar")
			}
			box := e.searchEntry().rect
			if row := e.searchRow(); row.Hint != c.hint || row.Kind != popupEdit {
				t.Fatalf("the field's row %+v", row)
			}
			cam := searchView(e)
			shown, records := e.panelsShown(), len(e.doc.Records)
			updateEditorKeys(t, e, ebiten.KeyControl, ebiten.KeyF)
			if _, ok := searchText(e); !ok {
				t.Fatal("Ctrl+F did not focus the search field")
			}
			if searchView(e) != cam {
				t.Fatal("Ctrl+F fitted the map as F does")
			}
			if !e.searchRow().Focus {
				t.Fatal("the focused field does not show its caret")
			}
			e.TextInput("unit")
			updateEditorKeys(t, e, ebiten.KeyU)
			if got, _ := searchText(e); got != "unit" || len(e.doc.Records) != records || e.panelsShown() != shown || e.tree.search != "" {
				t.Fatalf("typed %q, records %d, panels %v, tree search %q", got, len(e.doc.Records), e.panelsShown(), e.tree.search)
			}
			if e.menu != "search" {
				t.Fatalf("the hit list is not open: menu %q", e.menu)
			}
			_ = e.Key("escape")
			if _, ok := searchText(e); ok || e.menu != "" || e.searchRow().Value != "" {
				t.Fatalf("Esc left the field typing, menu %q", e.menu)
			}
			// Owner item 6: File opens the same field after Esc.
			openMenuRow(t, e, "file", "search")
			chromeClick(e, center(box))
			if _, ok := searchText(e); !ok {
				t.Fatal("a press on the field did not focus it")
			}
		})
	}
}

// One query lists the members of several categories, each named by its
// category's word.
func TestMapEditorBarSearchListsEveryCategory(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, _ := menuEditor(t, lang)
			_ = e.Key("search")
			e.TextInput("1")
			seen := map[string]bool{}
			for _, label := range searchLabels(e) {
				for _, c := range editorCategories {
					if strings.HasPrefix(label, e.word(c.word)+": ") {
						seen[c.key] = true
					}
				}
			}
			if len(seen) < 3 || !seen["players"] || !seen["units"] || !seen["scripts"] {
				t.Fatalf("categories %v in\n%s", seen, strings.Join(searchLabels(e), "\n"))
			}
		})
	}
}

// The search keeps what the tree's search keeps for the same text, in the
// tree's order.
func TestMapEditorBarSearchMatchesTheTreeSearch(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	_ = e.Key("search")
	for _, q := range []string{"UNIT3", "#1", "hide", "trigger 1", "group 2", "hall", "enemy", "p2", "  unit ", "nothing", "1"} {
		e.searchLeave(true)
		_ = e.Key("search")
		e.TextInput(q)
		got, want := strings.Join(searchLabels(e), "\n"), strings.Join(treeHitLabels(e, q), "\n")
		if got != want {
			t.Errorf("query %q lists\n%s\nthe tree keeps\n%s", q, got, want)
		}
		if want == "" {
			b, _ := e.openMenuEntry()
			if e.menuDropdown(b, true).Open {
				t.Errorf("query %q opened an empty list", q)
			}
		}
	}
	if got := strings.Join(searchLabels(e), "\n"); !strings.Contains(got, "Units: Unit #1  unit1  P1  12,10") {
		t.Fatalf("query 1 lists\n%s", got)
	}
}

// Choosing a hit selects it as a press on its tree row does: the same record,
// the same record window and view; the tree marks it and no side panel
// changes.
func TestMapEditorBarSearchChoiceIsATreePress(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, _ := menuEditor(t, lang)
			twin, _, _ := menuEditor(t, lang)
			for _, m := range []*MapEditor{e, twin} {
				c := m.doc.Viewer.Camera()
				c.SetZoom(2)
				c.X, c.Y = 0, 0
				c.Clamp()
			}
			hall := 5
			if p := e.CellPoint(image.Pt(30, 30)); p.In(e.mapArea()) {
				t.Fatalf("the hall shows at %v before the choice", p)
			}
			shown, closed := e.panelsShown(), fmt.Sprint(e.tree.closed)
			_ = e.Key("search")
			e.TextInput("hall")
			labels := searchLabels(e)
			if len(labels) != 1 {
				t.Fatalf("hall lists %q", labels)
			}
			chromeClick(e, menuPoint(t, e, 0))
			clickTreeRow(t, twin, treeRowOf(twin, hall))
			if e.selected != hall || twin.selected != hall {
				t.Fatalf("the hit selects %d, the tree row %d", e.selected, twin.selected)
			}
			if p := e.CellPoint(image.Pt(30, 30)); !p.In(e.mapArea()) || searchView(e) != searchView(twin) {
				t.Fatalf("the hall shows at %v, outside the map %v or off the tree press's view", p, e.mapArea())
			}
			a, _, okA := e.objectSpec()
			b, _, okB := twin.objectSpec()
			if !okA || !okB || a.Docked || rowsText(a) != rowsText(b) {
				t.Fatalf("record windows differ\n%s\n%s", rowsText(a), rowsText(b))
			}
			if e.panelsShown() != shown || fmt.Sprint(e.tree.closed) != closed || e.tree.search != "" {
				t.Fatalf("panels %v were %v, tree closed %v was %s, tree search %q", e.panelsShown(), shown, e.tree.closed, closed, e.tree.search)
			}
			if !strings.Contains(treeText(e), ">  "+strings.TrimPrefix(labels[0], e.word("editor.bar.structures")+": ")) {
				t.Fatalf("the tree does not mark the hall\n%s", treeText(e))
			}
			if _, ok := searchText(e); ok || e.menu != "" {
				t.Fatal("the field still takes the typed text after the choice")
			}
			// Down moves the highlighted hit and Enter chooses it.
			// Ctrl+F again keeps the text, selected: typing replaces it.
			_ = e.Key("search")
			if got, _ := searchText(e); got != "hall" || e.menu != "search" {
				t.Fatalf("Ctrl+F after the choice holds %q, menu %q", got, e.menu)
			}
			e.TextInput("unit")
			n := -1
			for i, label := range searchLabels(e) {
				if strings.Contains(label, "unit2") {
					n = i
				}
			}
			if n < 1 {
				t.Fatalf("unit lists %q", searchLabels(e))
			}
			for range n {
				_ = e.Key("down")
			}
			_ = e.Key("enter")
			if e.selected != 2 || !e.CellPoint(image.Pt(14, 10)).In(e.mapArea()) {
				t.Fatalf("Enter on hit %d selects %d", n, e.selected)
			}
			// A group hit chooses the group as its tree row does.
			_ = e.Key("search")
			e.TextInput("1")
			n = -1
			for i, label := range searchLabels(e) {
				if n < 0 && strings.HasPrefix(label, e.word("editor.bar.groups")+": ") {
					n = i
				}
			}
			if n < 0 {
				t.Fatalf("1 lists no group: %q", searchLabels(e))
			}
			chromeClick(e, menuPoint(t, e, n))
			byKey, _, _ := menuEditor(t, lang)
			_ = byKey.Key("group1")
			if !e.groupSel.set || fmt.Sprint(e.groupSel) != fmt.Sprint(byKey.groupSel) {
				t.Fatalf("the group hit chose %+v, the key %+v", e.groupSel, byKey.groupSel)
			}
		})
	}
}

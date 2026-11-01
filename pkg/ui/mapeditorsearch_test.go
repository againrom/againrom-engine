package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

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

// Ctrl+F puts the typing in the bar's search field, after the category
// entries, its hint the search word; typed text goes to it and no key it
// binds acts; Esc leaves it and empties it.
func TestMapEditorBarSearchTakesCtrlF(t *testing.T) {
	for _, c := range []struct{ lang, hint string }{{"en", "Search (Ctrl+F)"}, {"ru", "Поиск (Ctrl+F)"}} {
		t.Run(c.lang, func(t *testing.T) {
			e, _, _ := menuEditor(t, c.lang)
			var keys []string
			for _, b := range e.barEntries() {
				keys = append(keys, b.key)
			}
			if got := strings.Join(keys, " "); !strings.Contains(got, "structures search objects") {
				t.Fatalf("bar entries %s", got)
			}
			box, _ := e.BarRect("search")
			if box.Dx() < 200 || box.Max.X > e.window.X {
				t.Fatalf("the search field stands at %v in a window %v", box, e.window)
			}
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
			// A press on the field focuses it as Ctrl+F does.
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
		if want == "" && e.menu == "search" {
			t.Errorf("query %q opened an empty list", q)
		}
	}
	if got := strings.Join(searchLabels(e), "\n"); !strings.Contains(got, "Units: Unit #1  unit1  P1  12,10") {
		t.Fatalf("query 1 lists\n%s", got)
	}
}

// Choosing a hit selects it as a press on its tree row does: the same record,
// the same right panel, the same view; the tree marks it and no side panel
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
			if !okA || !okB || !e.panels.hosted || rowsText(a) != rowsText(b) {
				t.Fatalf("right panels differ\n%s\n%s", rowsText(a), rowsText(b))
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

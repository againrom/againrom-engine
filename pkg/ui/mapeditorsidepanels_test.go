package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/words"
)

// panelsEditor opens the groups map with its two players as records, the
// side panels at their defaults for a fresh profile, in a window of w by h.
// Records 0..4 are units, 5 the structure, 6 the action, 7 the trigger and
// 8, 9 the players. saved collects what the panels write to the profile.
func panelsEditor(t *testing.T, w, h int, lang string) (e *MapEditor, p *groupsProbe, saved *[][]byte) {
	t.Helper()
	p = &groupsProbe{units: []groupsUnit{{1, 1, image.Pt(10, 10)}, {1, 1, image.Pt(12, 10)}, {1, 0, image.Pt(14, 10)}, {2, 2, image.Pt(16, 10)}, {2, 0, image.Pt(18, 10)}}}
	p.calls, p.self, p.players = &modesCalls{}, p, []string{"Self", "Enemy"}
	build := func(source string) *InspectionDocument {
		v, err := NewViewer("panels", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		d := &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte("map"), Edits: p}
		members := map[int]int{}
		for i, u := range p.units {
			d.Records = append(d.Records, InspectionRecord{Kind: "Unit", Index: i, Section: 6, Cell: u.cell, Spatial: true, Size: image.Pt(1, 1), Label: fmt.Sprintf("unit%d", i), Player: u.player, Group: u.group})
			members[u.group]++
		}
		d.Records = append(d.Records,
			InspectionRecord{Kind: "Structure", Index: 0, Section: 4, Cell: image.Pt(30, 30), Spatial: true, Size: image.Pt(2, 2), Label: "hall"},
			InspectionRecord{Kind: "Action", Index: 0, Section: 7},
			InspectionRecord{Kind: "Trigger", Index: 0, Section: 7})
		for i, name := range p.players {
			d.Records = append(d.Records, InspectionRecord{Kind: "Player", Index: i, Section: 5, Label: name, Details: []string{"Name: " + name}})
		}
		var targets []InspectionScriptTarget
		for g := 1; g <= 3; g++ {
			target := InspectionScriptTarget{Value: uint32(g), Record: -1, Members: members[g]}
			if g == 1 {
				target.Uses = []InspectionScriptUse{{Node: 6, Slot: 0, Triggers: []int{7}}}
			}
			targets = append(targets, target)
		}
		d.Script = &InspectionScriptModel{
			Actions:  []InspectionScriptNode{{Record: 6, ID: 1, Opcode: 32, Label: "off map", Types: [10]uint32{scriptGroup}, Values: [10]uint32{1}}},
			Triggers: []InspectionTrigger{{Record: 7, Name: "Hide them", Acts: [4]uint32{1}}},
			Targets:  map[uint32][]InspectionScriptTarget{scriptGroup: targets},
		}
		return d
	}
	p.doc = func(InspectionEdits) *InspectionDocument { return build("edited") }
	e = NewMapEditor(panelFont(), []EditorMap{{Key: "map0", Label: "Map 0"}, {Key: "map1", Label: "Map 1"}}, func(source string) (*InspectionDocument, error) { return build(source), nil })
	if lang != "" {
		e.SetWords(words.For(lang))
	}
	e.Layout(w, h)
	saved = &[][]byte{}
	e.SetPanelProfile(nil, os.ErrNotExist, func(b []byte) error {
		*saved = append(*saved, append([]byte(nil), b...))
		return nil
	})
	if err := e.Open("map0"); err != nil {
		t.Fatal(err)
	}
	e.Layout(w, h)
	return e, p, saved
}

// treeText is the left panel's tree as an author reads it: one line per row,
// the selected row marked, indent and fold mark before the text.
func treeText(e *MapEditor) string {
	s, ok := e.treeSpec()
	if !ok {
		return ""
	}
	var b strings.Builder
	for i, r := range s.List.Rows() {
		mark := " "
		if i == s.List.Selection() && i < len(e.tree.rows) && e.treeChosen(e.tree.rows[i]) {
			mark = ">"
		}
		fold := ""
		if r.Fold != FoldNone {
			fold = rowFoldMarks[r.Fold] + " "
		}
		fmt.Fprintf(&b, "%s%s%s%s\n", mark, strings.Repeat("  ", r.Indent), fold, r.Text)
	}
	return b.String()
}

// treeRowOf is the tree row that stands for record, -1 for none.
func treeRowOf(e *MapEditor, record int) int {
	e.treeSpec()
	for i, r := range e.tree.rows {
		if r.member && !r.group.set && r.record == record {
			return i
		}
	}
	return -1
}

// clickTreeRow clicks tree row n where the left panel shows it.
func clickTreeRow(t *testing.T, e *MapEditor, n int) {
	t.Helper()
	s, ok := e.treeSpec()
	if !ok {
		t.Fatal("the left panel is closed")
	}
	top, count := s.List.Visible()
	if n < top || n >= top+count {
		t.Fatalf("tree row %d is not shown (%d..%d)", n, top, top+count)
	}
	chromeClick(e, center(layoutPopup(s, e.font, e.bodyFont()).List.Row(n-top)))
}

// rowsText is a popup's rows as text: kind, label and value of each.
func rowsText(s popupSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "title %q\n", s.Title)
	for _, r := range s.Rows {
		fmt.Fprintf(&b, "%d %q %q %v %q\n", r.Kind, r.Label, r.Value, r.Buttons, r.Hint)
	}
	return b.String()
}

// minimizedPanels is a profile file that minimizes both panels.
var minimizedPanels = []byte(`{"left":{"minimized":true,"width":260},"right":{"minimized":true,"width":520}}`)

// chromeRectsLine is one line of the chrome rectangles table, split into its
// fields.
type chromeRectsLine struct {
	name, status, bar string
	area              image.Rectangle
	frame             image.Point
	popups            []image.Rectangle
}

var (
	chromeRectsField = regexp.MustCompile(`^(\S+ \S+) map=\((\d+),(\d+)\)-\((\d+),(\d+)\) status=(\S+) bar=(\[[^\]]*\]) popups=\[([^\]]*)\] cell=\S+ frame=(\d+)x(\d+) `)
	chromeRectsRect  = regexp.MustCompile(`\((-?\d+),(-?\d+)\)-\((-?\d+),(-?\d+)\)`)
)

// chromeRectsLines splits the chrome rectangles table into its lines.
func chromeRectsLines(t *testing.T, text string) []chromeRectsLine {
	t.Helper()
	num := func(s string) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	var out []chromeRectsLine
	for _, ln := range strings.Split(strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n")), "\n") {
		m := chromeRectsField.FindStringSubmatch(ln)
		if m == nil {
			t.Fatalf("chrome line %q does not read", ln)
		}
		l := chromeRectsLine{name: m[1], status: m[6], bar: m[7], area: image.Rect(num(m[2]), num(m[3]), num(m[4]), num(m[5])), frame: image.Pt(num(m[9]), num(m[10]))}
		for _, r := range chromeRectsRect.FindAllStringSubmatch(m[8], -1) {
			l.popups = append(l.popups, image.Rect(num(r[1]), num(r[2]), num(r[3]), num(r[4])))
		}
		out = append(out, l)
	}
	return out
}

// (a) With both panels minimized, the map area is the area the editor had
// before the panels existed less the two strips; the status strip and the
// bar stand where they stood, the map frame fills the map area, and every
// window that is not a record category's palette keeps its size. The
// recorded table was read from the editor without panels.
func TestMapEditorMinimizedPanelsKeepTheChromeRects(t *testing.T) {
	raw, err := os.ReadFile("testdata/mapeditor_chrome_rects.golden")
	if err != nil {
		t.Fatal(err)
	}
	probe, _ := groupsEditor(t)
	strip := probe.panelStrip()
	base := chromeRectsLines(t, string(raw))
	got := chromeRectsLines(t, chromeRectsText(t, func(e *MapEditor) { e.SetPanelProfile(minimizedPanels, nil, nil) }))
	if len(got) != len(base) {
		t.Fatalf("%d chrome lines, the base has %d", len(got), len(base))
	}
	for i, g := range got {
		b := base[i]
		players := strings.HasSuffix(g.name, " players") || strings.HasSuffix(g.name, " players-own")
		want := image.Rect(b.area.Min.X+strip, b.area.Min.Y, b.area.Max.X-strip, b.area.Max.Y)
		if players && b.area.Dx()-strip-sidePanels[panelLeft].width >= mapLeastWidth {
			// The Players entry restores the left panel to show its branch.
			want.Min.X = sidePanels[panelLeft].width
		}
		if g.name != b.name || g.area != want || g.status != b.status || g.bar != b.bar || g.frame != want.Size() {
			t.Errorf("%s: map %v frame %v status %s, want map %v status %s", g.name, g.area, g.frame, g.status, want, b.status)
		}
		// The windows less the two strips keep their sizes; the players
		// palette lost its record rows.
		var windows []image.Rectangle
		for _, r := range g.popups {
			if r.Dx() != strip || r.Min.Y != g.area.Min.Y || r.Max.Y != g.area.Max.Y {
				windows = append(windows, r)
			}
		}
		if players {
			continue
		}
		if len(windows) != len(b.popups) {
			t.Errorf("%s: windows %v, the base %v", g.name, windows, b.popups)
			continue
		}
		for n := range windows {
			if windows[n].Size() != b.popups[n].Size() {
				t.Errorf("%s: window %d is %v, the base %v", g.name, n, windows[n], b.popups[n])
			}
		}
	}
}

// (b) A fresh profile restores both panels where they fit beside the map's
// least width, else the left one, else none; a minimized panel shows its
// strip. The panels take their widths off the map, which keeps its least
// width when the window or a panel's edge moves.
func TestMapEditorPanelsTakeTheirWidthsOffTheMap(t *testing.T) {
	probe, _, _ := panelsEditor(t, 1280, 800, "")
	s := probe.panelStrip()
	for _, c := range []struct {
		w, h        int
		left, right int
	}{{1280, 800, 260, 520}, {1024, 768, 260, s}, {640, 480, s, s}} {
		e, _, _ := panelsEditor(t, c.w, c.h, "")
		if got := e.panelsShown(); got != [2]int{c.left, c.right} {
			t.Errorf("%dx%d shows panels %v, want %d and %d", c.w, c.h, got, c.left, c.right)
		}
		area := e.mapArea()
		if area.Min.X != c.left || area.Max.X != c.w-c.right || area.Dx() < mapLeastWidth {
			t.Errorf("%dx%d map area %v", c.w, c.h, area)
		}
		if v := e.doc.Viewer; v.frameW != area.Dx() || v.frameH != area.Dy() {
			t.Errorf("%dx%d map frame %dx%d for area %v", c.w, c.h, v.frameW, v.frameH, area)
		}
		if l, r := e.sidePanelRect(panelLeft), e.sidePanelRect(panelRight); c.left > 0 && (l.Max.X != area.Min.X || l.Min.Y != area.Min.Y || l.Max.Y != area.Max.Y) || c.right > 0 && r.Min.X != area.Max.X {
			t.Errorf("%dx%d panels %v %v beside map %v", c.w, c.h, l, r, area)
		}
		// A cell still shows where the map frame puts it, right of the left panel.
		if p := e.CellPoint(image.Pt(10, 10)); !p.In(area) && e.doc.Viewer.cam.ViewW > 0 {
			e.Fit()
			if p = e.CellPoint(image.Pt(10, 10)); !p.In(area) {
				t.Errorf("%dx%d cell 10,10 shows at %v, outside the map %v", c.w, c.h, p, area)
			}
		}
	}
	// A narrower window minimizes the right panel first, then the left, and
	// a wider one restores them again.
	e, _, _ := panelsEditor(t, 1280, 800, "")
	for _, c := range []struct {
		w           int
		left, right int
	}{{1260, 260, 520}, {1259, 260, s}, {740 + s, 260, s}, {739 + s, s, s}, {1280, 260, 520}} {
		e.Layout(c.w, 800)
		if got := e.panelsShown(); got != [2]int{c.left, c.right} || e.mapArea().Dx() < mapLeastWidth {
			t.Errorf("window %d shows panels %v, map %v", c.w, got, e.mapArea())
		}
	}
	// A dragged edge stops at the panel's greatest width and at the map's
	// least width, and the new width is kept.
	e, _, saved := panelsEditor(t, 1280, 800, "")
	dragPanelEdge(t, e, popTree, 10)
	if got := e.panelsShown()[panelLeft]; got != 270 {
		t.Fatalf("the left edge dragged by 10 gives %d", got)
	}
	dragPanelEdge(t, e, popTree, 400)
	if got := e.panelsShown(); got[panelLeft] != 1280-mapLeastWidth-520 || e.mapArea().Dx() != mapLeastWidth {
		t.Fatalf("the left edge dragged far gives %v, map %v", got, e.mapArea())
	}
	dragPanelEdge(t, e, popTree, -900)
	if got := e.panelsShown()[panelLeft]; got != sidePanels[panelLeft].least {
		t.Fatalf("the left edge dragged back gives %d", got)
	}
	dragPanelEdge(t, e, popProps, -100)
	if got := e.panelsShown()[panelRight]; got != 1280-mapLeastWidth-sidePanels[panelLeft].least {
		t.Fatalf("the right edge dragged left gives %d", got)
	}
	if len(*saved) == 0 || !bytes.Contains((*saved)[len(*saved)-1], []byte(`"left":{"minimized":false,"width":180}`)) {
		t.Fatalf("the dragged widths were not kept: %q", *saved)
	}
	// F6 and F7 minimize and restore the panels, and each change is kept.
	before := len(*saved)
	for _, k := range []string{"objectspanel", "propertiespanel"} {
		if err := e.Key(k); err != nil {
			t.Fatal(err)
		}
	}
	if got := e.panelsShown(); got != [2]int{s, s} || len(*saved) != before+2 || e.mapArea().Min.X != s || e.mapArea().Max.X != 1280-s {
		t.Fatalf("F6 and F7 left panels %v, %d saves, map %v", got, len(*saved)-before, e.mapArea())
	}
	_ = e.Key("objectspanel")
	if got := e.panelsShown(); got[panelLeft] != sidePanels[panelLeft].least {
		t.Fatalf("F6 restored the left panel at %d", got[panelLeft])
	}
	// The View menu's two entries minimize and restore the panels, marked
	// while restored.
	e, _, _ = panelsEditor(t, 1280, 800, "")
	for i, key := range []string{"objectspanel", "propertiespanel"} {
		n, ok := e.MenuRow("view", key)
		if !ok {
			t.Fatalf("the View menu has no %s entry", key)
		}
		if text := e.menuItems("view")[n].label; !strings.Contains(text, e.word(sidePanels[i].titleWord)) {
			t.Fatalf("View row %d reads %q", n, text)
		}
		modesClick(e, barPoint(t, e, "view"))
		r, ok := e.MenuRowRect(n)
		if !ok {
			t.Fatal("the View menu did not open")
		}
		modesClick(e, center(r))
		if e.panelOpen(i) || e.panelsShown()[i] != s {
			t.Fatalf("the View entry %s left panel %d restored", key, i)
		}
	}
	// A panel the window has no room for stays minimized and says so.
	e, _, _ = panelsEditor(t, 1024, 768, "")
	_ = e.Key("propertiespanel")
	if e.panelOpen(panelRight) || e.message != e.word("editor.side.narrow") {
		t.Fatalf("F7 at 1024 shows %v, message %q", e.panelsShown(), e.message)
	}
}

// dragPanelEdge drags the inner edge of the panel popup id stands in by dx
// with the pointer.
func dragPanelEdge(t *testing.T, e *MapEditor, id editorPopupID, dx int) {
	t.Helper()
	i, _ := e.panelOf(id)
	r := e.sidePanelRect(i)
	x := r.Max.X - 1
	if sidePanels[i].place == placeRight {
		x = r.Min.X
	}
	at := image.Point{}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if e.panelEdgeAt(id, image.Pt(x, y)) != 0 {
			at = image.Pt(x, y)
			break
		}
	}
	if at == (image.Point{}) {
		t.Fatalf("no point of the inner edge of panel %d grabs", i)
	}
	modesPress(e, at, true)
	modesPress(e, at.Add(image.Pt(dx, 0)), true)
	modesPress(e, at.Add(image.Pt(dx, 0)), false)
}

// (c) The tree holds one branch per kind with its count, and one row per
// player, trigger, unit, structure and group under it.
func TestMapEditorTreeRowsFollowTheMap(t *testing.T) {
	e, _, _ := panelsEditor(t, 1280, 800, "")
	want := ` - Scripts (1)
   Triggers
     Trigger 1  Hide them
   Variables: none
   Conditions: none
   Actions: none
 - Players (2)
   Player #0  Self
   Player #1  Enemy
 - Groups (2)
   Group 1  P1  2 units
   Group 2  P2  1 units
 - Units (5)
   Unit #0  unit0  P1  10,10
   Unit #1  unit1  P1  12,10
   Unit #2  unit2  P1  14,10
   Unit #3  unit3  P2  16,10
   Unit #4  unit4  P2  18,10
 - Sacks (0)
 - Structures (1)
   Structure #0  hall  30,30
`
	got := treeText(e)
	if got != want {
		t.Fatalf("tree\n%s\nwant\n%s", got, want)
	}
	// Each member row reads as the editor already words the record.
	for i, n := range []int{8, 7, 0, 5} {
		row := treeRowOf(e, n)
		if row < 0 {
			t.Fatalf("record %d has no tree row", n)
		}
		text := e.tree.model.Rows()[row].Text
		if i != 1 && text != e.recordSummary(n) || i == 1 && text != e.scriptRowText(n, 0) {
			t.Errorf("record %d reads %q", n, text)
		}
	}
	// A branch with no member stays, with count 0.
	e, p, _ := panelsEditor(t, 1280, 800, "")
	p.units, p.dirty = nil, true
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if got := treeText(e); !strings.Contains(got, " - Units (0)\n - Sacks (0)\n - Structures (1)") || !strings.Contains(got, " - Groups (0)\n") {
		t.Fatalf("empty branches:\n%s", got)
	}
	// A branch row closes and opens its branch.
	e, _, _ = panelsEditor(t, 1280, 800, "")
	clickTreeRow(t, e, 0)
	if got := treeText(e); !strings.Contains(got, " + Scripts (1)\n - Players (2)") {
		t.Fatalf("a press on the Scripts branch:\n%s", got)
	}
	clickTreeRow(t, e, 0)
	if got := treeText(e); !strings.Contains(got, " - Scripts (1)\n   Triggers\n") {
		t.Fatalf("a second press on the Scripts branch:\n%s", got)
	}
}

// (d) The search keeps the members whose name or ID holds the text, in any
// case, under their open branches; other branches leave the tree.
func TestMapEditorTreeSearchKeepsMatchingRows(t *testing.T) {
	e, _, _ := panelsEditor(t, 1280, 800, "")
	clickTreeRow(t, e, 0) // close Scripts: a search shows it open
	for _, c := range []struct{ search, want string }{
		{"UNIT3", " - Units (1)\n   Unit #3  unit3  P2  16,10\n"},
		{"#1", " - Players (1)\n   Player #1  Enemy\n - Units (1)\n   Unit #1  unit1  P1  12,10\n"},
		{"hide", " - Scripts (1)\n   Triggers\n     Trigger 1  Hide them\n"},
		{"trigger 1", " - Scripts (1)\n   Triggers\n     Trigger 1  Hide them\n"},
		{"group 2", " - Groups (1)\n   Group 2  P2  1 units\n"},
		{"nothing", ""},
	} {
		e.tree.search = c.search
		if got := treeText(e); got != c.want {
			t.Errorf("search %q:\n%s\nwant\n%s", c.search, got, c.want)
		}
	}
	// The search field takes typed text after a click on it, and Escape
	// clears it.
	e.tree.search = ""
	chromeClick(e, center(e.treeSearchBox()))
	if !e.treeTyping() {
		t.Fatal("a click on the search field did not focus it")
	}
	e.TextInput("Hall")
	if e.tree.search != "Hall" || treeText(e) != " - Structures (1)\n   Structure #0  hall  30,30\n" {
		t.Fatalf("typed search %q:\n%s", e.tree.search, treeText(e))
	}
	// A key the editor binds is text while the field takes it.
	before := e.panelsShown()
	e.TextInput("u")
	if e.tree.search != "Hallu" || e.panelsShown() != before {
		t.Fatalf("typed u gives %q", e.tree.search)
	}
	_ = e.Key("escape")
	if e.tree.search != "" || e.treeTyping() {
		t.Fatalf("Escape left the search %q, typing %v", e.tree.search, e.treeTyping())
	}
}

// (e) A tree choice selects as a map click does: the same record, the same
// right panel and the same marked tree row; a unit comes into view.
func TestMapEditorTreeChoiceIsAMapClick(t *testing.T) {
	byTree, _, _ := panelsEditor(t, 1280, 800, "")
	byMap, _, _ := panelsEditor(t, 1280, 800, "")
	clickTreeRow(t, byTree, treeRowOf(byTree, 0))
	byMap.Fit()
	modesClick(byMap, byMap.CellPoint(image.Pt(10, 10)))
	if byTree.selected != 0 || byMap.selected != 0 {
		t.Fatalf("tree selects %d, map click %d", byTree.selected, byMap.selected)
	}
	a, _, okA := byTree.objectSpec()
	b, _, okB := byMap.objectSpec()
	if !okA || !okB || !byTree.panels.hosted || !byMap.panels.hosted || rowsText(a) != rowsText(b) {
		t.Fatalf("right panels differ\n%s\n%s", rowsText(a), rowsText(b))
	}
	if treeText(byTree) != treeText(byMap) || !strings.Contains(treeText(byMap), ">  Unit #0  unit0  P1  10,10\n") {
		t.Fatalf("marked rows differ\n%s\n%s", treeText(byTree), treeText(byMap))
	}
	if p := byTree.CellPoint(image.Pt(10, 10)); !p.In(byTree.mapArea()) {
		t.Fatalf("the chosen unit shows at %v, outside the map %v", p, byTree.mapArea())
	}
	// A trigger and a player row select for the right panel.
	for _, n := range []int{7, 8} {
		clickTreeRow(t, byTree, treeRowOf(byTree, n))
		byMap.Select(n, false)
		a, _, okA = byTree.objectSpec()
		b, _, okB = byMap.objectSpec()
		if byTree.selected != n || !okA || !okB || !byTree.panels.hosted || rowsText(a) != rowsText(b) {
			t.Fatalf("record %d: tree selects %d, panels\n%s\n%s", n, byTree.selected, rowsText(a), rowsText(b))
		}
	}
	// A group row chooses the group as its number key does.
	byKey, _, _ := panelsEditor(t, 1280, 800, "")
	_ = byKey.Key("group1")
	e, _, _ := panelsEditor(t, 1280, 800, "")
	e.treeSpec()
	clickTreeRow(t, e, e.TreeMemberRow("groups", 0))
	if !e.groupSel.set || fmt.Sprint(e.groupSel) != fmt.Sprint(byKey.groupSel) || refsText(e.selectionRefs()) != refsText(byKey.selectionRefs()) {
		t.Fatalf("the group row chose %+v, the key %+v", e.groupSel, byKey.groupSel)
	}
}

// (f) After an add, a delete, an undo and a load, the tree is the tree a
// fresh editor builds for the same map.
func TestMapEditorTreeFollowsEdits(t *testing.T) {
	e, p, _ := panelsEditor(t, 1280, 800, "")
	fresh := func(step string) {
		t.Helper()
		doc := e.doc
		f := NewMapEditor(panelFont(), []EditorMap{{Key: "same"}}, func(string) (*InspectionDocument, error) { return doc, nil })
		f.Layout(1280, 800)
		f.SetPanelProfile(nil, os.ErrNotExist, nil)
		if err := f.Open("same"); err != nil {
			t.Fatal(err)
		}
		if e.groupSel.set {
			f.chooseGroup(e.groupSel, false)
		} else {
			f.Select(e.selected, false)
		}
		if got, want := treeText(e), treeText(f); got != want {
			t.Fatalf("after %s the tree is\n%s\na fresh tree\n%s", step, got, want)
		}
	}
	treeText(e)
	for _, key := range []string{"players", "player", "enter", "enter", "enter"} {
		if err := e.Key(key); err != nil {
			t.Fatal(err)
		}
	}
	if len(p.players) != 3 || !strings.Contains(treeText(e), " - Players (3)\n") {
		t.Fatalf("add: %v\n%s", p.players, treeText(e))
	}
	fresh("add")
	p.units = p.units[:4]
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(treeText(e), " - Players (2)\n") || !strings.Contains(treeText(e), " - Units (4)\n") {
		t.Fatalf("delete:\n%s", treeText(e))
	}
	fresh("delete")
	p.players = append(p.players, "Back")
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	fresh("undo")
	if err := e.Key("redo"); err != nil {
		t.Fatal(err)
	}
	fresh("redo")
	p.dirty = false
	if err := e.Open("map1"); err != nil {
		t.Fatal(err)
	}
	fresh("load")
}

// (g) The right panel shows the window a floating one shows: the same rows,
// words and values for an object, a trigger and a player.
func TestMapEditorRightPanelHostsTheFloatingWindow(t *testing.T) {
	for _, n := range []int{0, 5, 7, 8} {
		e, _, _ := panelsEditor(t, 1280, 800, "")
		e.Select(n, false)
		hosted, _, ok := e.objectSpec()
		if !ok || !e.panels.hosted {
			t.Fatalf("record %d is not in the right panel", n)
		}
		if l := layoutPopup(hosted, e.font, e.bodyFont()); l.Rect != e.sidePanelRect(panelRight) {
			t.Fatalf("record %d window %v, panel %v", n, l.Rect, e.sidePanelRect(panelRight))
		}
		// The same window floating at the panel's size.
		width := e.sidePanelRect(panelRight).Dx()
		e.panels.detached = e.detachKey()
		floating, _, ok := e.objectSpec()
		if !ok || e.panels.hosted {
			t.Fatalf("record %d did not float", n)
		}
		e.sizes[popObject], e.sizeKey[popObject] = image.Pt(width, 400), floating.Title
		floating, _, _ = e.objectSpec()
		if rowsText(hosted) != rowsText(floating) {
			t.Fatalf("record %d\nin the panel\n%s\nfloating\n%s", n, rowsText(hosted), rowsText(floating))
		}
		// The panel then names where the window is.
		if s, ok := e.propsSpec(); !ok || len(s.Rows) == 0 || s.Rows[0].Label != e.word("editor.side.floating") {
			t.Fatalf("record %d floating leaves the panel %v", n, s.Rows)
		}
	}
	// A double click on the hosted window's title takes it out to float.
	e, _, _ := panelsEditor(t, 1280, 800, "")
	e.Select(7, false)
	s, _, _ := e.objectSpec()
	title := layoutPopup(s, e.font, e.bodyFont()).Title
	at := image.Pt(title.Min.X+20, center(title).Y)
	now := time.Unix(1, 0)
	for i := 0; i < 2; i++ {
		e.Pointer(Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}, now)
		e.Pointer(Input{CursorX: at.X, CursorY: at.Y}, now)
	}
	if _, _, ok := e.objectSpec(); !ok || e.panels.hosted {
		t.Fatal("a double click on the title left the trigger window in the panel")
	}
	// Another selection comes back into the panel.
	e.Select(0, false)
	if _, _, ok := e.objectSpec(); !ok || !e.panels.hosted {
		t.Fatal("the next selection stayed floating")
	}
	// With nothing selected the panel shows its one hint line.
	e, _, _ = panelsEditor(t, 1280, 800, "")
	e.selected = -1
	if s, ok := e.propsSpec(); !ok || len(s.Rows) != 1 || s.Rows[0].Label != e.word("editor.side.none") {
		t.Fatalf("an empty right panel shows %v", s.Rows)
	}
	// The full inventory and the dialogue page editor stay floating.
	e.Select(0, false)
	e.obj.section = 2
	if e.nativeUnitWindow() {
		if _, _, ok := e.objectSpec(); !ok || e.panels.hosted {
			t.Fatal("the person window's inventory went into the panel")
		}
	}
}

// (h) The panel state goes to the profile and back; a missing or broken file
// gives the defaults.
func TestMapEditorPanelStateRoundTrip(t *testing.T) {
	defaults := [2]panelState{{true, 260}, {true, 520}}
	for _, c := range []struct {
		name string
		data []byte
		err  error
		want [2]panelState
	}{
		{"missing", nil, os.ErrNotExist, defaults},
		{"unreadable", []byte(`{"left":{"minimized":true}}`), os.ErrPermission, defaults},
		{"broken", []byte(`{"left":`), nil, defaults},
		{"wrong", []byte(`{"left":{"minimized":"yes","width":300}}`), nil, defaults},
		{"list", []byte(`[1,2]`), nil, defaults},
		{"one side", []byte(`{"right":{"minimized":true,"width":700}}`), nil, [2]panelState{{true, 260}, {false, 700}}},
		{"clamped", []byte(`{"left":{"minimized":false,"width":9999},"right":{"minimized":false,"width":1}}`), nil, [2]panelState{{true, 420}, {true, 360}}},
		// A side an earlier build kept as closed reads as minimized.
		{"earlier", []byte(`{"left":{"open":false,"width":300},"right":{"open":true,"width":600}}`), nil, [2]panelState{{false, 300}, {true, 600}}},
	} {
		if got := panelStateOf(c.data, c.err); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	e, _, saved := panelsEditor(t, 1280, 800, "")
	dragPanelEdge(t, e, popTree, 20)
	_ = e.Key("propertiespanel")
	last := (*saved)[len(*saved)-1]
	if string(last) != `{"left":{"minimized":false,"width":280},"right":{"minimized":true,"width":520}}`+"\n" {
		t.Fatalf("saved %q", last)
	}
	f, _, _ := panelsEditor(t, 1280, 800, "")
	f.SetPanelProfile(last, nil, nil)
	if f.panels.state != e.panels.state || f.panelsShown() != [2]int{280, f.panelStrip()} {
		t.Fatalf("read back %v from %v", f.panels.state, e.panels.state)
	}
	var kept map[string]map[string]any
	if json.Unmarshal(last, &kept) != nil || len(kept) != 2 {
		t.Fatalf("the saved state is not one JSON object per side: %q", last)
	}
	// A failed write says why and keeps the panels as they are.
	e.SetPanelProfile(last, nil, func([]byte) error { return fmt.Errorf("disk full") })
	_ = e.Key("objectspanel")
	if e.message != "disk full" || e.panelOpen(panelLeft) {
		t.Fatalf("a failed write: message %q, panels %v", e.message, e.panelsShown())
	}
}

// panelsGoldenPath records the side panels' pixels and rows;
// MAPEDITOR_PANELS_WRITE=1 rewrites it.
const panelsGoldenPath = "testdata/mapeditor_side_panels.golden"

// The side panels as they draw: both open at 1280x800 in English and in
// Russian with the trigger selected, a search applied, and the defaults of
// a 1024x768 window.
func TestMapEditorSidePanelsGolden(t *testing.T) {
	var out strings.Builder
	for _, c := range []struct {
		name   string
		w, h   int
		lang   string
		search string
		record int
	}{
		{"trigger-en-1280x800", 1280, 800, "", "", 7},
		{"trigger-ru-1280x800", 1280, 800, "russian", "", 7},
		{"search-en-1280x800", 1280, 800, "", "unit", 3},
		{"defaults-en-1024x768", 1024, 768, "", "", -1},
	} {
		e, _, _ := panelsEditor(t, c.w, c.h, c.lang)
		if c.record >= 0 {
			e.Select(c.record, false)
		}
		e.tree.search = c.search
		h := fnv.New64a()
		h.Write(e.ChromeImage().Pix)
		fmt.Fprintf(&out, "== %s %016x panels=%v map=%v\n%s", c.name, h.Sum64(), e.panelsShown(), e.mapArea(), treeText(e))
		if s, _, ok := e.objectSpec(); ok && e.panels.hosted {
			out.WriteString(rowsText(s))
		} else if s, ok := e.propsSpec(); ok {
			out.WriteString(rowsText(s))
		}
	}
	got := out.String()
	if os.Getenv("MAPEDITOR_PANELS_WRITE") != "" {
		if err := os.WriteFile(panelsGoldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(panelsGoldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.ReplaceAll(string(raw), "\r\n", "\n"); got != want {
		t.Fatalf("side panels differ from %s:\n%s", panelsGoldenPath, got)
	}
}

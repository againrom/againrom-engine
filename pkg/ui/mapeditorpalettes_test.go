package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// The bar's record entries and the tree's branches are the one category
// table: the same keys in the same order under the same words. Objects and
// Trees are brushes and have no branch.
func TestMapEditorTreeBranchesAreTheBarsRecordEntries(t *testing.T) {
	e, _, _ := panelsEditor(t, 1280, 800, "russian")
	var bar, tree []string
	for _, b := range e.barEntries() {
		if _, ok := categoryByKey(b.key); ok {
			bar = append(bar, b.key+" "+b.label)
		}
	}
	kinds := e.treeRowsNow()
	rows := e.tree.model.Rows()
	for n, r := range kinds {
		if r.branch == "" {
			continue
		}
		c, _ := categoryByKey(r.branch)
		word := e.word(c.word)
		if !strings.HasPrefix(rows[n].Text, word+" (") {
			t.Errorf("branch %s reads %q, not its bar word %q", r.branch, rows[n].Text, word)
		}
		tree = append(tree, r.branch+" "+word)
	}
	if want := []string{"scripts", "players", "groups", "units", "sacks", "structures"}; len(bar) != len(want) || !slices.Equal(bar, tree) {
		t.Fatalf("bar %q, tree %q, want the keys %q", bar, tree, want)
	}
	for i, k := range []string{"scripts", "players", "groups", "units", "sacks", "structures"} {
		if !strings.HasPrefix(bar[i], k+" ") {
			t.Fatalf("bar entry %d is %q, want %s", i, bar[i], k)
		}
	}
	for _, brush := range []string{"objects", "trees"} {
		if e.TreeBranchRow(brush) >= 0 {
			t.Errorf("the brush %s has a tree branch", brush)
		}
	}
}

// Each category's palette holds no record row and no "x-y / n" count: the
// placement grid with its note and Add for Units and Structures, Add for
// Sacks, the hint, the drop zone switch and Add for Scripts, the relation
// matrix and Add for Players, the selection notes and the two group buttons
// for Groups.
func TestMapEditorPalettesHoldNoRecordRow(t *testing.T) {
	count := regexp.MustCompile(`^\d+-\d+ / \d+$`)
	for _, c := range editorCategories {
		t.Run(c.key, func(t *testing.T) {
			e, _ := newModesEditor(t, "full")
			e.showRecords(c.filter)
			s, ok := e.listSpec()
			if !ok || !e.paletteShown() && !e.groupsShown() {
				t.Fatalf("the %s palette did not open", c.key)
			}
			if s.List != nil || e.ListRowCount() != 0 {
				t.Fatalf("the %s palette holds a list of %d rows", c.key, e.ListRowCount())
			}
			summaries := map[string]bool{}
			for id := range e.doc.Records {
				summaries[e.recordSummary(id)] = true
			}
			for _, id := range e.rows() {
				summaries[e.scriptRowText(id, 0)] = true
			}
			for _, r := range s.Rows {
				if summaries[r.Label] {
					t.Errorf("the %s palette holds the record row %q", c.key, r.Label)
				}
			}
			for _, n := range s.Notes {
				if count.MatchString(n) {
					t.Errorf("the %s palette keeps the count %q", c.key, n)
				}
			}
			add := e.word("editor.button.add")
			switch c.key {
			case "units", "structures":
				if s.Grid == nil || len(s.Rows) != 1 || !slices.Contains(s.Buttons, add) {
					t.Errorf("the %s palette lost its grid, note or Add: %+v", c.key, s)
				}
			case "sacks":
				if !slices.Equal(s.Buttons, []string{add}) {
					t.Errorf("the sacks palette buttons %q", s.Buttons)
				}
			case "scripts":
				if !slices.Contains(s.Notes, e.word("editor.scripts.hint")) || len(s.Buttons) != 2 || s.Buttons[1] != add {
					t.Errorf("the scripts palette %q %q", s.Notes, s.Buttons)
				}
			case "players":
				if s.Grid == nil || s.Grid.Rows != 2 || !slices.Contains(s.Buttons, add) {
					t.Errorf("the players palette lost its matrix or Add: %+v", s)
				}
			case "groups":
				if len(s.Notes) == 0 || len(s.Buttons) != 2 {
					t.Errorf("the groups palette %q %q", s.Notes, s.Buttons)
				}
			}
		})
	}
}

// branchShown reports that category key's branch row stands in the tree's
// shown rows of a restored left panel.
func branchShown(e *MapEditor, key string) bool {
	s, ok := e.treeSpec()
	n := e.TreeBranchRow(key)
	if !ok || n < 0 {
		return false
	}
	top, count := s.List.Visible()
	return n >= top && n < top+count
}

// Every route to a palette restores a minimized left panel, opens the
// category's branch and scrolls its title row into view: the bar entry, the
// key, the MCP key call and the MCP pointer click on the bar.
func TestMapEditorPaletteRoutesRevealTheirBranch(t *testing.T) {
	routes := []struct {
		name, key string
		act       func(t *testing.T, e *MapEditor)
	}{
		{"bar", "structures", func(t *testing.T, e *MapEditor) { chromeClick(e, barPoint(t, e, "structures")) }},
		{"key F2", "scripts", func(t *testing.T, e *MapEditor) { _ = e.Key("scripts") }},
		{"key P", "players", func(t *testing.T, e *MapEditor) { _ = e.Key("players") }},
		{"mcp key", "players", func(t *testing.T, e *MapEditor) {
			newMCPPipe(t, e).tool(t, "key", map[string]any{"name": "players"})
		}},
		{"mcp pointer", "sacks", func(t *testing.T, e *MapEditor) {
			at := barPoint(t, e, "sacks")
			newMCPPipe(t, e).tool(t, "pointer", map[string]any{"action": "click", "x": at.X, "y": at.Y})
		}},
	}
	for _, r := range routes {
		t.Run(r.name, func(t *testing.T) {
			e, _, saved := panelsEditor(t, 1280, 800, "")
			e.SetPanelProfile(minimizedPanels, nil, func(b []byte) error {
				*saved = append(*saved, b)
				return nil
			})
			// A closed branch, scrolled out of view, under a search that
			// hides it.
			e.tree.closed = map[string]bool{r.key: true}
			e.tree.search = "zzz"
			if e.panelOpen(panelLeft) {
				t.Fatal("the left panel did not minimize")
			}
			r.act(t, e)
			c, _ := categoryByKey(r.key)
			if !e.panelOpen(panelLeft) || e.tree.closed[r.key] || !branchShown(e, r.key) {
				t.Fatalf("the %s route left the panel %v, the branch closed %v, shown %v", r.name, e.panelOpen(panelLeft), e.tree.closed[r.key], branchShown(e, r.key))
			}
			if !e.paletteShown() || e.filter != c.filter {
				t.Fatalf("the %s route shows %q filter %d", r.name, e.listKind(), e.filter)
			}
			if len(*saved) == 0 || !strings.Contains(string((*saved)[len(*saved)-1]), `"left":{"minimized":false`) {
				t.Fatalf("the restored panel was not kept: %q", *saved)
			}
		})
	}
	// A branch scrolled below the window comes into view.
	e, p, _ := panelsEditor(t, 1280, 400, "")
	for i := 0; i < 40; i++ {
		p.units = append(p.units, groupsUnit{1, 0, image.Pt(i, 20)})
	}
	p.dirty = true
	_ = e.Key("undo")
	if branchShown(e, "structures") {
		t.Fatal("the Structures branch shows before the route")
	}
	chromeClick(e, barPoint(t, e, "structures"))
	if !branchShown(e, "structures") {
		t.Fatal("the Structures entry did not scroll its branch into view")
	}
}

// The tree moves the palette: a press on a branch row shows its palette, a
// member chosen in another branch shows that branch's palette, and a branch
// whose palette holds nothing closes a shown palette.
func TestMapEditorTreeShowsTheBranchPalette(t *testing.T) {
	e, _, _ := panelsEditor(t, 1280, 800, "")
	clickTreeRow(t, e, e.TreeBranchRow("units"))
	if !e.paletteShown() || e.filter != 1 {
		t.Fatalf("a press on the Units branch shows %q filter %d", e.listKind(), e.filter)
	}
	clickTreeRow(t, e, e.TreeMemberRow("players", 1))
	if !e.paletteShown() || e.filter != editorPlayersFilter || e.selected != 9 {
		t.Fatalf("a player chosen shows %q filter %d, selected %d", e.listKind(), e.filter, e.selected)
	}
	// A second member of the same branch keeps the palette as it stands.
	e.listAt = image.Pt(300, 300)
	clickTreeRow(t, e, e.TreeMemberRow("players", 0))
	if !e.paletteShown() || e.filter != editorPlayersFilter || e.selected != 8 || e.listAt != image.Pt(300, 300) {
		t.Fatalf("a second player shows %q filter %d, selected %d", e.listKind(), e.filter, e.selected)
	}
	clickTreeRow(t, e, e.TreeMemberRow("groups", 0))
	if !e.groupsShown() || !e.groupSel.set {
		t.Fatalf("a group chosen shows %q filter %d", e.listKind(), e.filter)
	}
	// This host adds and places nothing: the Units palette is empty, so its
	// branch closes the scripts palette and opens none.
	f, _ := popupEditor(t)
	_ = f.Open("popups")
	_ = f.Key("scripts")
	if !f.paletteShown() || f.filter != editorScriptsFilter {
		t.Fatalf("F2 shows %q", f.listKind())
	}
	clickTreeRow(t, f, f.TreeBranchRow("units"))
	if f.listKind() != "" {
		t.Fatalf("the empty Units palette left %q filter %d", f.listKind(), f.filter)
	}
	chromeClick(f, barPoint(t, f, "units"))
	if f.listKind() != "" || !branchShown(f, "units") {
		t.Fatalf("the Units entry with an empty palette shows %q", f.listKind())
	}
}

// A palette dragged over the tree's search field keeps its title: a press
// there drags the palette and leaves the search field alone.
func TestMapEditorPaletteOverTheSearchFieldMovesByItsTitle(t *testing.T) {
	e, _, _ := panelsEditor(t, 1280, 800, "")
	clickTreeRow(t, e, e.TreeBranchRow("units"))
	if !e.paletteShown() || e.selected >= 0 {
		t.Fatalf("the Units branch shows %q, selected %d", e.listKind(), e.selected)
	}
	drag := func(d image.Point) image.Rectangle {
		before, _ := e.TitleRect()
		p := before.Min.Add(image.Pt(10, 4))
		q := p.Add(d)
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
		e.Pointer(Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, time.Time{})
		e.Pointer(Input{CursorX: q.X, CursorY: q.Y}, time.Time{})
		after, _ := e.TitleRect()
		return after
	}
	box := e.treeSearchBox()
	title, _ := e.TitleRect()
	over := drag(center(box).Sub(title.Min.Add(image.Pt(10, 4))))
	if p := over.Min.Add(image.Pt(10, 4)); !p.In(box) {
		t.Fatalf("the palette's title %v does not lie over the search field %v", over, box)
	}
	moved := drag(image.Pt(200, 100))
	if moved.Min != over.Min.Add(image.Pt(200, 100)) || e.tree.typing {
		t.Fatalf("a drag from over the search field left the title at %v, search typing %v", moved, e.tree.typing)
	}
}

// Minimize and restore: the title row's button minimizes a panel to a
// strip as wide as one button, the strip's one button restores the
// remembered width, the buttons read the EN and RU words, and the state goes
// to the profile and back.
func TestMapEditorPanelsMinimizeToAStrip(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		e, _, saved := panelsEditor(t, 1280, 800, lang)
		strip := e.panelStrip()
		if f := e.font; strip != fontHeight(f, 15)+popupRowExtra+2 || strip <= 0 {
			t.Fatalf("strip %d", strip)
		}
		dragPanelEdge(t, e, popTree, 10)
		for i, want := range []int{270, 520} {
			s, ok := e.treeSpec()
			if i == panelRight {
				s, ok = e.propsSpec()
			}
			if !ok || s.CloseLabel != e.word(sidePanels[i].minimizeWord) {
				t.Fatalf("panel %d title button reads %q", i, s.CloseLabel)
			}
			r, ok := e.PanelButtonRect(i)
			if !ok || !r.In(e.sidePanelRect(i)) {
				t.Fatalf("panel %d minimize button %v", i, r)
			}
			chromeClick(e, center(r))
			if e.panelOpen(i) || e.panelsShown()[i] != strip || e.sidePanelRect(i).Dx() != strip {
				t.Fatalf("panel %d minimized to %v", i, e.sidePanelRect(i))
			}
			st, ok := e.stripSpec(i)
			if !ok || st.CloseLabel != e.word(sidePanels[i].restoreWord) || st.Title != "" {
				t.Fatalf("panel %d strip %+v", i, st)
			}
			r, ok = e.PanelButtonRect(i)
			if !ok || !r.In(e.sidePanelRect(i)) {
				t.Fatalf("panel %d restore button %v in strip %v", i, r, e.sidePanelRect(i))
			}
			chromeClick(e, center(r))
			if !e.panelOpen(i) || e.panelsShown()[i] != want {
				t.Fatalf("panel %d restored to %d, want %d", i, e.panelsShown()[i], want)
			}
		}
		// The words come from the word tables of both languages.
		for _, w := range []string{"editor.side.toleft", "editor.side.toright"} {
			if e.word(w) == w || e.word(w) == "" {
				t.Fatalf("%s has no word in %q", w, lang)
			}
		}
		// The profile keeps minimized and the width.
		_ = e.Key("objectspanel")
		last := (*saved)[len(*saved)-1]
		var kept map[string]struct {
			Minimized bool
			Width     int
		}
		if err := json.Unmarshal(last, &kept); err != nil || !kept["left"].Minimized || kept["left"].Width != 270 || kept["right"].Minimized {
			t.Fatalf("saved %q", last)
		}
		g, _, _ := panelsEditor(t, 1280, 800, lang)
		g.SetPanelProfile(last, nil, nil)
		if g.panelOpen(panelLeft) || g.panelsShown() != [2]int{strip, 520} {
			t.Fatalf("read back %v", g.panelsShown())
		}
		_ = g.Key("objectspanel")
		if g.panelsShown()[panelLeft] != 270 {
			t.Fatalf("the restored width %d", g.panelsShown()[panelLeft])
		}
	}
}

// The group's member pictures, its Delete group button and its Used by
// lines, which its Groups list row held, stand in the right panel when the
// group is chosen in the tree; Delete group takes its units out of it.
func TestMapEditorGroupWindowHoldsTheGroupRow(t *testing.T) {
	e, p, _ := panelsEditor(t, 1280, 800, "")
	for i := range 5 {
		e.doc.Records[i].Preview = &terrain.StaticFrame{Width: 4, Height: 4}
	}
	clickTreeRow(t, e, e.TreeMemberRow("groups", 0))
	s, links, ok := e.objectSpec()
	if !ok || !e.panels.hosted || !e.groupSel.set {
		t.Fatal("the chosen group is not in the right panel")
	}
	want := fmt.Sprintf(e.word("editor.group.number"), 1)
	if len(s.Rows) < 4 || s.Rows[1].Label != want || len(s.Rows[1].Icons) != 2 || s.Rows[1].Value != fmt.Sprintf(e.word("editor.group.count"), 2) {
		t.Fatalf("the group row %+v", s.Rows)
	}
	if !slices.Equal(s.Buttons, []string{e.word("editor.group.delete")}) {
		t.Fatalf("the group window's buttons %q", s.Buttons)
	}
	used := false
	for i, r := range s.Rows {
		used = used || r.Kind == popupLink && links[i] < 0 && e.doc.Records[-links[i]-1].Kind == "Trigger"
	}
	if !used {
		t.Fatalf("the group window names no trigger that uses it: %+v", s.Rows)
	}
	r, ok := e.GroupDeleteRect()
	if !ok || !r.In(e.sidePanelRect(panelRight)) {
		t.Fatalf("Delete group at %v", r)
	}
	// The group is used: Delete group asks first, as the list row's cross
	// did.
	chromeClick(e, center(r))
	if e.prompt.kind != promptGroupWarn || len(p.calls.lines) != 0 {
		t.Fatalf("Delete group of a used group asked %d and called %q", e.prompt.kind, p.calls.lines)
	}
	goOn, _ := e.PromptButtonRect(0)
	chromeClick(e, center(goOn))
	if got := strings.Join(p.calls.lines, "|"); got != "SetUnitsGroup [{Unit 0} {Unit 1}] 0" {
		t.Fatalf("Delete group called %q", got)
	}
}

// A palette grid cell names itself after the pointer rests on it for the
// editor's tooltip delay: none sooner, the name after, none once the
// pointer leaves the cell. The game's hover box draws it.
func TestMapEditorPaletteCellTooltipWaitsForTheDelay(t *testing.T) {
	if editorTipDelay < 200 || editorTipDelay > 300 {
		t.Fatalf("the tooltip delay is %d ms", editorTipDelay)
	}
	e, _ := newModesEditor(t, "full")
	e.listAt = image.Pt(8, e.barHeight()+8)
	chromeClick(e, barPoint(t, e, "units"))
	cell, ok := e.PlaceGridCellRect(1)
	g, _, _ := e.placeGridSpec()
	if !ok || g == nil || g.Cells[1].Tip == "" {
		t.Fatal("the Units palette has no named cell 1")
	}
	name := g.Cells[1].Tip
	at := center(cell)
	t0 := time.Unix(100, 0)
	move := func(p image.Point, d time.Duration) {
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, t0.Add(d))
	}
	move(at, 0)
	move(at.Add(image.Pt(1, 0)), 100*time.Millisecond)
	move(at, editorTipDelay*time.Millisecond-time.Millisecond)
	if got := e.TipShown(); got != "" {
		t.Fatalf("the tooltip %q shows before the delay", got)
	}
	move(at, editorTipDelay*time.Millisecond)
	if got := e.TipShown(); got != name {
		t.Fatalf("after the delay the tooltip is %q, want %q", got, name)
	}
	boxes := 0
	widgetRecord = func(c widgetCall) {
		if lines, _ := c.state.([]string); c.kind == widgetHoverBox && slices.Equal(lines, []string{name}) {
			boxes++
		}
	}
	e.ChromeImage()
	widgetRecord = nil
	if boxes != 1 {
		t.Fatalf("the hover box drew the tooltip %d times", boxes)
	}
	move(image.Pt(cell.Max.X+40, cell.Max.Y+40), editorTipDelay*time.Millisecond+50*time.Millisecond)
	if got := e.TipShown(); got != "" {
		t.Fatalf("the tooltip %q stays off the cell", got)
	}
	// A press stops the clock.
	move(at, time.Second)
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}, t0.Add(2*time.Second))
	if got := e.TipShown(); got != "" {
		t.Fatalf("the tooltip %q shows under a held button", got)
	}
}

// Each palette grid names its cells in the editor's words: the placement
// choice's label, the object's word, the height mode's word and the
// relation's line.
func TestMapEditorPaletteGridsNameTheirCells(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.showRecords(1)
	g, _, _ := e.placeGridSpec()
	for i, c := range e.placeChoices("Unit", "class") {
		if g.Cells[i].Tip != c.Label {
			t.Errorf("unit cell %d names %q, want %q", i, g.Cells[i].Tip, c.Label)
		}
	}
	_ = e.Key("players")
	s, _ := e.listSpec()
	if tip := s.Grid.Cells[1].Tip; !strings.HasPrefix(tip, fmt.Sprintf(e.word("editor.relation.cell"), "Self", "Enemy", "")) {
		t.Errorf("relation cell names %q", tip)
	}
	_ = e.Key("height")
	if g, ok := e.heightModeGridSpec(); ok {
		for m := range heightModeCount {
			if g.Cells[m].Tip != e.heightModeWord(m) {
				t.Errorf("height cell %d names %q", m, g.Cells[m].Tip)
			}
		}
	} else {
		t.Error("the height brush shows no mode grid")
	}
}

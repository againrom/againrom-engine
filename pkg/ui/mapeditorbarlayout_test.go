package ui

import (
	"image"
	"testing"
)

func TestMapEditorSearchUsesTemporaryRowBelowChrome(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	e.Layout(1920, 1080)
	e.openMaps = []*openMap{{doc: e.doc}, {doc: &InspectionDocument{Name: "Second map"}}}
	for _, floating := range []bool{false, true} {
		tabs := e.mapTabsRect()
		toolbar := e.toolbarRect()
		area := e.mapArea()
		file, _ := e.controlBarRect("file")
		if err := e.Key(searchKey); err != nil {
			t.Fatal(err)
		}
		box := e.searchEntry().rect
		if box.Min.X != file.Min.X || box.Min.Y < max(tabs.Max.Y, toolbar.Max.Y) || box.Max.Y > e.mapArea().Min.Y {
			t.Fatalf("floating %v: File %v, tabs %v, toolbar %v, search %v, map %v", floating, file, tabs, toolbar, box, e.mapArea())
		}
		chromeClick(e, center(box))
		if _, ok := searchText(e); !ok {
			t.Fatal("press on the search field lost focus")
		}
		if err := e.Key("escape"); err != nil {
			t.Fatal(err)
		}
		if e.mapArea() != area {
			t.Fatalf("closing search restored map %v, want %v", e.mapArea(), area)
		}
		if !floating {
			e.undockToolbar(image.Pt(900, 400))
		}
	}
}

func TestMapEditorToolbarFitsCompleteGroupsAsItGrows(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	e.Layout(1920, 1080)
	original := toolButtons
	t.Cleanup(func() { toolButtons = original })
	toolButtons = nil
	add := func(n int) {
		for range n {
			for _, group := range []toolGroup{toolLeft, toolRight} {
				toolButtons = append(toolButtons, toolButton{icon: &iconFit, group: group, kind: toolAction})
			}
		}
	}
	assertFit := func(side, rows int) {
		t.Helper()
		bar, area := e.toolbarRect(), e.mapArea()
		line := map[int]bool{}
		groupLine := map[toolGroup]int{}
		entries := e.toolbarEntries()
		for i, button := range entries {
			r := button.rect
			if r.Size() != image.Pt(side, side) || !r.In(bar) || r.Overlaps(e.toolbarGrip()) {
				t.Fatalf("button %d at %v in %v, want %d px square clear of the grip", i, r, bar, side)
			}
			if y, ok := groupLine[button.group]; ok && y != r.Min.Y {
				t.Fatalf("group %d split between %d and %d", button.group, y, r.Min.Y)
			}
			groupLine[button.group], line[r.Min.Y] = r.Min.Y, true
			for _, other := range entries[:i] {
				if r.Overlaps(other.rect) {
					t.Fatalf("two buttons overlap at %v and %v", r, other.rect)
				}
			}
		}
		if len(line) != rows || area.Min.Y != bar.Max.Y || area.Dy() < mapLeastHeight {
			t.Fatalf("got %d rows in %v with map %v, want %d rows and room for the map", len(line), bar, area, rows)
		}
	}
	add(10)
	assertFit(36, 1)
	add(20)
	assertFit(22, 1)
	add(15)
	assertFit(22, 2)
}

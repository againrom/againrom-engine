package ui

import (
	"fmt"
	"image"
	"testing"
)

// pagesHost is a mod host that gives the pages of every scope of its own.
type pagesHost struct{ *modesWorkspace }

func (h pagesHost) WorkspaceFields(scope string) []InspectionField {
	switch scope {
	case ModTerrainScope:
		out := manyGroundsHost{h.modesWorkspace, false}.WorkspaceFields(scope)
		for i := range 60 {
			out = append(out, InspectionField{Key: fmt.Sprintf("game.%d", i), Group: "Game", ReadOnly: true, Text: true, TextValue: "Grass", Lines: []InspectionLine{{Text: "Grass"}}, Picture: image.NewRGBA(image.Rect(0, 0, 32, 32))})
		}
		return out
	case ModTextsScope:
		return []InspectionField{{Key: "newkey", Label: "New key", Text: true}}
	case ModFilesScope:
		return []InspectionField{{Key: "filter", Label: "Filter", Text: true}}
	case ModItemsScope:
		return []InspectionField{{Key: "new.armour", Label: "New armour key", Text: true}}
	}
	return h.modesWorkspace.WorkspaceFields(scope)
}

// A click on any tab of W shows that page, from every page, and the Files
// page is no tab.
func TestModWindowTabClicksSwitchFromEveryPage(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {1024, 768}, {1920, 1080}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) { tabClicks(t, size) })
	}
}

func tabClicks(t *testing.T, size image.Point) {
	e := modesWordsEditor(t, "workspace", "en", size)
	e.doc.Edits = pagesHost{e.doc.Edits.(*modesWorkspace)}
	_ = e.Key("package")
	pages := e.modPages()
	if len(pages) != 4 {
		t.Fatalf("pages %v", pages)
	}
	for _, p := range pages {
		if p == modPageFiles {
			t.Fatal("the Files page is a tab")
		}
	}
	for from := range pages {
		for to := range pages {
			e.turnModPage(from)
			s, _, _ := e.objectSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			if len(l.Tabs) != len(pages) {
				t.Fatalf("%d tabs", len(l.Tabs))
			}
			// Both edges of the tab as drawn: a window the room moved up or
			// left of its first layout still takes the press.
			for _, at := range []image.Point{l.Tabs[to].Min.Add(image.Pt(2, 2)), l.Tabs[to].Max.Sub(image.Pt(3, 3))} {
				e.turnModPage(from)
				modesClick(e, at)
				if e.modPage != pages[to] {
					t.Fatalf("from tab %d a click at %v on tab %d shows page %d, want %d", from, at, to, e.modPage, pages[to])
				}
			}
		}
	}
}

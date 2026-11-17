package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
)

// textsHost is a mod host that gives the texts page's fields.
type textsHost struct{ *modesWorkspace }

func (h textsHost) WorkspaceFields(scope string) []InspectionField {
	if scope == ModTextsScope {
		return []InspectionField{
			{Key: "summary", ReadOnly: true, Text: true, TextValue: "1 keys; missing: EN 0, RU 1", Lines: []InspectionLine{{Text: "1 keys; missing: EN 0, RU 1"}}},
			{Key: "new", Label: "New key", Text: true},
		}
	}
	return h.modesWorkspace.WorkspaceFields(scope)
}

// pagingTextsHost gives the texts page forty text keys, enough to page.
type pagingTextsHost struct{ *modesWorkspace }

func (h pagingTextsHost) WorkspaceFields(scope string) []InspectionField {
	if scope != ModTextsScope {
		return h.modesWorkspace.WorkspaceFields(scope)
	}
	var out []InspectionField
	for n := range 40 {
		out = append(out, InspectionField{Key: fmt.Sprintf("text%02d", n), Label: fmt.Sprintf("Text key %02d", n), Text: true, TextValue: "Mission briefing"})
	}
	return out
}

// The texts page keeps its rows, labels and buttons inside the window at each
// size, with forty keys paging through it.
func TestModTextsPageHasFocusedLayoutProperties(t *testing.T) {
	for _, size := range modPageSizes {
		e := modesWordsEditor(t, "workspace", "en", size)
		host := e.doc.Edits.(*modesWorkspace)
		e.doc.Edits = pagingTextsHost{host}
		_ = e.Key("package")
		e.turnModPage(1)
		if e.modPage != modPageTexts {
			t.Fatalf("%v: W shows page %d, not the texts page", size, e.modPage)
		}
		modPageProperties(t, e, image.Rect(0, 0, size.X, size.Y), fmt.Sprint(size))
	}
}

// W shows the texts page as a tab while the host gives its fields, and a
// field of that page edits the texts scope.
func TestModWindowShowsTheTextsPageOfItsHost(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	e.doc.Edits = textsHost{e.doc.Edits.(*modesWorkspace)}
	_ = e.Key("package")
	tabs, _ := e.modTabs()
	if len(tabs) != 2 || tabs[1] != "Texts" {
		t.Fatalf("tabs %v", tabs)
	}
	e.turnModPage(1)
	if e.modPage != modPageTexts || e.modFieldScope() != ModTextsScope {
		t.Fatalf("page %d scope %q", e.modPage, e.modFieldScope())
	}
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, "New key") {
		t.Fatalf("the texts page lacks its fields:\n%s", lines)
	}
}

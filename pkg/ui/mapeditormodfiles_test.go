package ui

import (
	"strings"
	"testing"
)

// filesHost is a mod host that gives the files page's fields.
type filesHost struct{ *modesWorkspace }

func (h filesHost) WorkspaceFields(scope string) []InspectionField {
	if scope == ModFilesScope {
		return []InspectionField{
			{Key: "filter", Label: "Filter", Text: true},
			{Key: "common:graphics/pic/a.bmp", Group: "graphics/pic/a.bmp", Label: "Replace in COMMON", Text: true, Pick: []string{"bmp"}},
		}
	}
	return h.modesWorkspace.WorkspaceFields(scope)
}

// W holds the files page back: the host gives its fields and W shows no tab
// for them, and no key or tab step reaches the page.
func TestModWindowHoldsTheFilesPageBack(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	e.doc.Edits = filesHost{e.doc.Edits.(*modesWorkspace)}
	_ = e.Key("package")
	if tabs, _ := e.modTabs(); len(tabs) != 0 {
		t.Fatalf("tabs %v", tabs)
	}
	if e.modPages()[len(e.modPages())-1] == modPageFiles || e.turnModPage(1) {
		t.Fatal("the files page is reachable")
	}
	_ = e.Key("package")
	if e.modPage == modPageFiles || e.modFieldScope() == ModFilesScope {
		t.Fatalf("page %d scope %q", e.modPage, e.modFieldScope())
	}
	if lines := strings.Join(e.PopupLines(), "|"); strings.Contains(lines, "Replace in COMMON") {
		t.Fatalf("the files page shows: %s", lines)
	}
}

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

// W shows the files page as a tab while the host gives its fields, and a
// field of that page edits the files scope.
func TestModWindowShowsTheFilesPageOfItsHost(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	e.doc.Edits = filesHost{e.doc.Edits.(*modesWorkspace)}
	_ = e.Key("package")
	tabs, _ := e.modTabs()
	if len(tabs) != 2 || tabs[1] != "Files" {
		t.Fatalf("tabs %v", tabs)
	}
	e.turnModPage(1)
	if e.modPage != modPageFiles || e.modFieldScope() != ModFilesScope {
		t.Fatalf("page %d scope %q", e.modPage, e.modFieldScope())
	}
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, "Replace in COMMON") {
		t.Fatalf("the files page lacks its fields:\n%s", lines)
	}
}

package ui

import (
	"strings"
	"testing"
)

// itemsHost is a mod host that gives the items page's fields.
type itemsHost struct{ *modesWorkspace }

func (h itemsHost) WorkspaceFields(scope string) []InspectionField {
	if scope == ModItemsScope {
		return []InspectionField{{Key: "new.armour", Label: "New armour key", Text: true}}
	}
	return h.modesWorkspace.WorkspaceFields(scope)
}

// W shows the items page as a tab while the host gives its fields, and a
// field of that page edits the items scope.
func TestModWindowShowsTheItemsPageOfItsHost(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	e.doc.Edits = itemsHost{e.doc.Edits.(*modesWorkspace)}
	_ = e.Key("package")
	tabs, _ := e.modTabs()
	if len(tabs) != 2 || tabs[1] != "Items" {
		t.Fatalf("tabs %v", tabs)
	}
	e.turnModPage(1)
	if e.modPage != modPageItems || e.modFieldScope() != ModItemsScope {
		t.Fatalf("page %d scope %q", e.modPage, e.modFieldScope())
	}
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, "New armour key") {
		t.Fatalf("the items page lacks its fields:\n%s", lines)
	}
}

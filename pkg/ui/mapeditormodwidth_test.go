package ui

import (
	"image"
	"testing"
)

// allPagesHost gives the fields of every scope a tab shows.
type allPagesHost struct{ pagesHost }

func (h allPagesHost) WorkspaceFields(scope string) []InspectionField {
	if scope == ModCharactersScope {
		return []InspectionField{{Key: "newcharacter", Label: "New character", Text: true}}
	}
	return h.pagesHost.WorkspaceFields(scope)
}

// The mod window opens at a width where each of its seven tabs (mod, overview,
// campaign graph, terrain, texts, items, characters) is whole in one row.
func TestModWindowDefaultWidthHoldsEveryTabInOneRow(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e := modesWordsEditor(t, "workspace", lang, image.Pt(1280, 800))
		host := e.doc.Edits.(*modesWorkspace)
		host.overview, host.graph = modesOverview, modesGraph()
		e.doc.Edits = allPagesHost{pagesHost{host}}
		_ = e.Key("package")
		if n := len(e.modPages()); n != 7 {
			t.Fatalf("%s: %d pages, want 7", lang, n)
		}
		s, _, _ := e.objectSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		if len(l.Tabs) != 7 {
			t.Fatalf("%s: %d tab rectangles", lang, len(l.Tabs))
		}
		for i, r := range l.Tabs {
			if r.Min.Y != l.Tabs[0].Min.Y {
				t.Fatalf("%s: tab %d at %v is not in the row of tab 0 at %v (window %v)", lang, i, r, l.Tabs[0], l.Rect)
			}
			if r.Dx() < e.bodyFont().Advance(s.Tabs[i])+16 {
				t.Fatalf("%s: tab %d %q is %d wide, caption needs %d", lang, i, s.Tabs[i], r.Dx(), e.bodyFont().Advance(s.Tabs[i])+16)
			}
		}
	}
}

// The editor's window row opens the window maximized, and stays resizable so
// the engine accepts it.
func TestEditorWindowRowOpensMaximized(t *testing.T) {
	if !EditorWindow.Maximized || !EditorWindow.Resizable {
		t.Fatalf("editor window row %+v", EditorWindow)
	}
}

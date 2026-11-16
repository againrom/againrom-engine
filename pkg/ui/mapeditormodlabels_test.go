package ui

import (
	"image"
	"testing"

	"againrom/pkg/words"
)

// labelsHost gives the Texts and Items pages the labels the game gives them,
// each with a (?) mark, in the language of the book.
type labelsHost struct {
	*modesWorkspace
	lang string
}

func (h labelsHost) WorkspaceFields(scope string) []InspectionField {
	b := words.For(h.lang)
	field := func(key, id string) InspectionField {
		return InspectionField{Key: key, Label: b.Text(id), Meaning: b.Text(id + ".hint"), Text: true}
	}
	switch scope {
	case ModTextsScope:
		return []InspectionField{field("new", "editor.texts.new")}
	case ModItemsScope:
		return []InspectionField{field("name", "editor.items.name")}
	}
	return h.modesWorkspace.WorkspaceFields(scope)
}

// Every label, tab caption and button of W shows whole on every page, in both
// languages, at 1920x1080 and at the smaller sizes.
func TestModWindowShowsEveryLabelWhole(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{640, 480}, {1024, 768}, {1920, 1080}} {
			e := modesWordsEditor(t, "workspace", lang, size)
			e.doc.Edits = labelsHost{e.doc.Edits.(*modesWorkspace), lang}
			_ = e.Key("package")
			for page := range e.modPages() {
				e.turnModPage(page)
				s, _, _ := e.objectSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				for i, tab := range l.Tabs {
					if w := e.bodyFont().Advance(s.Tabs[i]); w > tab.Dx()-4 {
						t.Errorf("%s %v page %d: tab %q needs %d px of %d", lang, size, page, s.Tabs[i], w, tab.Dx())
					}
				}
				for _, rr := range l.Rows {
					row := s.Rows[rr.Index]
					if full, cut := rowFull(e.font, e.bodyFont(), popupRow{Label: row.Label, Kind: row.Kind}, rr); cut {
						t.Errorf("%s %v page %d: label %q is cut (label %v)", lang, size, page, full, rr.Label)
					}
				}
			}
		}
	}
}

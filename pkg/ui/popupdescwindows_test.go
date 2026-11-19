package ui

import (
	"image"
	"testing"
)

// A window shows the title its description is given.
func TestEditorWindowsReadTheirDescriptions(t *testing.T) {
	const marked = "editor.button.ok"
	for _, tc := range []struct {
		name string
		desc *func() popupDesc
		spec func(t *testing.T) (popupSpec, *MapEditor)
	}{
		{"reporthint", &reportHintDesc, func(t *testing.T) (popupSpec, *MapEditor) {
			e, _ := newModesEditor(t, "full")
			e.panels.state[panelBottom] = panelState{Restored: true, Width: 320}
			e.relayout()
			s, ok := e.reportHintSpec()
			if !ok {
				t.Fatal("no report hint")
			}
			return s, e
		}},
		{"treesearch", &treeSearchDesc, func(t *testing.T) (popupSpec, *MapEditor) {
			e, _ := newModesEditor(t, "full")
			return e.treeSearchSpec(), e
		}},
		{"groups", &groupsDesc, func(t *testing.T) (popupSpec, *MapEditor) {
			e, _ := groupsEditor(t)
			return e.groupsSpec(), e
		}},
		{"selection", &selectionDesc, func(t *testing.T) (popupSpec, *MapEditor) {
			e, _ := newModesEditor(t, "full")
			e.places[popObject].at = image.Pt(400, 100)
			e.Fit()
			dragOnMap(e, image.Pt(5, 5), image.Pt(35, 35))
			s, ok := e.selectionSpec()
			if !ok {
				t.Fatal("no selection window")
			}
			return s, e
		}},
		{"message", &messageDesc, func(t *testing.T) (popupSpec, *MapEditor) {
			e, _ := newModesEditor(t, "full")
			e.message = "Saved."
			s, ok := e.promptSpecBase()
			if !ok || len(s.Rows) != 1 || s.Rows[0].Value != e.editorHostText("Saved.") || len(s.Buttons) != 1 {
				t.Fatalf("the message window shows rows %+v, buttons %v", s.Rows, s.Buttons)
			}
			return s, e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kept := *tc.desc
			d := kept()
			if d.Window != tc.name {
				t.Fatalf("the description names its window %q", d.Window)
			}
			d.Title, d.TitleArgs = marked, nil
			*tc.desc = func() popupDesc { return d }
			defer func() { *tc.desc = kept }()
			s, e := tc.spec(t)
			if want := e.word(marked); s.Title != want {
				t.Errorf("the window's title is %q, not its description's %q", s.Title, want)
			}
		})
	}
	for i, d := range map[int]popupDesc{panelBottom: reportHintDesc(), panelLeft: treeSearchDesc()} {
		if d.Title != sidePanels[i].titleWord {
			t.Errorf("%s is titled %q, its panel %q", d.Window, d.Title, sidePanels[i].titleWord)
		}
	}
}

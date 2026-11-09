package ui

import (
	"slices"
	"testing"
)

// The Trees window keeps a One kind mode: its button stands in the tool row
// beside Forest, in both languages, the list then holds the clear row and
// the tree classes (not the Objects palette's), no grid stands over them, and
// the window and the rows it shows lie inside the map area at 1920x1080.
func TestMapEditorTreesWindowShowsItsOneKindRows(t *testing.T) {
	for _, lang := range paletteLangs {
		e := paletteEditorIn(t, 1920, 1080, lang, nil, nil)
		room := e.mapArea()

		_, l := paletteGrid(t, e, "trees")
		if !e.forestOn() {
			t.Fatalf("%s: the Trees entry did not open Forest mode", lang)
		}
		r, ok := e.FindControl(ControlQuery{Role: ControlListButton, N: 2, K: 0})
		if !ok {
			t.Fatalf("%s: the tool row has no One kind button", lang)
		}
		chromeClick(e, center(r))
		if e.forestOn() || !e.brush.forest.trees || e.brush.mode != brushObjects {
			t.Fatalf("%s: One kind left the Trees window (forest %v, trees %v)", lang, e.forestOn(), e.brush.forest.trees)
		}

		s := mustListSpec(t, e)
		l = layoutPopup(s, e.font, e.bodyFont())
		if s.Grid != nil {
			t.Errorf("%s: the Trees one-kind window shows a grid", lang)
		}
		tool := e.forestPopupRows()[0]
		wantOne, wantForest := e.word("editor.brush.tool.one"), e.word("editor.brush.tool.forest")
		if tool.Value != wantOne || !slices.Equal(tool.Buttons, []string{wantOne, wantForest}) || wantOne == "" || wantOne == wantForest {
			t.Errorf("%s: the tool row reads %q with buttons %q, want One kind %q and Forest %q", lang, tool.Value, tool.Buttons, wantOne, wantForest)
		}
		if !l.Rect.In(room) {
			t.Errorf("%s: the Trees window %v leaves the map area %v", lang, l.Rect, room)
		}
		for k := 0; k < 2; k++ {
			if b, ok := e.FindControl(ControlQuery{Role: ControlListButton, N: 2, K: k}); !ok || !b.In(room) {
				t.Errorf("%s: tool button %d at %v (%v) leaves the map area %v", lang, k, b, ok, room)
			}
		}
		var names []string
		for _, o := range e.brushObjectRows() {
			names = append(names, o.Name)
		}
		if len(names) != 76 || names[1] != "Tree 1" || names[75] != "Tree 75" {
			t.Errorf("%s: the one-kind rows are %d, first %q", lang, len(names), names[:min(3, len(names))])
		}
		if !l.HasList || l.ListRows < 3 || !l.List.Rect.In(room) {
			t.Errorf("%s: the one-kind list shows %d rows in %v inside %v", lang, l.ListRows, l.List.Rect, room)
		}

		// The Objects window keeps its own rows and the Trees entry returns
		// to the Trees palette.
		chromeClick(e, barPoint(t, e, "objects"))
		if e.brush.forest.trees || len(e.brushObjectRows()) != 1 {
			t.Errorf("%s: the Objects window lists %d rows after the Trees window", lang, len(e.brushObjectRows()))
		}
	}
}

// The One kind and Forest buttons read in the language of the editor.
func TestMapEditorTreesOneKindWordsAreTranslated(t *testing.T) {
	for lang, want := range map[string]string{"en": "One kind", "ru": "Один вид"} {
		e := paletteEditorIn(t, 1920, 1080, lang, nil, nil)
		if got := e.word("editor.brush.tool.one"); got != want {
			t.Errorf("%s: One kind reads %q, want %q", lang, got, want)
		}
	}
}

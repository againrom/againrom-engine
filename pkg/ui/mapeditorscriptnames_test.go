package ui

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/textfit"
)

// textEditor opens a map whose one trigger carries a UTF-8 Cyrillic name and
// one cell area, under a host encoder that writes "WIN" for that name.
func textEditor(t *testing.T) *MapEditor {
	t.Helper()
	build := func(string) (*InspectionDocument, error) {
		v, err := NewViewer("text", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		d := &InspectionDocument{Viewer: v, Source: "text", Width: 50, Height: 50, SourceBytes: []byte("map")}
		d.Records = []InspectionRecord{{Kind: "Trigger", Index: 0, Section: 7, Label: "WIN"}}
		d.Script = &InspectionScriptModel{
			Triggers: []InspectionTrigger{{Record: 0, Name: "Победа"}},
			Areas:    []InspectionArea{{Trigger: 0, Node: -1, Shape: AreaCell, Rect: image.Rect(10, 10, 11, 11)}},
		}
		return d, nil
	}
	e := NewMapEditor(panelFont(), nil, build)
	e.SetBodyFont(panelFont())
	e.SetHostTextEncoder(func(s string) string { return strings.ReplaceAll(s, "Победа", "WIN") })
	e.Layout(1024, 768)
	if err := e.Open("text"); err != nil {
		t.Fatal(err)
	}
	e.Fit()
	return e
}

// The area label and the trigger's tree row draw a trigger's UTF-8 name in the
// font's bytes, through the host encoder.
func TestTriggerNameIsDrawnThroughTheHostEncoder(t *testing.T) {
	e := textEditor(t)
	labels := e.mapLabelsOf(labelTrigger)
	if len(labels) != 1 || labels[0].text != "T1 WIN" {
		t.Fatalf("area labels %+v", labels)
	}
	e.showPanel(panelLeft, true)
	e.showRecords(editorScriptsFilter)
	rows := scriptsTreeRows(e)
	found := false
	for _, r := range rows {
		found = found || strings.HasSuffix(r.Text, "  WIN")
		if strings.Contains(r.Text, "Победа") {
			t.Fatalf("the list row draws UTF-8: %q", r.Text)
		}
	}
	if !found {
		t.Fatalf("no trigger row in %+v", rows)
	}
}

// A message longer than six lines shows whole, every word and no mark, and
// the window body pages when the message is taller than the window.
func TestALongMessageShowsWhole(t *testing.T) {
	e := textEditor(t)
	e.Layout(640, 480)
	e.message = strings.TrimSpace(strings.Repeat("word ", 1200))
	s, ok := e.promptSpec()
	if !ok {
		t.Fatal("no message popup")
	}
	lines := messageLines(s)
	if len(lines) <= 6 {
		t.Fatalf("message lines %q", lines)
	}
	got := strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
	if got != e.message || strings.Contains(got, textfit.Mark) {
		t.Fatalf("the message shows %d of 1200 words, mark %v", len(strings.Fields(got)), strings.Contains(got, textfit.Mark))
	}
	if l := layoutPopup(s, e.font, e.bodyFont()); l.RowBar.Empty() || !l.Rect.In(image.Rect(0, 0, 640, 480)) {
		t.Fatalf("the long message does not page inside the window: bar %v rect %v lines %d rows %d bounds %v", l.RowBar, l.Rect, len(lines), len(l.Rows), s.Bounds)
	}
}

// messageLines are the lines the message window shows: its text rows.
func messageLines(s popupSpec) []string {
	var out []string
	for _, r := range s.Rows {
		if r.Kind == popupText {
			out = append(out, r.Value)
		}
	}
	return out
}

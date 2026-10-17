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

// The area label and the trigger list row draw a trigger's UTF-8 name in the
// font's bytes, through the host encoder.
func TestTriggerNameIsDrawnThroughTheHostEncoder(t *testing.T) {
	e := textEditor(t)
	labels := e.areaLabels()
	if len(labels) != 1 || labels[0].text != "T1 WIN" {
		t.Fatalf("area labels %+v", labels)
	}
	rows, _ := e.listRows("records")
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

// A message longer than six lines ends its sixth line with one mark.
func TestALongMessageIsMarkedOnce(t *testing.T) {
	e := textEditor(t)
	e.message = strings.Repeat("word ", 400)
	s, ok := e.promptSpec()
	if !ok {
		t.Fatal("no message popup")
	}
	if len(s.Notes) < 6 {
		t.Fatalf("notes %q", s.Notes)
	}
	sixth := s.Notes[5]
	if !strings.HasSuffix(sixth, textfit.Mark) || strings.HasSuffix(sixth, textfit.Mark+textfit.Mark) || strings.Count(sixth, textfit.Mark) != 1 {
		t.Fatalf("the sixth line is %q", sixth)
	}
}

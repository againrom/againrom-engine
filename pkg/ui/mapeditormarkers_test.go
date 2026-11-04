package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/text"
)

func TestEditorSpatialRecordMarkerAndJoinedCells(t *testing.T) {
	e := newMapEditorFixture(t)
	e.Layout(1280, 800)
	e.doc.Records = []InspectionRecord{{Kind: KindEnchantment, Index: 0, Spatial: true, Cell: image.Pt(20, 20), Size: image.Pt(1, 1), Marker: "Lightning", LinkedCells: []image.Point{image.Pt(19, 20), image.Pt(21, 20)}}}
	e.selected = -1
	e.view.hideBorder, e.view.hideDrops, e.view.hideAreas = true, true, true
	if got := len(e.markerLines()); got != 12 {
		t.Fatalf("unselected record: %d strokes, want cross + two joins and cell boxes", got)
	}
	p := e.CellPoint(image.Pt(20, 20))
	ox, oy := e.mapOrigin()
	if hits := e.hitsAt(p.Sub(image.Pt(int(ox), int(oy)))); len(hits) != 1 || hits[0] != 0 {
		t.Fatal("map hit did not find marked record", hits)
	}
	labels := e.markerLabels()
	if len(labels) != 1 || labels[0].text != "Lightning" {
		t.Fatal("spell name absent", labels)
	}
	painted := false
	for _, call := range text.Record(func() { e.ChromeImage() }) {
		painted = painted || image.Pt(call.X, call.Y).In(labels[0].rect)
	}
	if !painted {
		t.Fatal("spell label did not reach chrome painter")
	}
}

// A spell name on the map is the install's text, already in the font's bytes;
// the label draws it as given and does not convert it a second time.
func TestEditorSpatialRecordMarkerKeepsTheInstallTextBytes(t *testing.T) {
	e := newMapEditorFixture(t)
	e.Layout(1280, 800)
	e.SetHostTextEncoder(func(s string) string {
		out := []byte(s)
		for i, b := range out {
			if b >= 0x80 {
				out[i] = '?'
			}
		}
		return string(out)
	})
	marker := "13: \x8c\xae\xab\xad\xa8\xef"
	e.doc.Records = []InspectionRecord{{Kind: KindEnchantment, Index: 0, Spatial: true, Cell: image.Pt(20, 20), Size: image.Pt(1, 1), Marker: marker}}
	e.selected = -1
	if labels := e.markerLabels(); len(labels) != 1 || labels[0].text != marker {
		t.Fatalf("marker label %v, want the install text %q", labels, marker)
	}
}

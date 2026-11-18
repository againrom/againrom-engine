package ui

import (
	"image"
	"reflect"
	"testing"
)

// The placement preview draws the record the click then adds.
func TestEditorAddPlacementPreviewIsTheClickedRecord(t *testing.T) {
	e, host := newWalkEditor(t)
	e.placeStart(KindUnit, map[string]int64{"identity": 1})
	cell := image.Pt(25, 20)
	want := e.addPlacement(cell)
	e.Fit()
	modesClick(e, e.CellPoint(cell))
	if len(host.added) != 1 || !reflect.DeepEqual(host.added[0], want) {
		t.Fatalf("the click added %v, the preview held %+v", host.added, want)
	}
}

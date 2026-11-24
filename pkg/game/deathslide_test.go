package game

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// A unit felled in a step keeps a slide that starts where it was drawn alive:
// before the crossing the body goes back to the cell it leaves, its logical
// cell, over the ticks the step has run; past the crossing it goes on to the
// cell it entered over the ticks still owed.
func TestABodyFelledMidStepSlidesFromWhereItWasDrawn(t *testing.T) {
	const id, span = sim.EntityID(7), 19
	for _, c := range []struct {
		name    string
		x       int32
		transit uint16
		want    crossingMemo
	}{
		{"before the crossing", 5, 16, crossingMemo{step: image.Pt(-1, 0), owed: 3, span: span}},
		{"past the crossing", 6, 8, crossingMemo{step: image.Pt(1, 0), owed: 8, span: span}},
	} {
		t.Run(c.name, func(t *testing.T) {
			mw := &mapWorld{prev: map[sim.EntityID]image.Point{id: image.Pt(5, 5)}}
			e := sim.Entity{ID: id, X: c.x, Y: 5, Transit: c.transit, TransitTotal: span,
				Stride: sim.NativeStride{Present: true, FromX: 5, FromY: 5, ToX: 6, ToY: 5}}
			mw.rememberCrossing(e)
			got, ok := mw.crossing[id]
			if !ok || got != c.want {
				t.Fatalf("memo %+v (kept %v), want %+v", got, ok, c.want)
			}
			// Positions in pixels times the span, so no rounding enters: the
			// living unit is drawn back from its step's cell by the transit
			// owed, the body back from its logical cell by the slide owed.
			cell := terrain.CellSize * span
			alive := drawnCell(e).X*cell - (drawnCell(e).X-5)*terrain.CellSize*int(c.transit)
			body := int(e.X)*cell - got.step.X*terrain.CellSize*got.owed
			if alive != body {
				t.Fatalf("the body starts at %d/%d px, the living unit was drawn at %d/%d", body, span, alive, span)
			}
		})
	}
}

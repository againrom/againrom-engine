package mapedit

import (
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
)

// TestSecondFormatPlaneRowsHoldBandAndValues pins the second format's brush
// rows: the edge ring on the height, object and impassable planes, and the
// values each plane takes.
func TestSecondFormatPlaneRowsHoldBandAndValues(t *testing.T) {
	e := &Editor{format: alm.SecondFormat}
	for _, p := range []Plane{PlaneHeight, PlaneObject, PlaneImpassable} {
		if got := e.Band(p); got != secondBand {
			t.Errorf("Band(%d) = %d, want %d", p, got, secondBand)
		}
	}
	for _, p := range []Plane{PlaneTile, PlaneStructure} {
		if got := e.Band(p); got != 0 {
			t.Errorf("Band(%d) = %d, want 0", p, got)
		}
	}
	if r, ok := e.PlaneRule(PlaneObject); !ok || !slices.Equal(r.Values, []int{0}) || r.Refusal != "editor.map.secondobjects" {
		t.Errorf("object row = %+v, %v", r, ok)
	}
	if r, ok := e.PlaneRule(PlaneStructure); !ok || r.Values == nil || len(r.Values) != 0 || r.Refusal != "editor.map.secondbridge" {
		t.Errorf("bridge row = %+v, %v", r, ok)
	}
	if r, ok := e.PlaneRule(PlaneTile); !ok || r.Values != nil {
		t.Errorf("tile row = %+v, %v", r, ok)
	}
	if _, ok := e.PlaneRule(0); ok {
		t.Error("a plane outside the table has a row")
	}
}

// TestFirstFormatTakesEveryPlaneWithNoBand pins the first format: no table,
// so every plane takes any value and no edge ring.
func TestFirstFormatTakesEveryPlaneWithNoBand(t *testing.T) {
	e := &Editor{format: alm.FirstFormat}
	for _, p := range []Plane{PlaneTile, PlaneHeight, PlaneObject, PlaneImpassable, PlaneStructure} {
		r, ok := e.PlaneRule(p)
		if !ok || r.Values != nil || r.Band != 0 || e.Band(p) != 0 {
			t.Errorf("plane %d: row %+v, ok %v, band %d", p, r, ok, e.Band(p))
		}
	}
}

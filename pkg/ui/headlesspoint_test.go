package ui

import (
	"image"
	"testing"
)

func TestHeadlessEntityPointAvoidsTheScrollingEdge(t *testing.T) {
	a, v := hoverHitFixture(t, false)
	v.entities[1].Cell = image.Pt(0, 6)
	x, y, err := a.HeadlessEntityPoint(7)
	if err != nil {
		t.Fatal(err)
	}
	if x < EdgeMargin || y < EdgeMargin {
		t.Fatalf("point %d,%d starts edge scrolling", x, y)
	}
	fx, fy := v.windowToFrame(x, y)
	if ref, ok := v.hoverAt(fx, fy); !ok || ref != (InspectionSubject{InspectionUnit, 7}) {
		t.Fatalf("delivered point hits %v/%v", ref, ok)
	}
}

func TestHeadlessGroundPointUsesAnUnoccupiedPartOfTheCell(t *testing.T) {
	a, v := hoverHitFixture(t, false)
	want := image.Pt(7, 6)
	x, y, err := a.HeadlessGroundPoint(want)
	if err != nil {
		t.Fatal(err)
	}
	fx, fy := v.windowToFrame(x, y)
	col, row, inside := v.groundCellAt(float64(fx), float64(fy))
	if !inside || image.Pt(col, row) != want {
		t.Fatalf("point names %d,%d/%v", col, row, inside)
	}
	if hit, ok := v.hoverAt(fx, fy); ok {
		t.Fatalf("ground point hits %v", hit)
	}
}

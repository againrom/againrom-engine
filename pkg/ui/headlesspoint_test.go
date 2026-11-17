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

func TestHeadlessOutcomeControlUsesRepresentablePixel(t *testing.T) {
	a, seam := failureOutcomeApp(t, nil, nil)
	a.Layout(800, 480)
	v := seam.v
	pic, at, scale, visible := v.noticePresent()
	if !visible || scale != 1 || !pic.Bounds().Add(at).In(image.Rectangle{Max: v.FrameSize()}) {
		t.Fatalf("outcome panel is not whole: visible %v scale %v at %v frame %v", visible, scale, at, v.FrameSize())
	}
	button := v.noticeLayout().Button.Add(at)
	centre := button.Min.Add(button.Max).Div(2)
	if _, _, ok := v.place.FrameToWindow(centre); ok {
		t.Fatalf("setup: centre %v has a window pixel in %v", centre, v.place)
	}
	bestDistance, hits := -1, 0
	window := v.place.WindowSize()
	for y := 0; y < window.Y; y++ {
		for x := 0; x < window.X; x++ {
			if action, hit := v.noticeActionAt(x, y); !hit || action != NoticeExitMain {
				continue
			}
			fx, fy := v.windowToFrame(x, y)
			dx, dy := fx-centre.X, fy-centre.Y
			distance := dx*dx + dy*dy
			if bestDistance < 0 || distance < bestDistance {
				bestDistance = distance
			}
			hits++
		}
	}
	if hits == 0 {
		t.Fatal("outcome control has no clickable window pixel")
	}
	t.Logf("window %v frame %v scale %v panel %v button %v centre %v hits %d nearest squared distance %d",
		window, v.FrameSize(), v.place.Scale(), pic.Bounds().Add(at), button, centre, hits, bestDistance)
	if err := a.HeadlessActivate("exit to main menu"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenMenu || a.flow.viewer != nil {
		t.Fatal("outcome control did not exit the mission")
	}
	fx, fy := v.windowToFrame(a.headlessPointer.X, a.headlessPointer.Y)
	dx, dy := fx-centre.X, fy-centre.Y
	if !image.Pt(fx, fy).In(button) || dx*dx+dy*dy != bestDistance {
		t.Fatalf("activated frame point (%d,%d), want nearest point inside %v at squared distance %d", fx, fy, button, bestDistance)
	}
}

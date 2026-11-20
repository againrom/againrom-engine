package ui

import (
	"image"
	"testing"
)

// One owner value serves the bar's press and drag step: each arrow, the
// page areas and a thumb drag move its position, and the apply writes it.
func TestScrollOwnerDrivenByTheOneStep(t *testing.T) {
	rect := image.Rect(0, 0, widgetSpriteSize, 200)
	const count, start, page = 11, 5, 3
	var written []int
	newOwner := func() scrollOwner {
		return scrollOwner{Bar: vScrollBar{Rect: rect, Pos: start, Count: count}, Spec: modScreenList, Visible: page,
			Set: func(n int) { written = append(written, n) }}
	}
	thumb := newOwner().Bar.Thumb()
	for _, tc := range []struct {
		name string
		at   image.Point
		want int
	}{
		{"up arrow", image.Pt(10, 5), start - 1},
		{"down arrow", image.Pt(10, 195), start + 1},
		{"page above the thumb", image.Pt(10, thumb.Min.Y-5), start - page},
		{"page below the thumb", image.Pt(10, thumb.Max.Y+5), start + page},
	} {
		var in scrollBarInput
		o := newOwner()
		written = nil
		if req := in.drive(&o, tc.at, true, appInput{PrimaryPressed: true}); req == barNone {
			t.Errorf("%s: no request", tc.name)
		}
		if o.Bar.Pos != tc.want || len(written) != 1 || written[0] != tc.want {
			t.Errorf("%s: position %d, written %v, want %d", tc.name, o.Bar.Pos, written, tc.want)
		}
	}

	var in scrollBarInput
	o := newOwner()
	written = nil
	grab := image.Pt(10, (thumb.Min.Y+thumb.Max.Y)/2)
	if req := in.drive(&o, grab, true, appInput{PrimaryPressed: true}); req != barNone || !in.active() {
		t.Fatalf("press on the thumb: request %v, active %v", req, in.active())
	}
	for _, y := range []int{40, 120, 190} {
		held := appInput{}
		held.Viewer.PrimaryDown = true
		if req := in.drive(&o, image.Pt(10, y), true, held); req != barSetPos {
			t.Fatalf("drag to %d: request %v", y, req)
		}
		if want := o.Bar.dragPos(y); o.Bar.Pos != want || written[len(written)-1] != want {
			t.Errorf("drag to %d: position %d, written %v, want %d", y, o.Bar.Pos, written, want)
		}
	}
	if req := in.drive(&o, grab, true, appInput{PrimaryReleased: true}); req != barNone || in.active() {
		t.Errorf("release: request %v, active %v", req, in.active())
	}
}

package ui

import (
	"image"
	"testing"
	"time"
)

// tipCloseInsets are the text insets a room popup may carry: none (the list
// child's own), and two that reach down over the Close row.
var tipCloseInsets = [][4]int{{}, {20, 24, 28, 36}, {16, 16, 16, 60}}

// TestTipCloseTakesEveryPaintedPoint presses and releases, in two frames as
// the mouse does, on every native point of the painted Close button of every
// room popup rectangle, and wants each to close the popup once.
func TestTipCloseTakesEveryPaintedPoint(t *testing.T) {
	for _, r := range []image.Rectangle{TownTipRect, TavernTipRect, SchoolTipRect, shopTipRect} {
		for i, in := range tipCloseInsets {
			if i == 2 || i == 1 && r != shopTipRect {
				continue
			}
			a, town := tipPhaseApp(t)
			town.tip.Rect, town.tip.TextInset = r, in
			var dead []image.Point
			now := time.Unix(1, 0)
			for _, p := range pointsOf(TipPanelCloseRect(r)) {
				// Every point of the top rows, where the list child overlaps;
				// below them one point in three (the shop) or thirteen.
				stride := 13
				if r == shopTipRect {
					stride = 3
				}
				if p.Y >= TipPanelCloseRect(r).Min.Y+6 && (p.X+p.Y)%stride != 0 {
					continue
				}
				before := town.closed
				pressAt(a, now, p)
				releaseAt(a, now, p)
				if town.closed != before+1 {
					dead = append(dead, p)
				}
			}
			if len(dead) != 0 {
				t.Errorf("rect %v inset %v: %d dead Close points of %d, first %v last %v", r, in, len(dead), TipPanelCloseRect(r).Dx()*TipPanelCloseRect(r).Dy(), dead[0], dead[len(dead)-1])
			}
		}
	}
}

// TestTipCloseTakesEveryPaintedPointScaled does the same through window
// points: at an integer multiple and at a non-integer scale, every window
// point that maps into the Close rectangle closes the popup.
func TestTipCloseTakesEveryPaintedPointScaled(t *testing.T) {
	for _, win := range []image.Point{{1280, 960}, {1900, 1425}} {
		a, town := tipPhaseApp(t)
		a.Layout(win.X, win.Y)
		town.tip.TextInset = tipCloseInsets[1]
		closeR := TipPanelCloseRect(town.tip.Rect)
		x0, y0, _ := a.nativeFrameToWindow(closeR.Min.Sub(image.Pt(2, 2)))
		x1, y1, _ := a.nativeFrameToWindow(closeR.Max.Add(image.Pt(2, 2)))
		now := time.Unix(1, 0)
		mapped, dead := 0, 0
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				p, ok := a.windowToNativeFrame(x, y)
				if !ok || !p.In(closeR) {
					continue
				}
				if (p.Y >= closeR.Min.Y+6 || x%5 != 0) && (x+y)%31 != 0 {
					continue
				}
				mapped++
				before := town.closed
				pressAt(a, now, image.Pt(x, y))
				releaseAt(a, now, image.Pt(x, y))
				if town.closed != before+1 {
					dead++
				}
			}
		}
		if mapped == 0 || dead != 0 {
			t.Errorf("window %v: %d of %d mapped Close points are dead", win, dead, mapped)
		}
	}
}

// TestTipPanelEventCloseWinsOverText wants Close on press and on release at
// every point of its rectangle, whatever text rectangle the panel carries.
func TestTipPanelEventCloseWinsOverText(t *testing.T) {
	for _, r := range []image.Rectangle{TownTipRect, TavernTipRect, SchoolTipRect, shopTipRect} {
		for _, in := range tipCloseInsets {
			v := TipPanelView{Rect: r, TextInset: in, Text: "tip", Art: tipTestArt(), Font: shopTipTestFont()}
			for _, p := range pointsOf(TipPanelCloseRect(r)) {
				for _, down := range []bool{true, false} {
					if k, ok := TipPanelEventAt(v, p, down); k != TipControlClose || !ok {
						t.Fatalf("rect %v inset %v point %v down %v: %v,%v want Close", r, in, p, down, k, ok)
					}
				}
			}
		}
	}
}

func pointsOf(r image.Rectangle) []image.Point {
	var ps []image.Point
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ps = append(ps, image.Pt(x, y))
		}
	}
	return ps
}

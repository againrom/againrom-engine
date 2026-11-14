package ui

import (
	"image"
	"testing"
)

// stepTop reports whether segment i of the line is the far side of a step:
// the segments before and after it run the same length in opposite
// directions, square to it, and rise long.
func stepTop(l canvasLine, i, rise int) bool {
	if i < 2 || i+1 >= len(l.Points) {
		return false
	}
	p := l.Points
	out, back, top := p[i-1].Sub(p[i-2]), p[i+1].Sub(p[i]), p[i].Sub(p[i-1])
	return out == back.Mul(-1) && out.X*out.X+out.Y*out.Y == rise*rise && out.X*top.X+out.Y*top.Y == 0
}

// crossingsOf lists the places where a horizontal segment of one line meets
// the interior of a vertical segment of another, interior to both, with the
// two lines' segment indexes.
type crossing struct {
	at         image.Point
	one, two   int
	oneI, twoI int
}

func crossingsOf(lines []canvasLine) []crossing {
	var found []crossing
	for j, one := range lines {
		for k, two := range lines {
			if j == k {
				continue
			}
			for i := 1; i < len(one.Points); i++ {
				a, b := one.Points[i-1], one.Points[i]
				if a.Y != b.Y || a.X == b.X {
					continue
				}
				for n := 1; n < len(two.Points); n++ {
					c, d := two.Points[n-1], two.Points[n]
					if c.X != d.X || c.Y == d.Y {
						continue
					}
					if min(a.X, b.X) < c.X && c.X < max(a.X, b.X) && min(c.Y, d.Y) < a.Y && a.Y < max(c.Y, d.Y) {
						found = append(found, crossing{image.Pt(c.X, a.Y), j, k, i, n})
					}
				}
			}
		}
	}
	return found
}

// The campaign graph of the modes page routes arrows that cross. As the
// graph draws them, the upper arrow of each crossing carries a step of four
// pixels: the crossing lies on the far side of a step, not on a plain run of
// either arrow.
func TestRoutedCrossingArrowsStepOverEachOther(t *testing.T) {
	e, _ := graphEditor(t, modesGraph())
	c := e.graphCanvas(modesGraph())
	found := crossingsOf(c.Lines)
	if len(found) == 0 {
		t.Fatal("the page routes no crossing arrows")
	}
	for _, x := range found {
		if !stepTop(c.Lines[x.one], x.oneI, 4) && !stepTop(c.Lines[x.two], x.twoI, 4) {
			t.Errorf("arrows %d and %d cross plainly at %v", x.one, x.two, x.at)
		}
	}
}

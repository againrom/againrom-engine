package ui

import (
	"image"
	"slices"
	"testing"
)

func TestTriggerWindowLeavesAGapBetweenActions(t *testing.T) {
	for _, size := range []image.Point{{800, 600}, {640, 480}} {
		e, _ := wizardChoiceEditor(t)
		e.Layout(size.X, size.Y)
		s, _, ok := e.objectSpec()
		if !ok {
			t.Fatal("the wizard builds no popup")
		}
		var acts []int
		for n, a := range e.obj.rowActs {
			if slices.Contains(a, actRemoveRow) && s.Rows[n].Ink == editorGold {
				acts = append(acts, n)
			}
		}
		slices.Sort(acts)
		if len(acts) != 2 {
			t.Fatalf("%v: %d action rows, want 2", size, len(acts))
		}
		first, second := acts[0], acts[1]
		if s.Rows[first-1].Kind == popupText && s.Rows[first-1].Label == "" {
			t.Fatalf("%v: a gap stands above the first action", size)
		}
		gap := second - 1
		if g := s.Rows[gap]; g.Kind != popupText || g.Label != "" || len(g.Buttons) != 0 || gap <= first {
			t.Fatalf("%v: row %d above the second action is %+v, want a blank row after the first action's rows", size, gap, g)
		}
		if _, has := e.obj.rowField[gap]; has {
			t.Fatalf("%v: the gap row holds a field", size)
		}
		l := layoutPopup(s, e.font, e.bodyFont())
		rects := map[int]image.Rectangle{}
		for _, r := range l.Rows {
			rects[r.Index] = r.Rect
		}
		for _, n := range []int{first, gap, second} {
			if _, ok := rects[n]; !ok {
				t.Fatalf("%v: row %d is not shown", size, n)
			}
		}
		if rects[gap].Min.Y < rects[first].Max.Y || rects[second].Min.Y != rects[gap].Max.Y {
			t.Fatalf("%v: the gap %v does not stand between %v and %v", size, rects[gap], rects[first], rects[second])
		}
		for _, n := range []int{first, second} {
			p := image.Pt(rects[n].Min.X+4, (rects[n].Min.Y+rects[n].Max.Y)/2)
			if part, row := l.hit(p); part != popupRowLine || row != n {
				t.Fatalf("%v: a press on action row %d hits %v row %d", size, n, part, row)
			}
			if f, ok := e.obj.rowField[n]; !ok || e.fieldRow(f) != n {
				t.Fatalf("%v: the row of action row %d's field moved", size, n)
			}
		}
	}
}

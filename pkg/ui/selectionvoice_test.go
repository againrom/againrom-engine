package ui

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

// svOnMap is mkOnMap with a recording selection reply: owned units 4 and 9, a
// foreign unit 7 and an owned body 12, on cells (3..6, 3).
func svOnMap(t *testing.T) (*App, *Viewer, *[][]uint32) {
	t.Helper()
	a, v := mkOnMap(t)
	calls := new([][]uint32)
	v.SetSelectionAcknowledgment(func(ids []uint32, _ time.Time) {
		*calls = append(*calls, slices.Clone(ids))
	})
	return a, v, calls
}

// svTap is atPress with the Shift level held for both frames when asked.
func svTap(a *App, v *Viewer, col, row int, shift bool) {
	x, y := cellPoint(v, col, row)
	down := atFrame(x, y)
	down.PrimaryPressed, down.Viewer.PrimaryDown, down.Viewer.Shift = true, true, shift
	a.step(down, atAt)
	up := atFrame(x, y)
	up.PrimaryReleased, up.Viewer.Shift = true, shift
	a.step(up, atAt)
}

// svGroup is one group digit through the App's frame with its modifier levels.
func svGroup(a *App, digit int, ctrl, shift bool) {
	in := appInput{GroupKey: true, GroupDigit: digit}
	in.Viewer.Ctrl, in.Viewer.Shift = ctrl, shift
	a.step(in, atAt)
}

// svSameFrame presses on the cell, then releases it in one frame with the
// extra facts of that frame set by also.
func svSameFrame(a *App, v *Viewer, col, row int, also func(*appInput)) {
	x, y := cellPoint(v, col, row)
	down := atFrame(x, y)
	down.PrimaryPressed, down.Viewer.PrimaryDown = true, true
	a.step(down, atAt)
	up := atFrame(x, y)
	up.PrimaryReleased = true
	also(&up)
	a.step(up, atAt)
}

// svDragShift is atDrag with the Shift level held on every frame.
func svDragShift(a *App, v *Viewer, c0, r0, c1, r1 int) {
	x0, y0, x1, y1 := cellSpan(v, c0, r0, c1, r1)
	down := atFrame(x0, y0)
	down.PrimaryPressed, down.Viewer.PrimaryDown, down.Viewer.Shift = true, true, true
	a.step(down, atAt)
	mid := atFrame(x1, y1)
	mid.Viewer.PrimaryDown, mid.Viewer.Shift = true, true
	a.step(mid, atAt)
	up := atFrame(x1, y1)
	up.PrimaryReleased, up.Viewer.Shift = true, true
	a.step(up, atAt)
}

// TestSelectionFormsNameTheirUnitsToTheSelectionReply drives each selection
// form through the App's own frame. Only a plain click and a plain marquee ask
// for a reply, once per frame, naming the units they put into the selection;
// ownership and speech are the reply's own business, so a foreign unit is
// named too. The E key, a group recall and any form with Shift held ask for
// none.
func TestSelectionFormsNameTheirUnitsToTheSelectionReply(t *testing.T) {
	for _, tc := range []struct {
		name  string
		sel   selection
		do    func(*App, *Viewer)
		want  [][]uint32
		after selection
	}{
		{"click on an owned unit", nil, func(a *App, v *Viewer) { svTap(a, v, 3, 3, false) }, [][]uint32{{4}}, selection{4}},
		{"click on the selected unit again", selection{4}, func(a *App, v *Viewer) { svTap(a, v, 3, 3, false) }, [][]uint32{{4}}, selection{4}},
		{"click on a foreign unit", nil, func(a *App, v *Viewer) { svTap(a, v, 4, 3, false) }, [][]uint32{{7}}, selection{7}},
		{"click on empty ground", selection{4}, func(a *App, v *Viewer) { svTap(a, v, atEmptyCol, atEmptyRow, false) }, nil, selection{4}},
		{"marquee over the row", nil, func(a *App, v *Viewer) { atDrag(a, v, 3, 3, 6, 3) }, [][]uint32{{4, 9}}, selection{4, 9}},
		{"Shift click adding", selection{4}, func(a *App, v *Viewer) { svTap(a, v, 5, 3, true) }, nil, selection{4, 9}},
		{"Shift marquee adding", selection{4}, func(a *App, v *Viewer) { svDragShift(a, v, 5, 3, 5, 3) }, nil, selection{4, 9}},
		{"Shift click removing", selection{4, 9}, func(a *App, v *Viewer) { svTap(a, v, 5, 3, true) }, nil, selection{4}},
		{"E", selection{7}, func(a *App, v *Viewer) { a.step(appInput{SelectAll: true}, atAt) }, nil, selection{4, 9}},
		{"group recall", selection{4}, func(a *App, v *Viewer) {
			v.groups[2] = selection{9}
			svGroup(a, 2, false, false)
		}, nil, selection{9}},
		{"Shift group recall", selection{4}, func(a *App, v *Viewer) {
			v.groups[2] = selection{9}
			svGroup(a, 2, false, true)
		}, nil, selection{4, 9}},
		{"group assignment", selection{4}, func(a *App, v *Viewer) { svGroup(a, 2, true, false) }, nil, selection{4}},
		{"recall of an empty group", selection{4}, func(a *App, v *Viewer) { svGroup(a, 3, false, false) }, nil, selection{4}},
		{"right click", selection{4}, func(a *App, v *Viewer) { atRightClick(a, v, atEmptyCol, atEmptyRow) }, nil, nil},
		{"click cancelled in its own frame", nil, func(a *App, v *Viewer) {
			svSameFrame(a, v, 5, 3, func(in *appInput) { in.SecondaryReleased = true })
		}, nil, nil},
		{"E and a click in one frame", nil, func(a *App, v *Viewer) {
			svSameFrame(a, v, 3, 3, func(in *appInput) { in.SelectAll = true })
		}, [][]uint32{{4}}, selection{4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v, calls := svOnMap(t)
			v.sel = tc.sel
			tc.do(a, v)
			if !reflect.DeepEqual(*calls, tc.want) || !slices.Equal(v.sel, tc.after) {
				t.Fatalf("selection replies %v and selection %v, want %v and %v", *calls, v.sel, tc.want, tc.after)
			}
		})
	}
}

// TestAnOrderingClickIsNotASelectionReply gives the selected unit a move order:
// the command reply hears it and the selection reply does not. Without a
// selection reply the same click still selects.
func TestAnOrderingClickIsNotASelectionReply(t *testing.T) {
	a, v, calls := svOnMap(t)
	var ordered [][]uint32
	a.flow.order = func(uint32, int, int) {}
	v.SetCommandAcknowledgment(func(_ VoiceGesture, ids []uint32, _ time.Time) { ordered = append(ordered, slices.Clone(ids)) })
	v.sel = selection{4}
	svTap(a, v, atEmptyCol, atEmptyRow, false)
	if len(*calls) != 0 || !reflect.DeepEqual(ordered, [][]uint32{{4}}) {
		t.Fatalf("selection replies %v, command replies %v", *calls, ordered)
	}
	v.SetSelectionAcknowledgment(nil)
	svTap(a, v, 5, 3, false)
	if !slices.Equal(v.sel, selection{9}) {
		t.Fatal("without a selection reply the click left", v.sel)
	}
}

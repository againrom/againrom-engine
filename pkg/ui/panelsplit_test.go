package ui

import (
	"bytes"
	"image"
	"testing"
)

// The composition split: the box's geometry, measurement, fit and paint take
// already-resolved rows, and the unit panel is that same path with its own
// resolution in front.

// TestComposeItemsIsThePanelsOwnPath — 0060 SC-1: a hand-built row list composes
// the identical picture the subject-driven path composes, pixel for pixel.
//
// This is what makes the split a refactor rather than a second drawing path. If
// the shared half ever grows a branch that depends on where its rows came from,
// these two images stop being the same bytes and this fails; asserting that the
// unit panel still looks right, on its own, could not see such a branch at all.
func TestComposeItemsIsThePanelsOwnPath(t *testing.T) {
	f := panelFont()
	l := panelInkLayout()
	s := panelSubjectFixture()

	want := composePanel(l, f, s)
	if want == nil {
		t.Fatal("the unit panel composed nothing to compare against")
	}

	// The authored layout's rows for THIS subject: the count row has no value
	// below two, and the fixture carries no character and no combat numbers, so
	// what is left is the name, the health block — a heading and the pair, both
	// slid into the left column because the statistics they pair with state
	// nothing — and the cell.
	got := composeItems(l, f, []panelItem{
		{value: "Warrior"},
		{label: "HEALTH", value: ""},
		{value: "63/100"},
		{label: "CELL", value: "12, 34"},
	})
	if got == nil {
		t.Fatal("composeItems composed nothing")
	}

	if got.Bounds() != want.Bounds() {
		t.Fatalf("the boxes differ: items %v, subject %v", got.Bounds(), want.Bounds())
	}
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Error("the same rows through composeItems and through composePanel are different pixels")
	}
}

func TestLayoutLinesFlowsTheRowsItIsGiven(t *testing.T) {
	f := panelFont()
	l := panelInkLayout()

	lines := layoutLines(l, f, []panelItem{
		{label: "A", value: "1"},
		{label: "BB", value: "22"},
		{label: "CCC", value: "333"},
	})
	if len(lines) != 3 {
		t.Fatalf("laid out %d lines, want 3", len(lines))
	}
	step := f.Height() + l.Gap
	for i, ln := range lines {
		want := image.Pt(l.Pad.X, l.Pad.Y+i*step)
		if ln.at != want {
			t.Errorf("line %d sits at %v, want %v", i, ln.at, want)
		}
	}

	// A placed layout reads each row's own offset instead, and the same call
	// serves both — which is what keeps the readout free to use either.
	placed := l
	placed.Flow = false
	at := image.Pt(31, 41)
	got := layoutLines(placed, f, []panelItem{{label: "A", value: "1", at: at}})
	if len(got) != 1 || got[0].at != at {
		t.Errorf("a placed row landed at %v, want %v", got, at)
	}
}

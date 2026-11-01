package ui

import (
	"image"
	"image/color"
	"testing"
)

// focusSubjects are the fixture's objects by owner: unit 0 is player 1's, unit
// 1 player 2's, unit 3 owner 0; structure 0 is player 2's, structure 1 owner 0.
var (
	unitOf1   = InspectionSubject{Kind: InspectionUnit, ID: 0}
	unitOf2   = InspectionSubject{Kind: InspectionUnit, ID: 1}
	unitOf0   = InspectionSubject{Kind: InspectionUnit, ID: 3}
	structOf2 = InspectionSubject{Kind: InspectionStructure, ID: 0}
	structOf0 = InspectionSubject{Kind: InspectionStructure, ID: 1}
)

func focusTint(x *tintFixture, s InspectionSubject) spriteTint {
	f := x.e.playerTint()
	if f == nil {
		return spriteTint{}
	}
	return f(s)
}

func TestPlayerFocusTintsOnlyTheSelectedPlayer(t *testing.T) {
	x := newTintFixture(t)
	e := x.e
	e.doc.Edits = &tintProbe{}
	if focusTint(x, unitOf2).On {
		t.Fatal("tint answered with the layer off")
	}
	_ = e.Key("playertint")
	ordinary := func(s InspectionSubject, player int) {
		t.Helper()
		got := focusTint(x, s)
		if want := (spriteTint{Color: editorPlayerTints[player-1], On: true}); got != want {
			t.Fatalf("%v answers %+v, want the ordinary %+v", s, got, want)
		}
	}
	all := func() {
		t.Helper()
		ordinary(unitOf1, 1)
		ordinary(unitOf2, 2)
		ordinary(structOf2, 2)
		if focusTint(x, unitOf0).On || focusTint(x, structOf0).On {
			t.Fatal("an owner-0 object is tinted")
		}
	}
	e.selected = -1
	all()
	e.selected = 3 // a unit record
	all()
	e.selected = 1 // player 2
	want := spriteTint{Color: editorPlayerTints[1], On: true, Focus: true}
	if got := focusTint(x, unitOf2); got != want {
		t.Fatalf("player 2's unit answers %+v, want %+v", got, want)
	}
	if got := focusTint(x, structOf2); got != want {
		t.Fatalf("player 2's structure answers %+v, want %+v", got, want)
	}
	for _, s := range []InspectionSubject{unitOf1, unitOf0, structOf0} {
		if focusTint(x, s).On {
			t.Fatalf("%v is tinted while player 2 is selected", s)
		}
	}
	_ = e.Key("playertint")
	if e.playerTint() != nil {
		t.Fatal("a player selection tints with the layer off")
	}
	_ = e.Key("playertint")
	e.selected = -1
	all()
}

func TestPlayerFocusFollowsTheTreeRowAndThePlayersFilter(t *testing.T) {
	x := newTintFixture(t)
	e := x.e
	e.doc.Edits = &tintProbe{}
	_ = e.Key("playertint")
	e.Layout(1280, 800)
	if !e.panelOpen(panelLeft) {
		_ = e.Key("objectspanel")
	}
	clickTreeRow(t, e, e.TreeMemberRow("players", 1))
	if got := focusTint(x, unitOf2); !got.Focus || !got.On {
		t.Fatalf("after the tree row, player 2's unit answers %+v", got)
	}
	if focusTint(x, unitOf1).On {
		t.Fatal("after the tree row, player 1's unit is tinted")
	}
	e.selected = -1
	e.showRecords(editorPlayersFilter)
	if got := focusTint(x, unitOf1); !got.Focus || !got.On {
		t.Fatalf("after the Players filter, player 1's unit answers %+v (selected %d)", got, e.selected)
	}
	if focusTint(x, unitOf2).On {
		t.Fatal("after the Players filter, player 2's unit is tinted")
	}
}

func TestFocusTintPaintsBrighterAndKeepsAlpha(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.SetRGBA(0, 0, color.RGBA{100, 150, 200, 255})
	c := color.RGBA{0x3c, 0xb4, 0x4b, 255}
	plain := tintPixels(src, c, false)
	focus := tintPixels(src, c, true)
	if plain.Pix[0] == focus.Pix[0] && plain.Pix[1] == focus.Pix[1] && plain.Pix[2] == focus.Pix[2] {
		t.Fatal("the focus tint paints the ordinary pixel")
	}
	if focus.Pix[3] != 255 || focus.Pix[7] != 0 {
		t.Fatalf("the focus tint changed the alpha: %v", focus.Pix)
	}
	lum := func(p []uint8) int { return int(p[0]) + int(p[1]) + int(p[2]) }
	if lum(focus.Pix) <= lum(plain.Pix) {
		t.Fatalf("focus pixel %v is not brighter than %v", focus.Pix[:3], plain.Pix[:3])
	}
}

func TestPlayerLegendMarksTheSelectedPlayer(t *testing.T) {
	x := newTintFixture(t)
	e := x.e
	e.doc.Edits = &tintProbe{}
	_ = e.Key("playertint")
	_, rows := e.playerLegend()
	for i, r := range rows {
		if r.focus || r.muted {
			t.Fatalf("row %d marked with nothing selected", i)
		}
	}
	e.selected = 1
	_, rows = e.playerLegend()
	for i, r := range rows {
		if r.focus != (i == 1) || r.muted != (i != 1) {
			t.Fatalf("row %d focus=%v muted=%v with player 2 selected", i, r.focus, r.muted)
		}
	}
	pic := e.ChromeImage()
	for i, r := range rows {
		mid := image.Pt(r.rect.Min.X+legendSwatch/2, r.rect.Min.Y+r.rect.Dy()/2)
		if got := pic.RGBAAt(mid.X, mid.Y); (got == r.color) == r.muted {
			t.Fatalf("row %d swatch centre %v: filled=%v but muted=%v", i, got, got == r.color, r.muted)
		}
	}
	if got := pic.RGBAAt(rows[1].rect.Min.X, rows[1].rect.Min.Y+rows[1].rect.Dy()/2); got != editorGold {
		t.Fatalf("selected row has no gold outline: %v", got)
	}
}

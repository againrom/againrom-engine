package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/ui"
)

// tavernViewFixture is a synthetic first-game tavern: its screen over the
// synthetic tavern art, and a view of two mercenary seats with a selected
// candidate whose slot mask holds one worn-item pixel. The view carries no
// page; each test sets it as the release tests do.
func tavernViewFixture(t *testing.T) (*townScreen, ui.TownSurfaceView, *ui.TownTavernArt) {
	t.Helper()
	art, err := LoadTownTavernArt(ROM1TownDescription(), townTavernSource())
	if err != nil {
		t.Fatal(err)
	}
	s := bareTownScreen(t)
	s.in.TownTavernArt = resolved(art, nil)
	mask := &ui.SlotMask{W: 20, H: 30, Slot: make([]uint8, 20*30)}
	mask.Slot[7*mask.W+5] = 3
	v := ui.TownSurfaceView{Kind: ui.TownSurfaceTavern, TavernArt: art, HoverCell: -1,
		Cells:     []ui.TownSurfaceCell{{Portrait: true, Selected: true}, {Portrait: true}},
		Candidate: ui.TownCharacterView{HasSubject: true, Figure: image.NewRGBA(image.Rect(0, 0, 20, 30))},
	}
	v.CandidateSlotMask = mask
	v.CandidateSlotInfo[2] = []string{"worn item"}
	return s, v, art
}

// hoverPoint is the screen point of mask pixel (5,7) for a view whose page
// places the figure.
func hoverPoint(v ui.TownSurfaceView) image.Point {
	f := v.Page.Person.Figure
	return f.Origin(image.Pt(v.CandidateSlotMask.W, v.CandidateSlotMask.H)).Add(image.Pt(5, 7))
}

// TestTavernViewPageGivesTheViewItsSeatsAndHover builds the view the way the
// tavern release tests do, through setTavernPage: its first seat is painted
// from the room data, a click on it selects that seat and the candidate's
// worn-item hover answers over its figure. The same view without a page
// paints no seat, answers no click and no hover, so a hand-built view would
// fail every check.
func TestTavernViewPageGivesTheViewItsSeatsAndHover(t *testing.T) {
	s, v, art := tavernViewFixture(t)
	bare := v

	s.setTavernPage(&v, len(v.Cells), len(v.Cells))
	seat := image.Pt(180, 420)
	hireBack := color.RGBAModel.Convert(art.ManBack.At(art.ManBack.Bounds().Min.X+4, art.ManBack.Bounds().Min.Y+4)).(color.RGBA)

	if got := ui.ComposeTownSurface(v).RGBAAt(seat.X, seat.Y); got != hireBack {
		t.Errorf("built page paints seat 0 at %v as %v, want the hire back %v", seat, got, hireBack)
	}
	if c, ok := ui.TownSurfaceControlAt(v, seat); !ok || c.Kind != ui.TownSurfaceControlCell || c.Index != 0 {
		t.Errorf("built page answers a click at %v with %+v, %v; want seat 0", seat, c, ok)
	}
	if c, ok := ui.TownSurfaceControlAt(v, image.Pt(228, 420)); !ok || c.Index != 1 {
		t.Errorf("built page answers a click on seat 1 with %+v, %v", c, ok)
	}
	if lines, ok := ui.TownCandidateHoverLines(v, hoverPoint(v)); !ok || len(lines) != 1 || lines[0] != "worn item" {
		t.Errorf("built page answers the worn-item hover with %q, %v", lines, ok)
	}

	// The instrument sees the difference: the hand-built view fails all three.
	if got := ui.ComposeTownSurface(bare).RGBAAt(seat.X, seat.Y); got == hireBack {
		t.Errorf("a view without a page paints seat 0 at %v as %v", seat, got)
	}
	if c, ok := ui.TownSurfaceControlAt(bare, seat); ok {
		t.Errorf("a view without a page answers a click at %v with %+v", seat, c)
	}
	if lines, ok := ui.TownCandidateHoverLines(bare, hoverPoint(v)); ok {
		t.Errorf("a view without a page answers the hover with %q", lines)
	}
}

// TestTavernPageLayoutPaintsTheFirstGamesSeats is the construction the price
// census uses: one mercenary seat from the install resources alone.
func TestTavernPageLayoutPaintsTheFirstGamesSeats(t *testing.T) {
	_, v, art := tavernViewFixture(t)
	in := &InstallResources{TownTavernArt: resolved(art, nil)}
	v.Cells, v.Candidate, v.CandidateSlotMask = v.Cells[:1], ui.TownCharacterView{}, nil
	seat := image.Pt(180, 420)
	hireBack := color.RGBAModel.Convert(art.ManBack.At(art.ManBack.Bounds().Min.X+4, art.ManBack.Bounds().Min.Y+4)).(color.RGBA)

	if got := ui.ComposeTownSurface(v).RGBAAt(seat.X, seat.Y); got == hireBack {
		t.Fatalf("a view without a page paints seat 0 at %v as %v", seat, got)
	}
	v.Page = in.TavernPageLayout(len(v.Cells), len(v.Cells))
	if got := ui.ComposeTownSurface(v).RGBAAt(seat.X, seat.Y); got != hireBack {
		t.Errorf("TavernPageLayout paints seat 0 at %v as %v, want the hire back %v", seat, got, hireBack)
	}
	if c, ok := ui.TownSurfaceControlAt(v, seat); !ok || c.Index != 0 {
		t.Errorf("TavernPageLayout answers a click at %v with %+v, %v", seat, c, ok)
	}
}

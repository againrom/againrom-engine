package ui

// The picture box's three sources and the spellbook cell's two (0141;
// `UNIT-PICT-035`, `MAGIC-ICON-024`).
//
// WHAT CHANGED IS AN ORDER, NOT A BOX. 0140 shipped two sources here — the
// composed figure for the party's own character and, for anything else, that
// unit's own world sprite as a disclosed placeholder. A third now sits between
// them, and the placeholder becomes what a MISSING FILE falls back to rather
// than what a missing decode did.

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// The shipped geometries, spelled here as plain numbers because this package may
// not import pkg/data (internal/archtest's allow-map) and must not learn a
// format either. They are the measured ones — every portrait node is 160x240 on
// both roots (`SPR256-PICT-043`) and an icon cell is 36x36 (`MAGIC-ICON-024`) —
// and what the two tests below assert is that THIS window's boxes hold them
// without scaling anything.
const (
	shippedPortraitW, shippedPortraitH = 160, 240
	shippedSpellIconSide               = 36
)

// portraitViewer is a viewer on a map with two entities: 5, which is the
// inventory subject and therefore has a composed figure, and 6, which is not.
func portraitViewer(t *testing.T) *Viewer {
	t.Helper()
	v := inventoryTestApp(t).flow.viewer
	layoutViewport(v, MenuWindowW, MenuWindowH)
	frame := &terrain.StaticFrame{Width: 4, Height: 4, Pixels: make([]terrain.StaticPixel, 16)}
	v.SetEntities([]MapEntity{
		{ID: 5, Cell: image.Pt(1, 1), Frame: frame},
		{ID: 6, Cell: image.Pt(2, 2), Frame: frame},
	})
	return v
}

// The source order is independently observable: figure, portrait, then frame.
func TestTheDollBoxPrefersFigureThenPortraitThenFrame(t *testing.T) {
	figure := solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff})
	portrait := solidPic(shippedPortraitW, shippedPortraitH, color.RGBA{G: 0xff, A: 0xff})

	t.Run("the subject's own figure wins over a portrait pushed for it", func(t *testing.T) {
		v := portraitViewer(t)
		v.sel = selection{5}
		v.SetInventorySubject(InventorySubject{ID: 5, Figure: figure})
		v.SetUnitPortrait(5, portrait)

		src, ok := v.dollSubject()
		if !ok {
			t.Fatal("no doll source with the subject selected")
		}
		if dollPicture(src) != figure {
			t.Error("the pushed portrait won over the composed figure")
		}
	})

	t.Run("an ally portrait wins over its world frame", func(t *testing.T) {
		v := portraitViewer(t)
		v.sel = selection{6}
		v.SetUnitPortrait(6, portrait)
		src, ok := v.dollSubject()
		if !ok || dollPicture(src) != portrait {
			t.Fatalf("ally doll source = %+v/%v, want its portrait", src, ok)
		}
	})

	t.Run("the world frame remains the final fallback", func(t *testing.T) {
		v := portraitViewer(t)
		v.sel = selection{6}
		src, ok := v.dollSubject()
		if !ok || src.frame == nil || dollPicture(src) == nil {
			t.Fatalf("frame fallback = %+v/%v, want a picture", src, ok)
		}
	})
}

// The portrait's own geometry fits the box that draws it, unscaled: the shipped
// nodes are the same 160x240 an equipment figure sheet is, and this window
// centres and clips a picture but never resamples one.
func TestAShippedPortraitFitsTheFigureBoxExactly(t *testing.T) {
	if invFigureW != shippedPortraitW || invFigureH != shippedPortraitH {
		t.Errorf("the figure box is %dx%d and a shipped portrait is %dx%d — one of them "+
			"would be clipped", invFigureW, invFigureH, shippedPortraitW, shippedPortraitH)
	}
}

// The spellbook cell draws the icon where there is one and the abbreviation
// where there is not — never both, and the cell is sized so the icon lands
// unscaled inside its border.
func TestASpellCellDrawsItsIconOrItsLetters(t *testing.T) {
	if bookCellSize != shippedSpellIconSide {
		t.Errorf("a book cell is %d and an icon is %d — the icon would be clipped "+
			"or would float", bookCellSize, shippedSpellIconSide)
	}

	bar := image.Rect(0, 0, 400, bookBarH)
	f := panelFont()

	icon := solidPic(shippedSpellIconSide, shippedSpellIconSide, color.RGBA{B: 0xff, A: 0xff})
	withIcon := composeSpellBar(f, []SpellEntry{{ID: 1, Name: "Fire Arrow", Icon: icon}}, 0, 4, bar, 0)
	withLetters := composeSpellBar(f, []SpellEntry{{ID: 1, Name: "Fire Arrow"}}, 0, 4, bar, 0)

	cell := bookCellRects(bar, 4)[0].Sub(bar.Min)
	if !invCellHasArt(withIcon, cell) {
		t.Error("a cell holding an icon drew nothing")
	}
	if !invCellHasArt(withLetters, cell) {
		t.Error("a cell holding no icon drew no letters either")
	}
	// The two are different pictures, which is what says the icon is not simply
	// the letters again.
	if same := imagesEqual(withIcon, withLetters); same {
		t.Error("a cell with an icon and one without composed identically")
	}

	// THE ICON'S OWN PIXELS REACH THE CELL. The fixture is one flat colour, so
	// the cell's middle must be exactly it — an icon drawn through some other
	// path, or scaled, would not land there.
	mid := image.Pt((cell.Min.X+cell.Max.X)/2, (cell.Min.Y+cell.Max.Y)/2)
	if got := withIcon.RGBAAt(mid.X, mid.Y); got != (color.RGBA{B: 0xff, A: 0xff}) {
		t.Errorf("the cell's middle is %+v, want the icon's own colour", got)
	}
}

// imagesEqual is a byte compare over two pictures of the same bounds.
func imagesEqual(a, b *image.RGBA) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}

package game

import (
	"image"
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseTownSquareOverlayDoesNotPaintBlackOverTheSky(t *testing.T) {
	f := releaseFront(t)
	if f.TownSquareArt.Value() == nil || f.TownSquareArt.Value().Background == nil {
		t.Fatalf("production town square art did not resolve: %v", f.TownSquareArt.Err())
	}
	art := *f.TownSquareArt.Value()
	art.Exterior = nil // this witness isolates the overlay from later motion families
	composed := ui.ComposeTownSquare(ui.TownSquareView{Art: &art, Font: f.Font.Value(),
		Exterior: &ui.TownExteriorFrame{BirdOverlayVisible: true}})
	base := f.TownSquareArt.Value().Background
	bb := base.Bounds()
	bad := 0
	for y := bb.Min.Y; y < bb.Max.Y; y++ {
		for x := bb.Min.X; x < bb.Max.X; x++ {
			br, bg, bl, _ := base.At(x, y).RGBA()
			if br == 0 && bg == 0 && bl == 0 {
				continue // the base itself is black here; not this test's own subject
			}
			cr, cg, cl, _ := composed.At(x, y).RGBA()
			if cr == 0 && cg == 0 && cl == 0 {
				bad++
			}
		}
	}
	if bad != 0 {
		t.Fatalf("composed town square: %d pixel(s) are pure black where the base picture "+
			"is not, want 0 (the overlay's own transparent surround painted opaque)", bad)
	}
}

// TestReleaseTownSquareOverlayIsKeyedAtLoad is a second, cheaper witness from
// the same fixture: LoadTownSquareArt keys town_add.bmp (keyBlack) before
// ComposeTownSquare ever sees it, so the loaded Add picture itself must carry
// exactly the shipped overlay's own black-pixel count as transparent (alpha
// 0), not merely the composed canvas.
func TestReleaseTownSquareOverlayIsKeyedAtLoad(t *testing.T) {
	f := releaseFront(t)
	if f.TownSquareArt.Value() == nil || f.TownSquareArt.Value().Add == nil {
		t.Fatalf("production town square overlay did not resolve: %v", f.TownSquareArt.Err())
	}
	add := f.TownSquareArt.Value().Add
	ab := add.Bounds()
	transparent := 0
	for y := ab.Min.Y; y < ab.Max.Y; y++ {
		for x := ab.Min.X; x < ab.Max.X; x++ {
			_, _, _, a := add.At(x, y).RGBA()
			if a == 0 {
				transparent++
			}
		}
	}
	// THE EXACT COUNT, NOT A NONZERO ONE (round-3 review, W-1). `transparent
	// != 0` passes on a single keyed pixel, which is not the property. The
	// shipped overlay's own pure-black count is 20644 of 50784, and the two
	// preserved installs ship byte-identical art: both roots' town_add.bmp
	// hash to the same value, as do both townmain.bmp. A root whose overlay
	// carries a different count is a different install, and this test should
	// say so rather than pass.
	const shippedOverlayBlack = 20644
	if transparent != shippedOverlayBlack {
		t.Fatalf("loaded town-square overlay has %d transparent pixel(s) out of %d, want %d; "+
			"want its own pure-black pixels keyed out (LoadTownSquareArt, keyBlack)",
			transparent, ab.Dx()*ab.Dy(), shippedOverlayBlack)
	}
}

// TestReleaseTownSquareCompositesEveryPatchWhereTheInstallPutsIt is the
// placement half of the same witness, and it exists because the black-pixel
// test above does not cover placement (round-3 adversarial review, W-1).
//
// THE GAP IT CLOSES. Moving TownSquareAddOrigin from (0,0) to (60,100), or a
// label's own origin by any amount, puts a shipped patch visibly in the middle
// of the square. Before this test, both mutations left `go test ./...` green,
// check-scenarios green and check-release-tests green; only cmd/townsquarecheck
// caught them, and no gate runs that tool. A witness nothing executes is not a
// witness, and a placement no test asserts is not asserted.
//
// WHAT MAKES THE ASSERTION POSSIBLE. The overlay's 30140 opaque pixels are
// byte-identical to the base picture at its own offset, and its other 20644
// are keyed transparent. So a correct composite differs from the base
// picture in exactly one place: the three door labels.
//
// THE THREE EXCLUSION RECTS ARE LITERALS ON PURPOSE. Deriving them from
// townSquareLabelOrigin would move the exclusion with the value under test, and
// the test could not then see a label move at all. These are the label sizes and
// origins as measured from the shipped art: shop 52x76 at (264,264), tavern
// 28x64 at (144,332), school 140x116 at (436,300).
func TestReleaseTownSquareCompositesEveryPatchWhereTheInstallPutsIt(t *testing.T) {
	f := releaseFront(t)
	if f.TownSquareArt.Value() == nil || f.TownSquareArt.Value().Background == nil {
		t.Fatalf("production town square art did not resolve: %v", f.TownSquareArt.Err())
	}
	labels := [3]image.Rectangle{
		image.Rect(264, 264, 264+52, 264+76),
		image.Rect(144, 332, 144+28, 332+64),
		image.Rect(436, 300, 436+140, 300+116),
	}
	art := *f.TownSquareArt.Value()
	art.Exterior = nil // this witness isolates the overlay from later motion families
	composed := ui.ComposeTownSquare(ui.TownSquareView{Art: &art, Font: f.Font.Value(),
		Exterior: &ui.TownExteriorFrame{BirdOverlayVisible: true}})
	base := f.TownSquareArt.Value().Background
	bb := base.Bounds()
	differing, first := 0, image.Point{-1, -1}
	for y := bb.Min.Y; y < bb.Max.Y; y++ {
		for x := bb.Min.X; x < bb.Max.X; x++ {
			p := image.Pt(x, y)
			if p.In(labels[0]) || p.In(labels[1]) || p.In(labels[2]) {
				continue // a door label is the one thing that may differ
			}
			br, bg, bl, _ := base.At(x, y).RGBA()
			cr, cg, cl, _ := composed.At(x, y).RGBA()
			if br != cr || bg != cg || bl != cl {
				if differing == 0 {
					first = p
				}
				differing++
			}
		}
	}
	if differing != 0 {
		t.Fatalf("composed town square differs from the base picture in %d pixel(s) outside the "+
			"three door labels, first at (%d,%d), want 0 — a patch is composited somewhere the "+
			"install does not put it", differing, first.X, first.Y)
	}
}

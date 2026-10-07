package ui

import (
	"image"
	"image/color"
	"testing"
)

func townAmbientPatch(bounds image.Rectangle, c color.RGBA) *image.RGBA {
	p := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			p.SetRGBA(x, y, c)
		}
	}
	return p
}

func TestTownAmbientComposeUsesExplicitVisibilityAndExactLayerOrder(t *testing.T) {
	base := townAmbientPatch(image.Rect(0, 0, 640, 480), color.RGBA{A: 255})
	add := townAmbientPatch(image.Rect(0, 0, 2, 2), color.RGBA{B: 0xff, A: 0xff})
	bird := townAmbientPatch(image.Rect(0, 0, 2, 2), color.RGBA{R: 0xff, A: 0xff})
	// Tavern begins at (124,312); this deliberately wide patch reaches the
	// star's (340,288)..(404,332) rectangle so their relative order is tested.
	motion := townAmbientPatch(image.Rect(0, 0, 300, 40), color.RGBA{R: 0xff, A: 0xff})
	star := townAmbientPatch(image.Rect(0, 0, 64, 44), color.RGBA{G: 0xff, A: 0xff})
	art := &TownSquareArt{Background: base, Add: add, Exterior: &TownExteriorArt{
		Tavern: []image.Image{motion},
		Stars:  []image.Image{star},
	}}
	art.Exterior.Birds[0] = []image.Image{bird}

	// Zero is a real frame but a zero-valued descriptor is not visible.
	quiet := ComposeTownSquare(TownSquareView{Art: art, Exterior: &TownExteriorFrame{}})
	if got := quiet.RGBAAt(0, 0); got != (color.RGBA{A: 255}) {
		t.Fatalf("zero-valued bird/add painted %v", got)
	}
	if got := quiet.RGBAAt(340, 312); got != (color.RGBA{R: 0xff, A: 0xff}) {
		t.Fatalf("resting entrance motion changed %v", got)
	}

	frame := &TownExteriorFrame{
		Tavern:             0,
		BirdOverlayVisible: true,
		Star:               TownExteriorSpriteFrame{Frame: 0, Visible: true},
	}
	frame.Birds[0] = TownExteriorSpriteFrame{Family: 0, Frame: 0, Visible: true}
	pix := ComposeTownSquare(TownSquareView{Art: art, Exterior: frame})
	if got := pix.RGBAAt(0, 0); got != (color.RGBA{B: 0xff, A: 0xff}) {
		t.Fatalf("keyed overlay did not follow bird: %v", got)
	}
	if got := pix.RGBAAt(340, 312); got != (color.RGBA{G: 0xff, A: 0xff}) {
		t.Fatalf("star did not follow entrance motion: %v", got)
	}
}

package game

import (
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const townSquareArtPrefix = graphicsPrefix + "interface/town/"

// townSquareLabelSpec is one door label's shipped address and shipped size.
var townSquareLabelSpec = [3]struct {
	addr string
	w, h int
}{
	ui.TownSquareLabelShop:   {"shop_l.bmp", 52, 76},
	ui.TownSquareLabelTavern: {"tavern_l.bmp", 28, 64},
	ui.TownSquareLabelSchool: {"trener_l.bmp", 140, 116},
}

// LoadTownSquareArt resolves the town square's install-backed picture: the
// base scene, its overlay strip, the raster hit-test mask and the three door
// labels. It is cosmetic on LoadTownSchoolArt's own rule: a missing or
// mis-sized node carries its own address in the returned error, and the
// square falls back to the row-button layout it drew before this story
// rather than making the game unusable (NewFrontEnd carries the error beside
// a nil result).
func LoadTownSquareArt(src terrain.EntrySource) (*ui.TownSquareArt, error) {
	a := &ui.TownSquareArt{}
	var err error
	if a.Background, err = readChargenBMP(src, townSquareArtPrefix+"townmain.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.Background, 640, 480, townSquareArtPrefix+"townmain.bmp"); err != nil {
		return nil, err
	}
	overlay, err := readChargenBMP(src, townSquareArtPrefix+"town_add.bmp")
	if err != nil {
		return nil, err
	}
	if err = chargenSize(overlay, 552, 92, townSquareArtPrefix+"town_add.bmp"); err != nil {
		return nil, err
	}
	// KEYED, NOT OPAQUE (round-3 review). 20644 of the overlay's 50784 pixels
	// (40.7%) are pure black, the sky above the rooftops the overlay does not
	// cover; the base picture itself carries none (0/307200). Before this fix
	// ComposeTownSquare painted the overlay with draw.Src, putting every one of
	// those pixels down opaque as a black band. keyBlack is the shop's own rule
	// (shopart.go) for exactly this shape of art: pure black is the transparent
	// colour. The remaining 30140 opaque pixels are byte-identical to the base
	// at this offset (DIV-149), so a correctly keyed composite changes nothing
	// further — see ComposeTownSquare's own draw.Over.
	a.Add = keyBlack(overlay)
	if a.Mask, err = readChargenMask(src, townSquareArtPrefix+"townmask.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.Mask, 640, 480, townSquareArtPrefix+"townmask.bmp"); err != nil {
		return nil, err
	}
	if err = chargenMaskCodes(a.Mask, ui.TownSquareMaskCodes[:], townSquareArtPrefix+"townmask.bmp"); err != nil {
		return nil, err
	}
	for i, spec := range townSquareLabelSpec {
		addr := townSquareArtPrefix + spec.addr
		if a.Labels[i], err = readChargenBMP(src, addr); err != nil {
			return nil, err
		}
		if err = chargenSize(a.Labels[i], spec.w, spec.h, addr); err != nil {
			return nil, err
		}
	}
	a.Exterior, a.ExteriorProblems = loadTownExteriorArt(src)
	return a, nil
}

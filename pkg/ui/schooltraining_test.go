package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestSchoolTrainingOpaqueSidesDrawBetweenBackgroundAndSchoolOverlays(t *testing.T) {
	background := color.RGBA{R: 1, G: 2, B: 3, A: 0xff}
	mage := color.RGBA{R: 0xa1, A: 0xff}
	fighter := color.RGBA{B: 0xb2, A: 0xff}
	column := color.RGBA{G: 0xc3, A: 0xff}
	skill := color.RGBA{R: 0xd4, G: 0xd4, A: 0xff}
	diamond := color.RGBA{R: 0xe5, B: 0xe5, A: 0xff}
	art := &TownSchoolArt{Background: uniform(480, 480, background)}
	art.Column[0] = uniform(148, 208, column)
	art.Diamond[1] = uniform(80, 76, diamond)
	art.Skills[0][0][0] = uniform(20, 20, skill)
	mask := image.NewPaletted(image.Rect(0, 0, 92, 120), color.Palette{color.Black})
	for i := range mask.Pix {
		mask.Pix[i] = 0x37
	}
	art.Masks[0] = mask
	v := TownSurfaceView{
		Kind:                TownSurfaceSchool,
		SchoolArt:           art,
		SchoolClass:         0,
		SchoolColumnSet:     true,
		SchoolColumnFrame:   0,
		SchoolDiamondActive: true,
		SchoolDiamondFrame:  1,
		SchoolTraining:      SchoolTrainingFrame{Mage: uniform(172, 224, mage), Fighter: uniform(160, 224, fighter)},
		HoverCell:           -1,
		Cells:               []TownSurfaceCell{{Enabled: true, Selected: true}},
	}
	pix := ComposeTownSurface(v)
	if got := pix.RGBAAt(0, 200); got != mage {
		t.Fatalf("mage side = %v, want opaque training frame", got)
	}
	if got := pix.RGBAAt(479, 300); got != fighter {
		t.Fatalf("fighter side = %v, want opaque training frame", got)
	}
	// Mage reaches x171, while the existing column begins at x168. The
	// column must win their overlap because it is painted after m/tr.
	if got := pix.RGBAAt(168, 200); got != column {
		t.Fatalf("column overlap = %v, want column after training", got)
	}
	center := SchoolSkillRect(0, 0).Min.Add(image.Pt(SchoolSkillRect(0, 0).Dx()/2, SchoolSkillRect(0, 0).Dy()/2))
	if got := pix.RGBAAt(center.X, center.Y); got != skill {
		t.Fatalf("skill overlay = %v, want skill after column", got)
	}
	if got := pix.RGBAAt(200, 60); got != diamond {
		t.Fatalf("diamond = %v, want diamond after skills", got)
	}
	if got := pix.RGBAAt(319, 400); got != background {
		t.Fatalf("gap between sides = %v, want retained background", got)
	}
}

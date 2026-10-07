package ui

import (
	"image"
	"image/color"
	"testing"
)

// schoolIdleArt gives every slot a picture per art state in its own colour, so
// a frame names the state and slot it was drawn from.
func schoolIdleArt() (*TownSchoolArt, func(class, slot, state int) color.RGBA) {
	colour := func(class, slot, state int) color.RGBA {
		return color.RGBA{R: uint8(40 + 30*state), G: uint8(20 + 40*slot), B: uint8(30 + 100*class), A: 0xff}
	}
	art := &TownSchoolArt{Background: uniform(480, 480, color.RGBA{R: 0x11, A: 0xff})}
	for class := range art.Faces {
		art.Faces[class] = uniform(148, 208, color.RGBA{G: 0x70, A: 0xff})
	}
	for class := 0; class < 2; class++ {
		for slot := 0; slot < 5; slot++ {
			r := SchoolSkillRect(class, slot)
			for state := 0; state < 3; state++ {
				art.Skills[class][slot][state] = uniform(r.Dx(), r.Dy(), colour(class, slot, state))
			}
		}
	}
	return art, colour
}

func schoolIdleView(art *TownSchoolArt, class, slot int, selected, hover int) TownSurfaceView {
	cells := make([]TownSurfaceCell, 10)
	for i := range cells {
		cells[i].Enabled = i/5 == class
	}
	if selected >= 0 {
		cells[selected].Selected = true
	}
	return TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: class, Cells: cells,
		HoverCell: hover, SchoolIdleShine: true, SchoolIdleSlot: slot}
}

// TestSchoolIdleShineDrawsTheCyclingSlot is TOWN-500: with no skill hovered the
// slot at the cycle index draws its shine picture, the shine-on picture when it
// is the selected slot, and every other slot draws nothing. A hovered skill
// stops the idle icon, and a view without the idle flag draws none.
func TestSchoolIdleShineDrawsTheCyclingSlot(t *testing.T) {
	art, colour := schoolIdleArt()
	centre := func(class, slot int) image.Point {
		r := SchoolSkillRect(class, slot)
		return r.Min.Add(r.Size().Div(2))
	}
	for class := 0; class < 2; class++ {
		for slot := 0; slot < 5; slot++ {
			got := ComposeTownSurface(schoolIdleView(art, class, slot, -1, -1))
			if c := got.RGBAAt(centre(class, slot).X, centre(class, slot).Y); c != colour(class, slot, 1) {
				t.Fatalf("class %d slot %d idle = %+v, want its shine %+v", class, slot, c, colour(class, slot, 1))
			}
			for other := 0; other < 5; other++ {
				if other == slot {
					continue
				}
				p := centre(class, other)
				if c := got.RGBAAt(p.X, p.Y); c == colour(class, other, 1) || c == colour(class, other, 2) {
					t.Fatalf("class %d slot %d drew a shine while slot %d was cycling", class, other, slot)
				}
			}
			sel := ComposeTownSurface(schoolIdleView(art, class, slot, class*5+slot, -1))
			if c := sel.RGBAAt(centre(class, slot).X, centre(class, slot).Y); c != colour(class, slot, 2) {
				t.Fatalf("class %d selected slot %d idle = %+v, want its shine-on %+v", class, slot, c, colour(class, slot, 2))
			}
		}
	}
	hovered := ComposeTownSurface(schoolIdleView(art, 0, 2, -1, 4))
	if c := hovered.RGBAAt(centre(0, 2).X, centre(0, 2).Y); c == colour(0, 2, 1) {
		t.Fatal("the idle shine drew while a skill was hovered")
	}
	off := schoolIdleView(art, 0, 2, -1, -1)
	off.SchoolIdleShine = false
	if c := ComposeTownSurface(off).RGBAAt(centre(0, 2).X, centre(0, 2).Y); c == colour(0, 2, 1) {
		t.Fatal("a view without the idle flag drew the idle shine")
	}
}

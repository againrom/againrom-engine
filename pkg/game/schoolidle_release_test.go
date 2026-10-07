package game

import (
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/ui"
)

// The school's idle shine (TOWN-500) against the installed pictures. The
// expected pixels come from the shipped shine and shine-on bitmaps read at the
// claim's own paths and stored slot order, placed at the centre of the slot's
// rectangle and keyed on pure black, over the same frame drawn without the idle
// shine. The loss controls draw the neighbouring stored slot, the shine-on and
// the on picture and compare them with the screen.
func TestReleaseSchoolIdleShineDrawsTheStoredSlotsShinePicture(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil {
		t.Fatalf("production school art: %v", f.TownSchoolArt.Err())
	}
	now := time.Unix(1000, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.Carried = f.NextParty()
	f.Town.gold = 100000
	s := f.TownScreen().(*townScreen)
	stored := [2][5]string{
		{"sword", "axe", "pike", "club", "bow"},
		{"fire", "water", "earth", "air", "astral"},
	}
	dirs := [2]string{"fighter", "mage"}
	decode := func(class, c int, state string) *bmp.Image {
		raw, err := f.Archives.Containers.ReadFile(fmt.Sprintf("graphics/interface/training/column/%s/%s/%s.bmp", dirs[class], stored[class][c], state))
		if err != nil {
			t.Fatal(err)
		}
		pic, err := bmp.Decode(raw)
		if err != nil {
			t.Fatal(err)
		}
		return pic
	}
	// differences counts the pixels of rect in which the frame disagrees with
	// base with pic keyed over it, pic being centred in the slot's rectangle.
	differences := func(frame, base *image.RGBA, pic *bmp.Image, r image.Rectangle) int {
		at := image.Pt(r.Min.X+(r.Dx()-pic.Width)/2, r.Min.Y+(r.Dy()-pic.Height)/2)
		n := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				want := base.RGBAAt(x, y)
				if px, py := x-at.X, y-at.Y; px >= 0 && py >= 0 && px < pic.Width && py < pic.Height {
					if c := pic.At(px, py); c.R != 0 || c.G != 0 || c.B != 0 {
						want = color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
					}
				}
				if frame.RGBAAt(x, y) != want {
					n++
				}
			}
		}
		return n
	}
	for class := 0; class < 2; class++ {
		f.Carried[0].Mage = class == 1
		s.atSquare()
		s.Choose(2)
		for i := 0; s.room == roomTalk && i < 64; i++ {
			s.AdvanceTownDialogue()
		}
		if s.room != roomSchool {
			t.Fatal("production school entry failed")
		}
		s.CloseTip()
		now = now.Add(time.Hour)
		s.AdvanceTownSurfaceAnimation()
		for c := 0; c < 5; c++ {
			if c > 0 {
				now = now.Add(500 * time.Millisecond)
				s.AdvanceTownSurfaceAnimation()
			}
			v := s.TownSurface()
			if v.SchoolClass != class || !v.SchoolIdleShine {
				t.Fatalf("class %d stored slot %d: panel class %d idle %v", class, c, v.SchoolClass, v.SchoolIdleShine)
			}
			frame := ui.ComposeTownSurface(v)
			quiet := v
			quiet.SchoolIdleShine = false
			base := ui.ComposeTownSurface(quiet)
			r := ui.SchoolSkillRect(class, v.SchoolIdleSlot)
			if bad := differences(frame, base, decode(class, c, "shine"), r); bad != 0 {
				t.Fatalf("class %d stored slot %d (%s): %d pixels differ from its shine picture", class, c, stored[class][c], bad)
			}
			if differences(frame, base, decode(class, (c+1)%5, "shine"), r) == 0 {
				t.Errorf("class %d stored slot %d: the check cannot tell the slot from its neighbour's picture", class, c)
			}
			for _, wrong := range []string{"shine_on", "on"} {
				if differences(frame, base, decode(class, c, wrong), r) == 0 {
					t.Errorf("class %d stored slot %d: the check cannot tell shine from %s", class, c, wrong)
				}
			}
		}
	}
}

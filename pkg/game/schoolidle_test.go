package game

import (
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/ui"
)

func TestSchoolIdleEligibilityAndPostDrawClock(t *testing.T) {
	f, s := diamondSchool(t)
	at := time.Unix(1000, 0)
	now := at
	f.TownAnimationNow = func() time.Time { return now }
	for _, tc := range []struct {
		ms          int
		shown       bool
		slot, count int
	}{
		{0, false, 0, 0}, {499, false, 0, 0}, {500, true, 0, 0},
		{799, true, 0, 0}, {800, true, 0, 1}, {1100, true, 1, 1},
		{1101, true, 1, 2}, {5000, true, 2, 3},
	} {
		now = at.Add(time.Duration(tc.ms) * time.Millisecond)
		s.AdvanceTownSurfaceAnimation()
		v := s.TownSurface()
		if v.SchoolIdleShine != tc.shown || tc.shown && v.SchoolIdleSlot != tc.slot || s.schoolShine().Index != tc.count {
			t.Fatalf("at %dms: shown=%v slot=%d next=%d, want %v/%d/%d", tc.ms,
				v.SchoolIdleShine, v.SchoolIdleSlot, s.schoolShine().Index, tc.shown, tc.slot, tc.count)
		}
	}
}

func TestSchoolIdleHoverRemembersTheFollowingCounter(t *testing.T) {
	for _, mage := range []bool{false, true} {
		f, s := diamondSchool(t)
		f.Carried[0].Mage = mage
		s.schoolColumn().Rest(s.schoolMemberClass())
		now := time.Unix(1000, 0)
		f.TownAnimationNow = func() time.Time { return now }
		s.AdvanceTownSurfaceAnimation()
		now = now.Add(500 * time.Millisecond)
		cell := 2
		if mage {
			cell += 5
		}
		s.TownSurfaceHover(cell)
		s.AdvanceTownSurfaceAnimation()
		if s.TownSurface().SchoolIdleShine {
			t.Fatal("the hovered skill admitted an idle picture")
		}
		s.TownSurfaceHover(-1)
		now = now.Add(499 * time.Millisecond)
		s.AdvanceTownSurfaceAnimation()
		if s.TownSurface().SchoolIdleShine {
			t.Fatal("idle picture resumed before the 500ms eligibility delay")
		}
		now = now.Add(time.Millisecond)
		s.AdvanceTownSurfaceAnimation()
		want := 3
		if mage {
			want = 2
		}
		if v := s.TownSurface(); !v.SchoolIdleShine || v.SchoolIdleSlot != want {
			t.Fatalf("mage=%v: remembered hover resumed slot %d shown=%v, want %d", mage, v.SchoolIdleSlot, v.SchoolIdleShine, want)
		}
	}
}

func TestSchoolIdlePictureOrderMatchesEachClass(t *testing.T) {
	markers := map[string]uint8{"sword": 17, "axe": 31, "club": 43, "pike": 59, "bow": 71,
		"fire": 83, "water": 97, "earth": 109, "air": 127, "astral": 139}
	for class, tc := range []struct {
		dir   string
		names [5]string
		at    [5]image.Point
	}{
		{"fighter", [5]string{"sword", "axe", "club", "pike", "bow"}, [5]image.Point{{240, 212}, {240, 234}, {240, 262}, {240, 280}, {240, 298}}},
		{"mage", [5]string{"fire", "water", "earth", "air", "astral"}, [5]image.Point{{274, 246}, {204, 250}, {242, 285}, {238, 212}, {240, 249}}},
	} {
		f := shellFrontEnd()
		src := townSchoolSource()
		for _, name := range tc.names {
			src[townSchoolArtPrefix+"column/"+tc.dir+"/"+name+"/shine.bmp"] = synthBMP(20, 20, color.RGBA{R: markers[name], A: 255})
		}
		art, err := LoadTownSchoolArt(ROM1TownDescription(), src)
		if err != nil {
			t.Fatal(err)
		}
		f.TownSchoolArt = resolved(art, nil)
		f.Carried[0].Mage = class == 1
		s := f.townUI
		s.atSquare()
		s.Choose(2)
		now := time.Unix(1000, 0)
		f.TownAnimationNow = func() time.Time { return now }
		s.AdvanceTownSurfaceAnimation()
		now = now.Add(500 * time.Millisecond)
		for cycle, name := range tc.names {
			if cycle > 0 {
				now = now.Add(301 * time.Millisecond)
			}
			s.AdvanceTownSurfaceAnimation()
			frame := ui.ComposeTownSurface(s.TownSurface())
			at := tc.at[cycle]
			if got := frame.RGBAAt(at.X, at.Y).R; got != markers[name] {
				t.Fatalf("%s cycle %d: picture marker %d at %v, want %s marker %d", tc.dir, cycle, got, at, name, markers[name])
			}
		}
	}
}

// TestSchoolViewCarriesTheIdleSlotInTheSharedOrder: the view names the slot the
// idle clock stands on in the shared slot order. The fighter panel's shine
// visits sword, axe, club, pike, bow in the order drawn, and carries none
// before idle processing becomes eligible.
func TestSchoolViewCarriesTheIdleSlotInTheSharedOrder(t *testing.T) {
	f := shellFrontEnd()
	schoolArt, err := LoadTownSchoolArt(ROM1TownDescription(), townSchoolSource())
	if err != nil {
		t.Fatal(err)
	}
	f.TownSchoolArt = resolved(schoolArt, nil)
	now := time.Unix(1000, 0)
	f.TownAnimationNow = func() time.Time { return now }
	s := f.townUI
	s.atSquare()
	s.Choose(2)
	if v := s.TownSurface(); v.SchoolIdleShine {
		t.Fatal("the idle shine showed before any school paint")
	}
	s.AdvanceTownSurfaceAnimation()
	now = now.Add(500 * time.Millisecond)
	s.AdvanceTownSurfaceAnimation()
	want := []int{0, 1, 2, 3, 4, 0}
	for cycle, slot := range want {
		if cycle > 0 {
			now = now.Add(301 * time.Millisecond)
			s.AdvanceTownSurfaceAnimation()
		}
		if v := s.TownSurface(); !v.SchoolIdleShine || v.SchoolIdleSlot != slot {
			t.Fatalf("cycle %d: view idle = %v slot %d, want slot %d", cycle, v.SchoolIdleShine, v.SchoolIdleSlot, slot)
		}
	}
}

// TestSchoolShineDisplaySlotKeepsTheMageStoredOrder: only the fighter panel
// follows the drawn order; the mage panel keeps the stored order.
func TestSchoolShineDisplaySlotKeepsTheMageStoredOrder(t *testing.T) {
	for cycle, want := range []int{0, 1, 2, 3, 4} {
		s := &townScreen{}
		s.schoolShine().Index = cycle
		if got := s.schoolShine().Slot(schoolFighterClass); got != want {
			t.Errorf("fighter cycle %d: slot %d, want %d", cycle, got, want)
		}
	}
	for cycle, want := range []int{0, 1, 3, 2, 4} {
		s := &townScreen{}
		s.schoolShine().Index = cycle
		if got := s.schoolShine().Slot(schoolMageClass); got != want {
			t.Errorf("mage cycle %d: slot %d, want %d", cycle, got, want)
		}
	}
}

package ui

import (
	"image"
	"testing"
)

// fakeLatchShopTown is a shop with live command buttons.
type fakeLatchShopTown struct {
	fakeTipShopCrossingTown
}

func (f *fakeLatchShopTown) ShopScreen() ShopScreenView {
	v := f.fakeTipShopCrossingTown.ShopScreen()
	v.Live = [4]bool{true, true, true, true}
	return v
}

// townLatchSite is a former town latch; count is its activations.
// focusKeeps: the shop buttons' press survives a lost focus, as before.
type townLatchSite struct {
	app             *App
	inside, outside image.Point
	count           func() int
	focusKeeps      bool
}

func (s townLatchSite) pointer(t *testing.T, action string, p image.Point) {
	t.Helper()
	if err := s.app.HeadlessPointer(action, p.X, p.Y); err != nil {
		t.Fatal(err)
	}
}

func latchTownApp(t *testing.T, town TownScreen) *App {
	t.Helper()
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the town fixture")
	}
	return a
}

var townLatchSites = []struct {
	name string
	open func(t *testing.T) townLatchSite
}{
	{"town surface button", func(t *testing.T) townLatchSite {
		buttons := []TownSurfaceButton{{Enabled: true}, {Enabled: true}}
		town := &fakeTipSurfaceEnumTown{view: TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: -1, Buttons: buttons}}
		a := latchTownApp(t, town)
		inside, _ := sampleInside(TownSurfaceButtonRect(TownSurfaceSchool, 0))
		outside := image.Pt(240, 20)
		if _, hit := TownSurfaceControlAt(town.view, outside); hit {
			t.Fatal("fixture: the outside point hits a control")
		}
		return townLatchSite{app: a, inside: inside, outside: outside, count: func() int {
			n := 0
			for _, c := range town.surfaceClicks {
				if c.Kind == TownSurfaceControlButton {
					n++
				}
			}
			return n
		}}
	}},
	{"room tip close button", func(t *testing.T) townLatchSite {
		tip := TipPanelView{Rect: SchoolTipRect, Text: "the school teaches skills", Art: tipTestArt(), Font: shopTipTestFont()}
		town := &fakeTipSurfaceEnumTown{view: TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: -1, Tip: tip}}
		a := latchTownApp(t, town)
		inside, _ := sampleInside(TipPanelCloseRect(tip.Rect))
		return townLatchSite{app: a, inside: inside, outside: image.Pt(600, 300), count: func() int { return town.closed }}
	}},
	{"shop command button", func(t *testing.T) townLatchSite {
		town := &fakeLatchShopTown{fakeTipShopCrossingTown{mask: shopTestMask()}}
		a := latchTownApp(t, town)
		inside, _ := sampleInside(shopButtonRects[1])
		return townLatchSite{app: a, inside: inside, outside: image.Pt(320, 2), focusKeeps: true, count: func() int {
			n := 0
			for _, c := range town.clicked {
				if c.Kind == ShopControlButton {
					n++
				}
			}
			return n
		}}
	}},
}

// TestTownLatchesRunTheKitCases drives each former town latch by App input.
func TestTownLatchesRunTheKitCases(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T, s townLatchSite) (int, int)
	}{
		{"press and release inside", func(t *testing.T, s townLatchSite) (int, int) {
			s.pointer(t, "press", s.inside)
			s.pointer(t, "release", s.inside)
			return s.count(), 1
		}},
		{"press inside, release outside", func(t *testing.T, s townLatchSite) (int, int) {
			s.pointer(t, "press", s.inside)
			s.pointer(t, "move", s.outside)
			s.pointer(t, "release", s.outside)
			s.pointer(t, "release", s.inside)
			return s.count(), 0
		}},
		{"press inside, moved out and back", func(t *testing.T, s townLatchSite) (int, int) {
			s.pointer(t, "press", s.inside)
			s.pointer(t, "move", s.outside)
			s.pointer(t, "move", s.inside)
			s.pointer(t, "release", s.inside)
			return s.count(), 1
		}},
		{"press inside, focus lost", func(t *testing.T, s townLatchSite) (int, int) {
			s.pointer(t, "press", s.inside)
			if err := s.app.HeadlessFocus(false); err != nil {
				t.Fatal(err)
			}
			if err := s.app.HeadlessFocus(true); err != nil {
				t.Fatal(err)
			}
			s.pointer(t, "release", s.inside)
			return s.count(), boolCount(s.focusKeeps)
		}},
	}
	for _, site := range townLatchSites {
		t.Run(site.name, func(t *testing.T) {
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					s := site.open(t)
					if got := s.count(); got != 0 {
						t.Fatalf("fixture activated %d times before any gesture", got)
					}
					if got, want := c.run(t, s); got != want {
						t.Fatalf("activated %d times, want %d", got, want)
					}
				})
			}
		})
	}
}

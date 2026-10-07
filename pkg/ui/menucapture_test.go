package ui

import (
	"fmt"
	"image"
	"testing"
	"time"
)

// While the Esc menu is up its panel is the root's capture object: every mouse
// message goes to it first, and a click outside the panel reaches no other
// child of the root (MENU-INPUT-016). Nothing behind the panel answers, on the
// map or in the town. Each case below runs the same gestures with the menu
// closed first, so a fixture that never answered cannot pass for one that was
// silenced.

// menuOutsideFrame is the frame pixels of a 640x480 town frame that lie outside
// the town's Esc panel, on a grid of the given stride.
func menuOutsideFrame(stride int) []image.Point {
	panel := gameMenuPanelRect(gameMenuTown)
	var out []image.Point
	for y := stride / 2; y < 480; y += stride {
		for x := stride / 2; x < 640; x += stride {
			if p := image.Pt(x, y); !p.In(panel) {
				out = append(out, p)
			}
		}
	}
	return out
}

// menuSweep presses and releases each button over every point: the primary
// button and then the secondary one.
func menuSweep(a *App, points []image.Point, now time.Time) {
	for _, p := range points {
		a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
		a.step(appInput{CursorX: p.X, CursorY: p.Y, SecondaryPressed: true}, now)
		a.step(appInput{CursorX: p.X, CursorY: p.Y, SecondaryReleased: true}, now)
	}
}

// TestAClickOutsideTheEscMenuReachesNothingInTheTown sweeps presses and
// releases of both buttons over every point of the town frame outside the
// panel. No picker row, no surface control and no Back is reached, and the menu
// is still up afterwards; with the menu closed the same sweep reaches the
// surface's controls.
func TestAClickOutsideTheEscMenuReachesNothingInTheTown(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	points := menuOutsideFrame(5)
	fresh := func() (*App, *fakeStatsSurfaceTown) {
		town := &fakeStatsSurfaceTown{}
		a := newTestApp(t, appRows(3), okLoader(t))
		a.SetTown(town)
		if !a.flow.showTown("") {
			t.Fatal("showTown refused the surface")
		}
		return a, town
	}

	a, town := fresh()
	menuSweep(a, points, now)
	if len(town.surfaceClicks) == 0 {
		t.Fatal("with the menu closed the sweep reached no surface control, so the gated sweep below proves nothing")
	}

	a, town = fresh()
	a.step(appInput{Escape: true}, now)
	if a.Screen() != ScreenGameMenu {
		t.Fatalf("Escape on the town square landed on %v, want the menu", a.Screen())
	}
	backs := town.backs
	menuSweep(a, points, now)
	if a.Screen() != ScreenGameMenu {
		t.Errorf("a click outside the panel left screen %v, want the menu still up", a.Screen())
	}
	if len(town.surfaceClicks) != 0 || len(town.chosen) != 0 || town.backs != backs {
		t.Errorf("a click outside the panel reached the town: surface clicks %v, rows chosen %v, %d extra Back calls",
			town.surfaceClicks, town.chosen, town.backs-backs)
	}
	if a.flow.msg != "" {
		t.Errorf("a click outside the panel posted %q", a.flow.msg)
	}
}

// menuMapSpot is a cell of the popup fixture's map with a free neighbour on all
// eight sides. Every one of the nine lies on drawn ground outside the mission
// panel, clear of the edge-scroll margin and of every piece of HUD furniture,
// and none is one of the fixture's own units.
func menuMapSpot(t *testing.T, f *popupFix) image.Point {
	t.Helper()
	panel := gameMenuPanelRect(gameMenuMission)
	win := f.v.place.WindowSize()
	free := func(col, row int) bool {
		for _, u := range popupEntities() {
			if u.Cell == image.Pt(col, row) {
				return false
			}
		}
		x, y := popupAt(f.v, col, row)
		if x < 2*EdgeMargin || y < 2*EdgeMargin || x > win.X-2*EdgeMargin || y > win.Y-2*EdgeMargin {
			return false
		}
		if p, ok := f.a.windowToNativeFrame(x, y); !ok || p.In(panel) {
			return false
		}
		if f.v.commandPanelCaptures(x, y) || f.v.inventoryCaptures(x, y) || f.v.spellbookCaptures(x, y) ||
			f.v.minimapCaptures(x, y) || f.v.panelCaptures(x, y) {
			return false
		}
		c, r, inside := f.v.groundCellAt(float64(x), float64(y))
		return inside && c == col && r == row
	}
	for row := 1; row < 15; row++ {
		for col := 1; col < 19; col++ {
			ok := true
			for dr := -1; dr <= 1 && ok; dr++ {
				for dc := -1; dc <= 1 && ok; dc++ {
					ok = free(col+dc, row+dr)
				}
			}
			if ok {
				return image.Pt(col, row)
			}
		}
	}
	t.Fatal("the fixture's map has no cell with a free ring outside the mission panel and the HUD")
	return image.Point{}
}

// menuMapMoment is what a gesture on the map could change: the selection, the
// three seams into the world, the armed attack and the camera.
type menuMapMoment struct {
	marked                 string
	orders, attacks, blows int
	armed                  bool
	camera                 [3]float64
}

func readMenuMapMoment(f *popupFix) menuMapMoment {
	c := f.v.Camera()
	return menuMapMoment{fmt.Sprint(f.v.marked()), len(f.s.orders), len(f.s.attacks), len(f.s.blows), f.v.AttackArmed(), [3]float64{c.X, c.Y, c.Zoom}}
}

// TestAClickOutsideTheEscMenuReachesNoOrderOnTheMap runs each map gesture from a
// selected unit, outside the mission panel: a tap on a unit, a selection
// rectangle round it, a tap on ground, a secondary click, the arming key and the
// cursor held in the edge margin. Under the menu none of them changes the
// selection, the world or the camera; with no menu each of them does.
func TestAClickOutsideTheEscMenuReachesNoOrderOnTheMap(t *testing.T) {
	const unitC = 77
	tap := func(f *popupFix, at image.Point) {
		in := f.at(at.X, at.Y)
		in.PrimaryPressed, in.PrimaryReleased = true, true
		f.frame(in)
	}
	gestures := []struct {
		name string
		run  func(f *popupFix, spot image.Point)
	}{
		{"tap on a unit", func(f *popupFix, spot image.Point) { tap(f, spot) }},
		{"selection rectangle", func(f *popupFix, spot image.Point) {
			from, to := spot.Sub(image.Pt(1, 1)), spot.Add(image.Pt(1, 1))
			press := f.at(from.X, from.Y)
			press.PrimaryPressed = true
			press.Viewer.PrimaryDown = true
			f.frame(press)
			drag := f.at(to.X, to.Y)
			drag.Viewer.PrimaryDown = true
			f.frame(drag)
			release := f.at(to.X, to.Y)
			release.PrimaryReleased = true
			f.frame(release)
		}},
		{"tap on ground", func(f *popupFix, spot image.Point) { tap(f, spot.Add(image.Pt(1, 0))) }},
		{"secondary click", func(f *popupFix, spot image.Point) {
			in := f.at(spot.X+1, spot.Y)
			in.SecondaryPressed, in.SecondaryReleased = true, true
			f.frame(in)
		}},
		{"arming key", func(f *popupFix, spot image.Point) {
			in := haltNeutral()
			in.Attack = true
			f.frame(in)
		}},
		{"cursor in the edge margin", func(f *popupFix, spot image.Point) {
			win := f.v.place.WindowSize()
			for n := 0; n < 20; n++ {
				in := haltNeutral()
				in.CursorX, in.CursorY = win.X-1, win.Y-1
				in.Viewer.CursorX, in.Viewer.CursorY = win.X-1, win.Y-1
				f.frame(in)
			}
		}},
	}
	setup := func(t *testing.T) (*popupFix, image.Point) {
		f := newPopupFix(t, haltOpts{})
		spot := menuMapSpot(t, f)
		f.v.SetEntities(append(popupEntities(), MapEntity{ID: unitC, Cell: spot}))
		f.selectA()
		f.s.orders = nil
		return f, spot
	}
	for _, g := range gestures {
		t.Run("control, no menu/"+g.name, func(t *testing.T) {
			f, spot := setup(t)
			before := readMenuMapMoment(f)
			g.run(f, spot)
			if readMenuMapMoment(f) == before {
				t.Fatalf("with no menu the %s changed nothing, so the gated case proves nothing", g.name)
			}
		})
		t.Run("the menu up/"+g.name, func(t *testing.T) {
			f, spot := setup(t)
			open := haltNeutral()
			open.Escape = true
			f.frame(open)
			if f.a.Screen() != ScreenGameMenu || !f.a.flow.popupOpen() {
				t.Fatalf("Escape on the map landed on %v, want the menu up over it", f.a.Screen())
			}
			before := readMenuMapMoment(f)
			g.run(f, spot)
			if got := readMenuMapMoment(f); got != before {
				t.Errorf("the %s outside the panel changed %+v to %+v", g.name, before, got)
			}
			if f.a.Screen() != ScreenGameMenu {
				t.Errorf("the %s outside the panel left screen %v, want the menu still up", g.name, f.a.Screen())
			}
			if f.a.flow.msg != "" {
				t.Errorf("the %s outside the panel posted %q", g.name, f.a.flow.msg)
			}
		})
	}
}

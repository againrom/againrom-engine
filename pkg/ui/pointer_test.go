package ui

// Attack pointer presentation and the press's own target hit test.

import (
	"image"
	"testing"
)

// ptPic is a picture standing in for the game's own art. Its content is never
// read by anything under test — what matters is that it is not nil — so one
// pixel is enough and no fixture has to carry a sprite.
func ptPic() *image.RGBA { return image.NewRGBA(image.Rect(0, 0, 1, 1)) }

// ptHover puts the cursor at a cell through the shipped path and returns the
// window point, so the viewer's own remembered cursor is the one the methods
// below read.
func ptHover(a *App, v *Viewer, col, row int) (int, int) {
	x, y := cellPoint(v, col, row)
	a.step(atFrame(x, y), atAt)
	return x, y
}

// TestThePointerIsDrawnExactlyWhileTheModeIsUp is AC-5 and AC-12.
func TestThePointerIsDrawnExactlyWhileTheModeIsUp(t *testing.T) {
	t.Run("mode down draws nothing", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		v.SetAttackPointer(ptPic())
		ptHover(a, v, atEmptyCol, atEmptyRow)
		if _, _, ok := v.attackPointerPresent(); ok {
			t.Error("a pointer is drawn with the mode down")
		}
	})

	t.Run("mode up with a picture draws that picture at the cursor less its hotspot", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		pic := ptPic()
		v.SetAttackPointer(pic)
		x, y := ptHover(a, v, atEmptyCol, atEmptyRow)
		a.step(afHeld(x, y), atAt)

		got, at, ok := v.attackPointerPresent()
		if !ok {
			t.Fatal("no pointer with the mode up")
		}
		if got != pic {
			t.Errorf("the pointer is %v, want the picture that was set", got)
		}
		want := image.Pt(x-AttackPointerHotspot.X, y-AttackPointerHotspot.Y)
		if at != want {
			t.Errorf("the pointer goes at %v, want the cursor less the hotspot %v", at, want)
		}
	})

	t.Run("mode up with no picture is the authored mark", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		x, y := ptHover(a, v, atEmptyCol, atEmptyRow)
		a.step(afHeld(x, y), atAt)

		got, _, ok := v.attackPointerPresent()
		if !ok {
			t.Fatal("no pointer with the mode up and no picture — the mode would be invisible")
		}
		if got != nil {
			t.Errorf("the pointer is %v, want nil so the authored mark is drawn", got)
		}
	})

	t.Run("a viewer that has never seen a cursor draws no pointer", func(t *testing.T) {
		_, v, _ := atOnMap(t)
		v.SetAttackPointer(ptPic())
		v.hasCursor = false
		v.attackHeld = true
		if _, _, ok := v.attackPointerPresent(); ok {
			t.Error("a pointer is drawn at a cursor nobody has moved")
		}
	})

	// The developer viewer has no route to the mode at all: both writers are
	// unexported and its own Update never reaches either. This asserts the frame
	// that follows from that.
	t.Run("the standalone developer viewer draws no attack pointer", func(t *testing.T) {
		v := newViewer(t, grid(60, 60))
		v.SetEntities(atEntities())
		v.step(Input{CursorX: 10, CursorY: 10}, atAt)
		if _, _, ok := v.attackPointerPresent(); ok {
			t.Error("the developer viewer drew an attack pointer")
		}
	})
}

// TestTheSystemPointerIsHiddenExactlyWhenOursIsDrawn is AC-6 — the risk this
// story's own change creates, closed by one bool driving both.
func TestTheSystemPointerIsHiddenExactlyWhenOursIsDrawn(t *testing.T) {
	a, v, _ := atOnMap(t)
	x, y := ptHover(a, v, atEmptyCol, atEmptyRow)

	// Down: nothing wanted, nothing to tell the engine.
	_, _, shown := v.attackPointerPresent()
	if v.pointerModeChange(shown) {
		t.Fatal("the engine is told to show a pointer it is already showing")
	}
	if v.pointerHidden {
		t.Fatal("the system pointer is hidden with the mode down")
	}

	// Up: the wanted state follows the pointer's own bool, and the engine is
	// told exactly once.
	a.step(afHeld(x, y), atAt)
	_, _, shown = v.attackPointerPresent()
	if !shown {
		t.Fatal("premise: no pointer with the mode up")
	}
	if !v.pointerModeChange(shown) {
		t.Fatal("the engine was not told to hide the system pointer")
	}
	if !v.pointerHidden {
		t.Fatal("the system pointer is not hidden while ours is drawn")
	}
	if v.pointerModeChange(shown) {
		t.Error("the engine is told again on a frame that changed nothing")
	}

	// Down again: told once more, the other way.
	a.step(atFrame(x, y), atAt)
	_, _, shown = v.attackPointerPresent()
	if shown {
		t.Fatal("premise: a pointer survives the modifier's release")
	}
	if !v.pointerModeChange(shown) {
		t.Fatal("the engine was not told to show the system pointer again")
	}
	if v.pointerHidden {
		t.Error("the system pointer is still hidden with the mode down")
	}
}

func TestAttackTargetHitNamesTheUnitAPressWouldName(t *testing.T) {
	for _, tc := range []struct {
		name     string
		col, row int
		want     uint32
		hit      bool
	}{
		{"enemy", atFoeCol, atFoeRow, atFoeID, true},
		{"corpse", atDeadCol, atDeadRow, 0, false},
		{"empty ground", atEmptyCol, atEmptyRow, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v, _ := atOnMap(t)
			x, y := cellPoint(v, tc.col, tc.row)
			a.step(afHeld(x, y), atAt)
			id, hit := targetAt(v.entities, v.entityPickRect, float64(x), float64(y), false)
			if hit != tc.hit || hit && id != tc.want {
				t.Fatalf("target hit = %d/%v, want %d/%v", id, hit, tc.want, tc.hit)
			}
		})
	}
	t.Run("mode down", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		ptHover(a, v, atFoeCol, atFoeRow)
		if _, _, shown := v.attackPointerPresent(); shown {
			t.Fatal("attack pointer shown with the mode down")
		}
	})
	t.Run("overlapping rectangles choose the lower id", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		v.SetEntities([]MapEntity{
			{ID: atHiID, Cell: image.Pt(atFoeCol, atFoeRow), Life: LifeAlive, HP: 100, MaxHP: 100},
			{ID: atLoID, Cell: image.Pt(atFoeCol, atFoeRow), Life: LifeAlive, HP: 100, MaxHP: 100},
		})
		x, y := cellPoint(v, atFoeCol, atFoeRow)
		a.step(afHeld(x, y), atAt)
		if id, hit := targetAt(v.entities, v.entityPickRect, float64(x), float64(y), false); !hit || id != atLoID {
			t.Fatalf("target hit = %d/%v, want lower id %d", id, hit, atLoID)
		}
	})
}

func TestAttackTargetPickRectangleStaysUnclippedAtTheWorldViewport(t *testing.T) {
	a, v, _ := atOnMap(t)
	layoutViewport(v, 200, 200)
	const edgeCol, edgeRow = 6, 2
	v.SetEntities([]MapEntity{{ID: atLoID, Cell: image.Pt(edgeCol, edgeRow), Life: LifeAlive, HP: 100, MaxHP: 100}})
	unclipped, ok := v.entityPickRect(v.entities[0])
	if !ok || unclipped.X+unclipped.W <= float64(v.cam.ViewW) || unclipped.X >= float64(v.cam.ViewW) {
		t.Fatalf("pick rectangle %+v/%v does not straddle viewport edge %d", unclipped, ok, v.cam.ViewW)
	}
	x, y := int(unclipped.X)+2, int(unclipped.Y)+int(unclipped.H)/2
	if x >= v.cam.ViewW {
		t.Fatal("fixture point lies outside the map surface")
	}
	a.step(afHeld(x, y), atAt)
	if id, hit := targetAt(v.entities, v.entityPickRect, float64(x), float64(y), false); !hit || id != atLoID {
		t.Fatalf("on-map target hit = %d/%v, want %d", id, hit, atLoID)
	}
	x = v.cam.ViewW + 1
	if id, hit := targetAt(v.entities, v.entityPickRect, float64(x), float64(y), false); !hit || id != atLoID {
		t.Fatalf("unclipped target hit = %d/%v, want %d", id, hit, atLoID)
	}
	if name := v.gestureCursorAt(x, y); name != "" {
		t.Fatalf("off-map gesture cursor = %q, want no arm", name)
	}
}

// A popup suppresses the attack pointer even before the next input step.
//
// The mode is already lowered by the time a popup is up, so this asserts the
// SECOND gate: what a frame composed without a step having run first would draw.
// The fields are set directly for that reason — the case is precisely the one no
// dispatch produces.
func TestAPopupDrawsNoAttackPointer(t *testing.T) {
	f := newPopupFix(t, haltOpts{})
	f.frame(f.at(popupACol, popupARow))
	f.v.SetAttackPointer(ptPic())

	f.v.attackHeld = true
	if _, _, ok := f.v.attackPointerPresent(); !ok {
		t.Fatal("premise: no pointer with the mode up and no popup")
	}

	f.open()
	if _, _, ok := f.v.attackPointerPresent(); ok {
		t.Error("an attack pointer is drawn under a popup")
	}
}

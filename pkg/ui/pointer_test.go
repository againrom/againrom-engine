package ui

// What the frame draws for the attack mode: the pointer, the system cursor it
// replaces, and the marker over the unit a press would name.
//
// Every case reads the two pure methods that DECIDE, never a drawn frame. That
// is deliberate and it is what this story's shape is for: the decisions were put
// above the draw so they could be asserted with no window, and asserting the
// draw instead would witness ebitengine rather than the contract.

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
	t.Run("the standalone developer viewer draws neither", func(t *testing.T) {
		v := newViewer(t, grid(60, 60))
		v.SetEntities(atEntities())
		v.step(Input{CursorX: 10, CursorY: 10}, atAt)
		if _, _, ok := v.attackPointerPresent(); ok {
			t.Error("the developer viewer drew an attack pointer")
		}
		if _, ok := v.attackTargetRect(); ok {
			t.Error("the developer viewer drew a target marker")
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

// TestTheMarkedUnitIsTheUnitAPressWouldName is AC-8.
func TestTheMarkedUnitIsTheUnitAPressWouldName(t *testing.T) {
	arm := func(t *testing.T, col, row int) (*Viewer, int, int) {
		t.Helper()
		a, v, _ := atOnMap(t)
		x, y := cellPoint(v, col, row)
		a.step(afHeld(x, y), atAt)
		return v, x, y
	}

	t.Run("over a unit, the unit's own rectangle", func(t *testing.T) {
		v, x, y := arm(t, atFoeCol, atFoeRow)
		got, ok := v.attackTargetRect()
		if !ok {
			t.Fatal("nothing marked with the cursor over a drawn unit")
		}
		// The marked rectangle is the pick rectangle of the very id the press
		// would name — asserted through the press's own hit test, not through a
		// remembered geometry.
		id, hit := targetAt(v.entities, v.entityPickRect, float64(x), float64(y), false)
		if !hit {
			t.Fatal("premise: the press names nothing where the marker marks something")
		}
		var want screenRect
		for _, e := range v.entities {
			if e.ID == id {
				want, _ = v.entityPickRect(e)
			}
		}
		if got != want {
			t.Errorf("marked %v, want %v — the rectangle of the id a press names", got, want)
		}
		if id != atFoeID {
			t.Errorf("the press names %d, want %d", id, atFoeID)
		}
	})

	t.Run("over a corpse, nothing", func(t *testing.T) {
		v, _, _ := arm(t, atDeadCol, atDeadRow)
		if _, ok := v.attackTargetRect(); ok {
			t.Error("a corpse was marked; a press there falls through to the move arm")
		}
	})

	t.Run("over empty ground, nothing", func(t *testing.T) {
		v, _, _ := arm(t, atEmptyCol, atEmptyRow)
		if _, ok := v.attackTargetRect(); ok {
			t.Error("empty ground was marked")
		}
	})

	t.Run("with the mode down, nothing", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		ptHover(a, v, atFoeCol, atFoeRow)
		if _, ok := v.attackTargetRect(); ok {
			t.Error("a unit was marked with the mode down")
		}
	})

	// Two units on ONE cell: their pick rectangles are identical, so the point
	// is held by both and the tie can only be settled over ids. The marker must
	// settle it the way the press does.
	t.Run("where two rectangles hold the point, the lower id", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		v.SetEntities([]MapEntity{
			{ID: atHiID, Cell: image.Pt(atFoeCol, atFoeRow), Life: LifeAlive, HP: 100, MaxHP: 100},
			{ID: atLoID, Cell: image.Pt(atFoeCol, atFoeRow), Life: LifeAlive, HP: 100, MaxHP: 100},
		})
		x, y := cellPoint(v, atFoeCol, atFoeRow)
		a.step(afHeld(x, y), atAt)

		got, ok := v.attackTargetRect()
		if !ok {
			t.Fatal("nothing marked where two units stand")
		}
		id, _ := targetAt(v.entities, v.entityPickRect, float64(x), float64(y), false)
		if id != atLoID {
			t.Fatalf("premise: the press names %d, want the lower id %d", id, atLoID)
		}
		var want screenRect
		for _, e := range v.entities {
			if e.ID == atLoID {
				want, _ = v.entityPickRect(e)
			}
		}
		if got != want {
			t.Errorf("marked %v, want the lower id's rectangle %v", got, want)
		}
	})
}

// TestTheAttackTargetMarkerIsClippedToTheWorldViewport is adversarial pass 2's
// F2: the marker annotates a unit standing on the map, so it must not stroke
// into the right column, even for a unit whose PICK rectangle — the one the
// press's own hit test reads — legitimately crosses the boundary and must
// stay unclipped so the unit stays pickable.
//
// MUTATION THIS FAILS AGAINST: attackTargetRect returning entityPickRect's
// result directly, with no call to clipScreenRectToViewport.
//
// THE ASSERTION ANSWERS FOR THE PAINTED SPAN, NOT THE RECT (adversarial pass
// 3, F2): Ebiten's vector.StrokeRect centres a strokeWidth-wide line on the
// rect's own edge rather than drawing inside it, so a rect clipped flush to
// ViewW still paints AttackMarkerWidth/2 pixels past it. The rect-only
// assertion below is kept for the pick-rectangle premise it also carries,
// but it cannot see this: clipScreenRectToViewport always returns a rect
// flush to ViewW regardless of stroke width, so the rect check is green
// whether or not the paint bleeds. The painted-span assertion computes its
// bound from Ebiten's own centring rule directly, not from
// clipScreenRectToViewport's formula, and reads AttackMarkerWidth only as
// the shipped fact of what is drawn.
//
// MUTATION THE PAINTED-SPAN ASSERTION FAILS AGAINST, WHEN THE OLD
// (PASS-2) CLIP IS RESTORED: clipScreenRectToViewport returning a rect
// flush to the viewport bound with no stroke-width inset. Verified by hand
// against a checked-out copy of that version: got.X+got.W == 200 exactly,
// so paintedRight == 200+AttackMarkerWidth/2 > 200, red at the current
// AttackMarkerWidth (2) already and reds harder if AttackMarkerWidth is
// raised to 4, which the rect-only assertion above cannot do at any width.
func TestTheAttackTargetMarkerIsClippedToTheWorldViewport(t *testing.T) {
	a, v, _ := atOnMap(t)
	layoutViewport(v, 200, 200)

	const edgeCol, edgeRow = 6, 2 // native zoom, 32px cells: footprint ~[192,224), straddling x=200
	v.SetEntities([]MapEntity{{ID: atLoID, Cell: image.Pt(edgeCol, edgeRow), Life: LifeAlive, HP: 100, MaxHP: 100}})

	unclipped, ok := v.entityPickRect(v.entities[0])
	if !ok {
		t.Fatal("premise: the entity has no pick rectangle")
	}
	if unclipped.X+unclipped.W <= float64(v.cam.ViewW) {
		t.Fatalf("premise: the unclipped pick rectangle %+v does not cross the viewport's right edge at %d", unclipped, v.cam.ViewW)
	}
	if unclipped.X >= float64(v.cam.ViewW) {
		t.Fatalf("premise: the unclipped pick rectangle %+v starts past the viewport's right edge at %d, leaves no on-map point inside it", unclipped, v.cam.ViewW)
	}

	// x sits two pixels into the rect from its left edge, which is inside
	// the straddling rect and, by the premise just checked, still short of
	// ViewW.
	x, y := int(unclipped.X)+2, int(unclipped.Y)+int(unclipped.H)/2
	if x >= v.cam.ViewW {
		t.Fatalf("premise: chosen cursor x=%d is not on the map surface (ViewW=%d)", x, v.cam.ViewW)
	}
	a.step(afHeld(x, y), atAt)

	got, ok := v.attackTargetRect()
	if !ok {
		t.Fatal("nothing marked over the unit")
	}
	if got.X+got.W > float64(v.cam.ViewW) {
		t.Errorf("marker rect %+v extends past the world viewport's right edge at %d", got, v.cam.ViewW)
	}

	paintedRight := got.X + got.W + float64(AttackMarkerWidth)/2
	if paintedRight > float64(v.cam.ViewW) {
		t.Errorf("painted right edge %v extends past the world viewport's right edge at %d (marker rect %+v, AttackMarkerWidth %d)",
			paintedRight, v.cam.ViewW, got, AttackMarkerWidth)
	}

	id, hit := topAt(v.entities, v.entityPickRect, float64(x), float64(y))
	if !hit || id != atLoID {
		t.Fatalf("premise: the unit near the edge must still be pickable through the unclipped rect, got id=%d hit=%v", id, hit)
	}
}

// TestAPopupDrawsNeitherInstrument is AC-9's drawn half.
//
// The mode is already lowered by the time a popup is up, so this asserts the
// SECOND gate: what a frame composed without a step having run first would draw.
// The fields are set directly for that reason — the case is precisely the one no
// dispatch produces.
func TestAPopupDrawsNeitherInstrument(t *testing.T) {
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
	if _, ok := f.v.attackTargetRect(); ok {
		t.Error("a target marker is drawn under a popup")
	}
}

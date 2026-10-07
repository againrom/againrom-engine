package ui

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// fiGrid is the side of every grid below, and fiSeen/fiDark two cells on it:
// one the plane marks visible and one it leaves unseen.
const fiGrid = 4

var (
	// fiSeen IS ROW ZERO ON PURPOSE. It was moved to row one when the health
	// bar was raised above the unit square, on the ground that row zero would
	// put the bar outside the viewport and the test would then be measuring
	// clipping rather than fog. Measured, that ground does not hold for this
	// fixture: fiViewer lays out 640x480 over a 4x4 grid of 32px cells, so the
	// map occupies 128px and the camera centres it with a margin of about 256px
	// on each side. A row-zero bar lands at screen y 168, nowhere near an edge.
	// The cell is back at the corner so that the case FACES the raised
	// geometry, and fiRowZeroBarIsOnScreen below asserts the margin rather than
	// leaving it as an assumption.
	fiSeen = image.Pt(0, 0)
	fiDark = image.Pt(2, 2)
)

// fiOwner is the local participant's roster slot, and fiFoe another. Neither
// is zero, so a gate comparing against the wrong operand — or against the
// zero value — answers differently.
const (
	fiOwner uint32 = 3
	fiFoe   uint32 = 5
)

// fiViewer is a viewer over a fiGrid square with fiSeen visible, every other
// cell unseen, and the local owner set.
func fiViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("fog-indicators", grid(fiGrid, fiGrid), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, 640, 480)
	v.SetLocalOwner(fiOwner)
	// The numeral cases need a font: composeNumeral answers nil without one
	// and numeralPlacements drops a record carrying no picture, so a viewer
	// with no font would pass the placement assertions for the wrong reason.
	v.SetFont(numeralFont())
	// The camera is put at the origin so the two cells below are inside the
	// view: placeArm culls off-screen glyphs, and a culled bar is absent for
	// a reason that has nothing to do with fog.
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()

	plane := make([]byte, fiGrid*fiGrid) // the zero value is FogUnseen
	plane[fiSeen.Y*fiGrid+fiSeen.X] = FogVisible
	v.SetFog(plane, fiGrid, fiGrid)
	return v
}

// fiUnit is one living unit with a health bar's worth of health.
func fiUnit(id uint32, owner uint32, cell image.Point) MapEntity {
	return MapEntity{ID: id, Owner: owner, Cell: cell, Life: LifeAlive, HP: 50, MaxHP: 100}
}

// The three ids every case pushes: an enemy in the dark, the same kind of
// enemy in the light, and one of the player's own in the dark.
const (
	fiDarkFoeID  uint32 = 1
	fiLightFoeID uint32 = 2
	fiDarkOwnID  uint32 = 3
)

func fiSnapshot() []MapEntity {
	return []MapEntity{
		fiUnit(fiDarkFoeID, fiFoe, fiDark),
		fiUnit(fiLightFoeID, fiFoe, fiSeen),
		fiUnit(fiDarkOwnID, fiOwner, fiDark),
	}
}

// TestAHealthBarDoesNotLeakThroughTheFog is the leak the owner reported: he
// saw a health bar standing over an enemy nothing else drew.
//
// It counts GROUNDS rather than naming ids, because healthBarScreenRects
// answers rectangles and not entities — so the assertion is over the number
// of bars the frame would paint, which is what the player actually sees.
func TestAHealthBarDoesNotLeakThroughTheFog(t *testing.T) {
	v := fiViewer(t)
	v.SetEntities(fiSnapshot())

	grounds, _ := v.healthBarScreenRects()
	if len(grounds) != 2 {
		t.Fatalf("%d health bars, want 2 — the enemy in the light and the player's own unit, "+
			"never the enemy in the dark", len(grounds))
	}

	// And with the enemy in the dark removed the count does not move, which
	// is what says the two survivors are the two intended ones rather than
	// any two.
	v.SetEntities(fiSnapshot()[1:])
	if again, _ := v.healthBarScreenRects(); len(again) != len(grounds) {
		t.Errorf("dropping the fogged enemy changed the bar count from %d to %d — "+
			"he was contributing one", len(grounds), len(again))
	}
}

// TestARowZeroHealthBarIsDrawnOnScreenAndNotClippedAway faces what the raised
// bar geometry does at the top edge of the map, which is the reason fiSeen was
// moved off row zero for a while.
//
// A health bar's top edge stands sixteen native rows ABOVE its own cell, so a
// unit on row zero has a bar with a negative map-space Y. StatusBarRect allows
// that on purpose and leaves the viewport to clip it after camera placement.
// This test states what the camera actually does with it here, so that a
// layout or camera change fails as a clipping failure rather than as one fewer
// bar in the counts above.
func TestARowZeroHealthBarIsDrawnOnScreenAndNotClippedAway(t *testing.T) {
	v := fiViewer(t)
	v.SetEntities(fiSnapshot())

	grounds, fills := v.healthBarScreenRects()
	if len(grounds) != 2 || len(fills) != 2 {
		t.Fatalf("%d ground(s) and %d fill(s), want 2 of each", len(grounds), len(fills))
	}
	if fiSeen.Y != 0 {
		t.Fatalf("fiSeen is at row %d; this case exists to exercise row zero", fiSeen.Y)
	}
	w, h := layoutViewport(v, 640, 480)
	for i, g := range grounds {
		if g.W <= 0 || g.H <= 0 {
			t.Fatalf("ground %d is %v — the bar was clipped to nothing", i, g)
		}
		if g.X < 0 || g.Y < 0 || g.X+g.W > float64(w) || g.Y+g.H > float64(h) {
			t.Errorf("ground %d is %v, outside the %dx%d viewport — a row-zero bar left the screen",
				i, g, w, h)
		}
	}
}

func TestAShotMarkDoesNotLeakThroughTheFog(t *testing.T) {
	v := fiViewer(t)
	ents := fiSnapshot()
	for i := range ents {
		shot := image.Pt(int(ents[i].Cell.X)*ShotScale, int(ents[i].Cell.Y)*ShotScale)
		ents[i].Shot = &shot
	}
	v.SetEntities(ents)

	if got := len(v.shotScreenRects()); got != 2 {
		t.Fatalf("%d shot marks, want 2 — a shot leaving an enemy in the fog is as much an "+
			"enemy indicator as his health bar", got)
	}
}

// TestADamageNumeralDoesNotLeakThroughTheFog covers BOTH halves of the
// numeral's gate, which is why it is one test in two parts: creation, in
// ingestDamage, and placement, in numeralPlacements.
//
// The second part is the one a creation-only gate would fail: a victim struck
// in the LIGHT and then walking into the DARK carries a figure that is still
// rising, and it must stop being painted the moment his cell goes dark.
func TestADamageNumeralDoesNotLeakThroughTheFog(t *testing.T) {
	t.Run("no figure is made for a blow struck in the dark", func(t *testing.T) {
		v := fiViewer(t)
		now := time.Unix(1_700_000_000, 0)

		v.SetEntities(fiSnapshot())
		v.step(Input{}, now) // the first sighting remembers and makes nothing

		hit := fiSnapshot()
		hit[0].HP = 10 // the enemy in the DARK takes 40
		v.SetEntities(hit)
		v.step(Input{}, now.Add(time.Millisecond))

		if _, live := v.DamageNumerals(); live != 0 {
			t.Errorf("%d figures for a blow landed on an enemy in the fog, want 0 — "+
				"a number floating in the dark says something is being hit there", live)
		}

		// The same blow on the enemy in the LIGHT does make one, so the
		// absence above is the fog and not a broken fixture.
		lit := fiSnapshot()
		lit[1].HP = 10
		v.SetEntities(lit)
		v.step(Input{}, now.Add(2*time.Millisecond))
		if _, live := v.DamageNumerals(); live != 1 {
			t.Errorf("%d figures for a blow landed in the light, want 1", live)
		}
	})

	t.Run("a figure already standing stops being painted when its unit goes dark", func(t *testing.T) {
		v := fiViewer(t)
		now := time.Unix(1_700_000_000, 0)

		v.SetEntities(fiSnapshot())
		v.step(Input{}, now)

		hit := fiSnapshot()
		hit[1].HP = 10 // the enemy in the LIGHT takes 40
		v.SetEntities(hit)
		v.step(Input{}, now.Add(time.Millisecond))
		if got := len(v.numeralPlacements()); got != 1 {
			t.Fatalf("%d placements for a visible victim, want 1", got)
		}

		// He walks into the dark with the figure still on its clock.
		moved := hit
		moved[1].Cell = fiDark
		v.SetEntities(moved)
		v.step(Input{}, now.Add(2*time.Millisecond))
		if got := len(v.numeralPlacements()); got != 0 {
			t.Errorf("%d placements after the victim walked into the fog, want 0 — "+
				"the creation gate alone cannot catch this one", got)
		}
	})
}

// TestASelectionRimDoesNotLeakThroughTheFog is the leak reachable in ordinary
// play: nothing scopes the selection to the local participant's own units, so
// tapping an enemy to look at him and watching him walk into the dark left a
// rim standing over him.
func TestASelectionRimDoesNotLeakThroughTheFog(t *testing.T) {
	v := fiViewer(t)
	v.SetEntities(fiSnapshot())

	v.sel = selection{fiLightFoeID}
	if len(v.selectionScreenRects()) == 0 {
		t.Fatal("premise: a selected enemy in the light draws no rim")
	}
	v.sel = selection{fiDarkFoeID}
	if got := len(v.selectionScreenRects()); got != 0 {
		t.Errorf("%d selection rects over an enemy in the fog, want 0", got)
	}
	v.sel = selection{fiDarkOwnID}
	if len(v.selectionScreenRects()) == 0 {
		t.Error("the local participant's OWN selected unit lost its rim in the fog; " +
			"the owner arm of fogGateEntity is decoded behaviour, not a convenience")
	}
}

// TestARouteDoesNotLeakThroughTheFog is the selection rim's sibling and says
// more than it does: a route shows where an enemy is AND where he is going.
func TestARouteDoesNotLeakThroughTheFog(t *testing.T) {
	v := fiViewer(t)
	ents := fiSnapshot()
	for i := range ents {
		ents[i].Route = []image.Point{{X: ents[i].Cell.X + 1, Y: ents[i].Cell.Y}}
	}
	v.SetEntities(ents)

	v.sel = selection{fiLightFoeID}
	if len(v.pathScreenSegments()) == 0 {
		t.Fatal("premise: a selected enemy in the light draws no route")
	}
	v.sel = selection{fiDarkFoeID}
	if got := len(v.pathScreenSegments()); got != 0 {
		t.Errorf("%d path segments over an enemy in the fog, want 0", got)
	}
	v.sel = selection{fiDarkOwnID}
	if len(v.pathScreenSegments()) == 0 {
		t.Error("the local participant's OWN selected unit lost its route in the fog")
	}
}

// TestTheAttackOutlineDoesNotLeakThroughTheFog is the cursor's own indicator,
// and it asserts BOTH sides of the asymmetry this hotfix chose: the outline is
// gone, and the press behind it is NOT — topAt still names the unit, because
// gating input is a behavioural claim no decoded fact backs and this is a
// hotfix.
func TestTheAttackOutlineDoesNotLeakThroughTheFog(t *testing.T) {
	a, v, _ := atOnMap(t)

	// Every cell unseen but the one the LOW unit stands on, so the foe at
	// (atFoeCol, atFoeRow) is in the dark.
	cols, rows := int(v.grid.Width), int(v.grid.Height)
	plane := make([]byte, cols*rows)
	plane[atLoRow*cols+atLoCol] = FogVisible
	v.SetFog(plane, cols, rows)
	v.SetLocalOwner(fiOwner) // no entity in atEntities carries it

	x, y := ptHover(a, v, atFoeCol, atFoeRow)
	a.step(afHeld(x, y), atAt)

	if _, ok := v.attackTargetRect(); ok {
		t.Error("the attack cursor outlined a unit standing in the fog")
	}
	if _, hit := topAt(v.entities, v.entityPickRect, float64(x), float64(y)); !hit {
		t.Error("the PRESS stopped naming the unit too — gating input is not what this " +
			"hotfix does, and cursor.go says why")
	}

	// Over the unit on the one visible cell the outline is still drawn, so
	// the absence above is the fog and not the fixture.
	bx, by := ptHover(a, v, atLoCol, atLoRow)
	a.step(afHeld(bx, by), atAt)
	if _, ok := v.attackTargetRect(); !ok {
		t.Error("nothing outlined over a unit on a VISIBLE cell")
	}
}

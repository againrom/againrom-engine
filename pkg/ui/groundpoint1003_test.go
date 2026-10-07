package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

// Where a spell sprite's own cell is.

// TestASpellSpriteStandsOnItsCellsCentre is the owner's Acid Stream and Wall
// of Earth reports at their shared cause: the cone's origin sitting left of
// the caster's cell in every projection, and the Wall of Earth sprite
// standing above the actor and appearing to float.
//
// Pos is in ShotScale units of a cell, so dividing it back gives the cell's
// TOP-LEFT CORNER. The decoded ground point is the cell's CENTRE —
// StaticAnchor's `col*CellSize + CellSize/2` (TERR-SPR-040, TERR-SPR-038), and
// the original's own actors stand at fine position 0x80 in both axes
// (`ANIM-MSG-005`). Missing, the sprite was drawn CellSize/2 = 16 px left and
// 16 px up of its cell in world pixels, before any projection.
//
// The assertion is against the SAME arithmetic the static-object layer uses,
// not against a restated literal, so the two derivations of a cell's ground
// point cannot come apart without this reddening.
func TestASpellSpriteStandsOnItsCellsCentre(t *testing.T) {
	t.Parallel()

	v := commandViewer(t)
	// Centring halves of zero, so the placement is the ground point itself.
	sheet := uiSheet(4, 8, 8, 0, 0)
	const col, row = 5, 7
	v.SetSpellBolts([]SpellBolt{{
		Cell: image.Pt(col, row), Pos: image.Pt(col*ShotScale, row*ShotScale), Sheet: sheet, Frame: 0,
	}})

	placed := v.spellArtPlacements()
	if len(placed) != 1 {
		t.Fatalf("one bolt placed %d rectangles, want one", len(placed))
	}

	// The static layer's own answer for the same cell with every centring term
	// zeroed, which leaves exactly the GROUND POINT. The two layers subtract
	// their halves differently — a projectile subtracts the registry's Width/2
	// outright (`ANIM-PROJ-026`) while a static object corrects by its canvas
	// and frame (TERR-SPR-040) — and the ground point they measure from is the
	// one thing they must agree on.
	wantX, wantY, _, _ := terrain.StaticAnchor(col, row, 0, 0, 0, 0, 0, 0, 0, 0)
	gotX, gotY := EffectGroundPoint(image.Pt(col*ShotScale, row*ShotScale), sheet)
	if gotX != wantX || gotY != wantY {
		t.Fatalf("a spell sprite on cell (%d,%d) is blitted at world pixel (%d,%d); "+
			"the static layer puts the same art on the same cell at (%d,%d). "+
			"A sprite off by half a cell is what the acid cone's left-shifted origin and the "+
			"floating Wall of Earth reports both are",
			col, row, gotX, gotY, wantX, wantY)
	}

	// And a whole cell of Pos moves the sprite by exactly one cell, so the term
	// added above is a constant and not a scale.
	nextX, _ := EffectGroundPoint(image.Pt((col+1)*ShotScale, row*ShotScale), sheet)
	if nextX-gotX != terrain.CellSize {
		t.Errorf("one cell east moved the sprite by %d world pixels, want %d", nextX-gotX, terrain.CellSize)
	}
}

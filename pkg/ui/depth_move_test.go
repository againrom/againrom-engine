package ui

// 0151-layers-and-names defect 7: a unit standing on a corpse or a sack draws
// above it (0111's DepthTie tiers already cover the stationary case). This
// file pins the moving case, which was wrong before this story: a unit one
// tick into a step off that cell drew BELOW what it was leaving, because its
// depth row followed its destination cell rather than the cell the step
// began at (overlay.go, entityLayer's "THE DEPTH ROW" comment).
//
// Both fixtures are bare 4x4 flat grids with an identity-adjacent camera,
// mirroring sacks_test.go's own sackIdentityViewer. No game data is read, no
// install is touched and no window opens.

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

// depthMoveViewer is sackIdentityViewer's own shape, rebuilt here so this
// file owns its fixture end to end rather than reaching into another one.
func depthMoveViewer(t *testing.T) *Viewer {
	t.Helper()
	v := newViewer(t, grid(4, 4))
	layoutViewport(v, 4*terrain.CellSize, 4*terrain.CellSize)
	return v
}

// TestAMovingUnitStaysAboveTheSackItLeaves: a sack sits at (1,2); a unit
// stands at (1,1) carrying Step (0,-1), the shape entityDraws hands over for
// a unit one or more ticks into a northward step away from (1,2) (Step is
// nonzero for the whole crossing, not only its first tick — pkg/game/world.go
// recordCells). The unit's depth row must still tie with the sack's, so
// DepthTie puts the unit after it in the merge (drawn later, on top).
func TestAMovingUnitStaysAboveTheSackItLeaves(t *testing.T) {
	v := depthMoveViewer(t)
	v.SetSackFrames([]*terrain.StaticFrame{sackFrame(4, 4)})
	v.SetSacks([]MapSack{{Cell: image.Pt(1, 2), FrameIndex: 0}})

	art := entityArtA()
	v.SetEntities([]MapEntity{
		{ID: 1, Cell: image.Pt(1, 1), Step: image.Pt(0, -1), Art: art, Frame: art.Frames[0], Life: LifeAlive},
	})

	v.planeSprites()
	if len(v.depthOrder) != 2 {
		t.Fatalf("depthOrder has %d entries, want 2 (one sack, one unit)", len(v.depthOrder))
	}
	if v.depthOrder[0].Kind != terrain.PlaneSack || v.depthOrder[1].Kind != terrain.PlaneEntity {
		t.Fatalf("draw order = %v, want sack then unit — a unit mid-step off a sack's cell must still "+
			"draw above it", v.depthOrder)
	}
}

// TestAMovingUnitDrawsAboveTheSackItEnters is the southward counterpart to
// the test above: a sack sits at (1,2); a unit's SIM cell — its destination
// while the crossing runs — is already (1,2), carrying Step (0,1), the
// shape entityDraws hands over for a unit one or more ticks into a
// southward step from (1,1) onto (1,2). Before this task, the depth row
// followed the origin cell (1,1) for the whole crossing (T3's fix), which
// put the unit BEFORE the sack in the merge — behind it — for the whole
// southward crossing. The later of the two rows a southward crossing spans
// is the destination, (1,2), the sack's own row, so the tie (0111 DepthTie)
// must place the unit after the sack.
func TestAMovingUnitDrawsAboveTheSackItEnters(t *testing.T) {
	v := depthMoveViewer(t)
	v.SetSackFrames([]*terrain.StaticFrame{sackFrame(4, 4)})
	v.SetSacks([]MapSack{{Cell: image.Pt(1, 2), FrameIndex: 0}})

	art := entityArtA()
	v.SetEntities([]MapEntity{
		{ID: 1, Cell: image.Pt(1, 2), Step: image.Pt(0, 1), Art: art, Frame: art.Frames[0], Life: LifeAlive},
	})

	v.planeSprites()
	if len(v.depthOrder) != 2 {
		t.Fatalf("depthOrder has %d entries, want 2 (one sack, one unit)", len(v.depthOrder))
	}
	if v.depthOrder[0].Kind != terrain.PlaneSack || v.depthOrder[1].Kind != terrain.PlaneEntity {
		t.Fatalf("draw order = %v, want sack then unit — a unit stepping onto a sack's cell must "+
			"draw above it", v.depthOrder)
	}
}

// TestAMovingUnitStaysAboveTheSackSideways is the sideways case: a sack
// sits at (2,1); a unit at (2,1) — the destination — carries Step (1,0), a
// step onto that cell from (1,1) that never changes row. Both rows the
// crossing spans are already equal, so the fix's comparison changes
// nothing and the ordinary same-row tie (0111 DepthTie) alone must still
// put the unit above the sack.
func TestAMovingUnitStaysAboveTheSackSideways(t *testing.T) {
	v := depthMoveViewer(t)
	v.SetSackFrames([]*terrain.StaticFrame{sackFrame(4, 4)})
	v.SetSacks([]MapSack{{Cell: image.Pt(2, 1), FrameIndex: 0}})

	art := entityArtA()
	v.SetEntities([]MapEntity{
		{ID: 1, Cell: image.Pt(2, 1), Step: image.Pt(1, 0), Art: art, Frame: art.Frames[0], Life: LifeAlive},
	})

	v.planeSprites()
	if len(v.depthOrder) != 2 {
		t.Fatalf("depthOrder has %d entries, want 2 (one sack, one unit)", len(v.depthOrder))
	}
	if v.depthOrder[0].Kind != terrain.PlaneSack || v.depthOrder[1].Kind != terrain.PlaneEntity {
		t.Fatalf("draw order = %v, want sack then unit — a unit crossing sideways on a sack's row must "+
			"draw above it", v.depthOrder)
	}
}

// TestAStationaryUnitAboveTheSackIsUnchanged is the fixture above with Step
// left zero: the regression guard that the fix above touches only a
// transiting entity's row, not an idle one's — 0111's own stationary
// behaviour must survive unmoved.
func TestAStationaryUnitAboveTheSackIsUnchanged(t *testing.T) {
	v := depthMoveViewer(t)
	v.SetSackFrames([]*terrain.StaticFrame{sackFrame(4, 4)})
	v.SetSacks([]MapSack{{Cell: image.Pt(1, 2), FrameIndex: 0}})

	art := entityArtA()
	v.SetEntities([]MapEntity{
		{ID: 1, Cell: image.Pt(1, 2), Art: art, Frame: art.Frames[0], Life: LifeAlive},
	})

	v.planeSprites()
	if len(v.depthOrder) != 2 {
		t.Fatalf("depthOrder has %d entries, want 2 (one sack, one unit)", len(v.depthOrder))
	}
	if v.depthOrder[0].Kind != terrain.PlaneSack || v.depthOrder[1].Kind != terrain.PlaneEntity {
		t.Fatalf("draw order = %v, want sack then unit — a stationary unit on a sack's cell draws above it",
			v.depthOrder)
	}
}

// TestAMovingUnitStaysAboveTheCorpseItLeaves: the owner named corpses beside
// sacks. A corpse is a dead entity in the SAME stream as a living, moving
// one (DepthTie — overlay.go: only LifeDead takes TieCorpse), so the row fix
// above is read by rowOrder's ordinary same-stream sort here, not by a
// second rule for a second stream. The corpse is entity 1 at (1,2); the
// moving unit is entity 2, stepping off that same cell northward.
func TestAMovingUnitStaysAboveTheCorpseItLeaves(t *testing.T) {
	v := depthMoveViewer(t)
	art := entityArtA()
	v.SetEntities([]MapEntity{
		{ID: 1, Cell: image.Pt(1, 2), Art: art, Frame: art.Frames[0], Life: LifeDead},
		{ID: 2, Cell: image.Pt(1, 1), Step: image.Pt(0, -1), Art: art, Frame: art.Frames[0], Life: LifeAlive},
	})

	sprites, _, _ := v.entityLayer()
	if len(sprites) != 2 {
		t.Fatalf("entityLayer() resolved %d sprites, want 2", len(sprites))
	}
	if sprites[0].DepthTie != terrain.TieCorpse || sprites[1].DepthTie != terrain.TieUnit {
		t.Fatalf("sprite DepthTie = [%v %v], want [TieCorpse TieUnit] — fixture does not discriminate",
			sprites[0].DepthTie, sprites[1].DepthTie)
	}

	v.planeSprites()
	if len(v.depthOrder) != 2 {
		t.Fatalf("depthOrder has %d entries, want 2 (the corpse and the moving unit)", len(v.depthOrder))
	}
	if v.depthOrder[0].Kind != terrain.PlaneEntity || v.depthOrder[0].Index != 0 ||
		v.depthOrder[1].Kind != terrain.PlaneEntity || v.depthOrder[1].Index != 1 {
		t.Fatalf("draw order = %v, want the corpse (index 0) then the moving unit (index 1) — a unit mid-step "+
			"off a corpse's cell must still draw above it", v.depthOrder)
	}
}

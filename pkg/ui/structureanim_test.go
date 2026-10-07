package ui

// TERR-STRUCT-102's per-drawable sight gate, read into the viewer that has a
// fog model to enforce it with (hotfix, owner report 3, DIV-1351): a
// structure's own animation phase advances only while its cell is in
// CURRENT sight, freezing at its last selected frame rather than resetting
// or continuing on the tick clock alone.

import (
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// structureAnimGrid is one 1x1 animating class anchored at (2,3) on an 8x8
// map -- structureFrame's own tile-sized frame, structures_test.go's own
// fixture shape, with a three-phase timeline so three consecutive live
// counters select three different frames (objectAnimTimeline's own reason).
func structureAnimGrid() (terrain.Grid, *terrain.StructureSet) {
	set := new(terrain.StructureSet)
	set.Classes[1] = &terrain.StructureClass{
		TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames:   []*terrain.StaticFrame{structureFrame(), structureFrame(), structureFrame()},
		Timeline: []int{0, 1, 2}, Rank: []int{0}, Live: 1,
	}
	g := terrain.Grid{
		Width: 8, Height: 8, Tiles: make([]uint16, 64), Altitudes: make([]uint8, 64),
		Structures: []terrain.StructureRecord{{ID: 7, X: 2 << 8, Y: 3 << 8, Key: 1}},
	}
	return g, set
}

func structureFrameAt(v *Viewer, id uint32) *terrain.StaticFrame {
	for _, p := range v.structurePlacements() {
		if p.StructureID == id {
			return p.Frame
		}
	}
	return nil
}

func allVisiblePlane(w, h int) []byte {
	p := make([]byte, w*h)
	for i := range p {
		p[i] = FogVisible
	}
	return p
}

func allExploredPlane(w, h int) []byte {
	p := make([]byte, w*h)
	for i := range p {
		p[i] = FogExplored
	}
	return p
}

// TestStructureAnimationFreezesOutOfCurrentSight is the claim itself: a
// structure in current sight advances with the shared live counter, exactly
// as SelectStructureFrame's own timeline lookup gives it (computed here,
// never copied); the moment its cell drops to explored-but-not-visible, that
// selection is overridden back to whatever phase was last observed while
// visible and holds there across further ticks -- not reset to the base grid
// frame, which is what a plain animate=false fallback would draw instead.
func TestStructureAnimationFreezesOutOfCurrentSight(t *testing.T) {
	g, set := structureAnimGrid()
	class := set.Classes[1]
	v := newStructureViewer(t, g, set, true)
	now := time.Unix(0, 0)
	now = driveObjectAnim(v, now, 0)
	v.SetFog(allVisiblePlane(8, 8), 8, 8)

	now = driveObjectAnim(v, now, 1)
	counter := v.AnimationCounter()
	visibleFrame := wantStructureFrame(v, class, counter)
	if got := structureFrameAt(v, 7); got != visibleFrame {
		t.Fatalf("in sight at counter %d: frame = %p, want the live selection %p", counter, got, visibleFrame)
	}
	// The fixture only proves something if this counter's own phase is not
	// the base grid frame -- otherwise freezing and the plain fallback would
	// draw the same thing and the rest of this test could not tell them
	// apart.
	if visibleFrame == class.Frames[0] {
		t.Fatalf("counter %d selects the base grid frame; pick a fixture counter where TestStructureAnimationFreezesOutOfCurrentSight's own timeline gives a nonzero phase", counter)
	}

	// Leave sight: explored, not visible (TERR-TILE-079's middle state). No
	// tick has run yet, so nothing should move.
	v.SetFog(allExploredPlane(8, 8), 8, 8)
	if got := structureFrameAt(v, 7); got != visibleFrame {
		t.Fatalf("frame changed the instant sight was lost with no tick elapsed: got %p, want %p", got, visibleFrame)
	}

	// Advance once more while out of sight. The live counter moves and would
	// select a DIFFERENT frame if nothing gated it -- proving the fixture can
	// tell the two apart -- but the drawn frame must stay the one last
	// observed in sight.
	now = driveObjectAnim(v, now, 1)
	liveNow := wantStructureFrame(v, class, v.AnimationCounter())
	if liveNow == visibleFrame {
		t.Fatalf("live counter %d selects the same frame as the frozen one; the fixture cannot discriminate", v.AnimationCounter())
	}
	if got := structureFrameAt(v, 7); got != visibleFrame {
		t.Fatalf("out of sight: frame = %p, want the frozen in-sight frame %p (live would have been %p)", got, visibleFrame, liveNow)
	}
}

// TestStructureAnimationResumesFromLiveCounterOnReturn is the other half:
// once a structure's cell returns to current sight, it advances from
// whatever the shared live counter is now -- exactly the plain
// AnimateStructureStates behaviour this pass leaves alone while visible --
// and not from a resumed count starting at its own frozen value.
func TestStructureAnimationResumesFromLiveCounterOnReturn(t *testing.T) {
	g, set := structureAnimGrid()
	v := newStructureViewer(t, g, set, true)
	now := time.Unix(0, 0)
	now = driveObjectAnim(v, now, 0)
	v.SetFog(allVisiblePlane(8, 8), 8, 8)
	now = driveObjectAnim(v, now, 1)

	v.SetFog(allExploredPlane(8, 8), 8, 8)
	now = driveObjectAnim(v, now, 5)
	frozen := structureFrameAt(v, 7)

	v.SetFog(allVisiblePlane(8, 8), 8, 8)
	resumed := structureFrameAt(v, 7)
	want := wantStructureFrame(v, set.Classes[1], v.AnimationCounter())
	if resumed != want {
		t.Fatalf("resumed frame = %p, want the live-counter selection %p (frozen was %p)", resumed, want, frozen)
	}
}

// wantStructureFrame is the render tier's own selector, called directly, so
// the expectation is computed from SelectStructureFrame and never from a
// second copy of its rule.
func wantStructureFrame(v *Viewer, c *terrain.StructureClass, counter uint32) *terrain.StaticFrame {
	idx := terrain.SelectStructureFrame(c, 0, counter, true, false)
	if idx < 0 || idx >= len(c.Frames) {
		return nil
	}
	return c.Frames[idx]
}

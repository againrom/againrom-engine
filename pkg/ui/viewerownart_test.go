package ui

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// ownArtBundle is two classes on one sheet of three frames: class 1 names
// frame 1 and class 2 names frame 2, as the nine stones name frames 0..8.
func ownArtBundle() *terrain.StaticSet {
	sheet := []*terrain.StaticFrame{staticsFrame(8, 8, 1), staticsFrame(8, 8, 2), staticsFrame(8, 8, 3)}
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{Width: 64, Height: 64, CenterX: 32, CenterY: 60, Frame: sheet[1], Frames: sheet, Index: 1}
	set.Classes[2] = &terrain.StaticClass{Width: 64, Height: 64, CenterX: 32, CenterY: 60, Frame: sheet[2], Frames: sheet, Index: 2}
	return set
}

func ownArtFrames(t *testing.T, set *terrain.StaticSet, ownArt bool) map[uint8]*terrain.StaticFrame {
	t.Helper()
	v := newStaticsViewer(t, set, true, false)
	v.SetAnimated(false)
	v.SetOwnArt(ownArt)
	out := map[uint8]*terrain.StaticFrame{}
	for _, p := range v.staticPlacements() {
		out[v.grid.Overlay[p.Cell.Y*v.grid.Width+p.Cell.X]] = p.Frame
	}
	return out
}

// With the animation held, a viewer that keeps its own art draws each class's
// Index frame; one that does not draws the original's frame 0 for them all.
func TestViewerOwnArtDrawsEachClassItsOwnFrameWhileHeld(t *testing.T) {
	set := ownArtBundle()
	held := ownArtFrames(t, set, false)
	if held[1] != set.Classes[1].Frames[0] || held[2] != set.Classes[1].Frames[0] {
		t.Fatalf("the held viewer without own art did not draw frame 0 for both classes")
	}
	own := ownArtFrames(t, set, true)
	if own[1] != set.Classes[1].Frames[1] || own[2] != set.Classes[2].Frames[2] || own[1] == own[2] {
		t.Fatalf("own art drew %p and %p, want frames 1 and 2 of the sheet", own[1], own[2])
	}
}

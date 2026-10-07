package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

func TestLiveDestroyedStructureSelectsOnlyItsOwnRuinFrames(t *testing.T) {
	c := structClass(1, 1, 1)
	c.Frames = structFrames(2) // intact block, then ruin block
	set := structSet(map[byte]*terrain.StructureClass{1: c})
	a, b := structAt(2, 3, 1), structAt(4, 3, 1)
	a.ID, b.ID = 11, 12
	places, _, animated := terrain.StructurePlacements(structGrid(8, 8, a, b), set, nil, 0)

	got := terrain.AnimateStructureStates(nil, places, animated, 0, true, false, map[uint32]bool{11: true})
	if len(got) != 2 {
		t.Fatalf("draw entries = %d, want 2", len(got))
	}
	byID := map[uint32]*terrain.StaticFrame{}
	for _, p := range got {
		byID[p.StructureID] = p.Frame
	}
	if byID[11] != c.Frames[1] {
		t.Errorf("destroyed structure frame = %p, want ruin frame %p", byID[11], c.Frames[1])
	}
	if byID[12] != c.Frames[0] {
		t.Errorf("intact structure frame = %p, want base frame %p", byID[12], c.Frames[0])
	}
}

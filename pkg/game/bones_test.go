package game

// What the seam carries for a body past its fall (AC-14).
//
// SEPARATE CONTEXT, and it is the sheet index: every expected frame below is
// computed HERE, from the block rule written out in this file, and never read
// back from the selection under test.
//
// EVERY ASSERTION IS MADE ON THE PUSHED ENTITY. A stage held correctly in the
// world while the seam carried the wrong frame is exactly the failure this is
// about. The bodies arrive at their stages through the simulation's own ladder,
// so what is drawn is a consequence of a world state and not of a flag set here.

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The corpse sheet's two blocks. The bone block sits past the dying one at the
// tail base the idle cycle shares, which is what the derivation computes.
const (
	bnDyingBase, bnDyingSlot, bnDirs = 40, 4, 8
	bnTailBase                       = bnDyingBase + bnDirs*bnDyingSlot
	bnBoneSlot                       = 3
	bnFrames                         = bnTailBase + bnDirs*bnBoneSlot
)

// bnBoneFrameAt is the sheet-contract rule transcribed: the tail base, the
// direction's own slot at the bone slot length, and the stage offset.
func bnBoneFrameAt(oct, stage int) int {
	return bnTailBase + oct*bnBoneSlot + (stage - 2)
}

// bnBundle is a two-class set: a unit whose body is a second class carrying both
// blocks, and a second pair whose corpse class carries a dying block and NO bone
// one — the fall-through case.
func bnBundle() *terrain.UnitSet {
	corpse := worldFixtureArt(16, 16, 8, 14, 4, 4, bnFrames)
	corpse.Anim = terrain.UnitAnim{S: 16, D: bnDirs,
		DyingBase: bnDyingBase, DyingSlot: bnDyingSlot,
		TailBase: bnTailBase, BoneSlot: bnBoneSlot}

	boneless := worldFixtureArt(16, 16, 8, 14, 4, 4, bnTailBase)
	boneless.Anim = terrain.UnitAnim{S: 16, D: bnDirs,
		DyingBase: bnDyingBase, DyingSlot: bnDyingSlot, TailBase: bnTailBase}

	classes := map[int32]*terrain.UnitClass{
		1: deathLiveArt(), 2: corpse,
		3: deathLiveArt(), 4: boneless,
	}
	linkCorpses(classes, map[int32]int32{1: 2, 2: 2, 3: 4, 4: 4})
	return &terrain.UnitSet{Classes: classes}
}

func bnWorld(t *testing.T, ents ...sim.Entity) *mapWorld {
	t.Helper()
	grid := make([]byte, deathW*deathH)
	w, err := sim.NewWorld(deathSeed, sim.Bounds{Width: deathW, Height: deathH},
		sim.ModeCanonical, grid, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	v, err := ui.NewViewer("bones", terrain.Grid{
		Width: deathW, Height: deathH, Tiles: make([]uint16, deathW*deathH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	return newMapWorld(w, nil, bnBundle(), v)
}

// bnBody is one body, at a health that puts it on a named rung once the ladder
// is read. Its dying time is none, so it is torn down on the first tick and the
// rung it lands on is the one its health names.
func bnBody(id sim.EntityID, class int32, x int32, hp int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: 3, Class: class, HP: hp, MaxHP: 100}
}

// TestABodyPastItsFallIsDrawnFromTheBoneBlock is AC-14: the three bone stages,
// each carrying the CORPSE class's art at that class's own bone frame for the
// direction the body died facing.
func TestABodyPastItsFallIsDrawnFromTheBoneBlock(t *testing.T) {
	// -15, -25 and -45 put the three on the three rungs the ladder names, and no
	// two of them share one.
	mw := bnWorld(t, bnBody(1, 1, 1, -15), bnBody(2, 1, 2, -25), bnBody(3, 1, 3, -45))
	corpse := mw.units.Classes[2]

	// The tick that reads the ladder. Before it every body is at the stage its
	// death put it at, which is the fall.
	mw.tick()

	for _, tc := range []struct {
		id    sim.EntityID
		stage int
	}{{1, 2}, {2, 3}, {3, 4}} {
		d := deathDraw(t, mw, tc.id)
		if d.Art != corpse {
			t.Errorf("entity %d is drawn as some class other than its corpse class", tc.id)
			continue
		}
		want := bnBoneFrameAt(north, tc.stage)
		if got := frameIndex(corpse, d.Frame); got != want {
			t.Errorf("entity %d at stage %d draws frame %d, want %d",
				tc.id, tc.stage, got, want)
		}
		if d.Mirror {
			t.Errorf("entity %d draws mirrored at the eight-way layout", tc.id)
		}
	}
}

// TestABodyStillFallingIsDrawnFromTheDyingBlock is the other side of the same
// chain, and the one that says the bone arm has not swallowed the fall: a body
// at the first stage still plays and then holds its dying frames.
func TestABodyStillFallingIsDrawnFromTheDyingBlock(t *testing.T) {
	mw := bnWorld(t, bnBody(1, 1, 1, -1))
	corpse := mw.units.Classes[2]
	mw.tick()

	if got := mw.world.Entities()[0].Decay; got != sim.DecayFallen {
		t.Fatalf("the body is at stage %d, want the first — this fixture measures the fall", got)
	}
	d := deathDraw(t, mw, 1)
	if d.Art != corpse {
		t.Fatal("a fallen body is drawn as some class other than its corpse class")
	}
	if got := frameIndex(corpse, d.Frame); got < bnDyingBase || got >= bnTailBase {
		t.Errorf("a body at the first stage draws frame %d, which is outside the dying block [%d,%d)",
			got, bnDyingBase, bnTailBase)
	}
}

func TestACorpseClassWithNoBoneBlockKeepsTheDrawingItHad(t *testing.T) {
	mw := bnWorld(t, bnBody(1, 3, 1, -45))
	boneless := mw.units.Classes[4]
	mw.tick()

	if got := mw.world.Entities()[0].Decay; got < sim.DecayBones {
		t.Fatalf("the body is at stage %d, want a bone stage — this fixture measures the fall-through",
			got)
	}
	d := deathDraw(t, mw, 1)
	if d.Frame == nil {
		t.Fatal("the body is not drawn at all")
	}
	if d.Art != boneless {
		t.Fatal("the body is drawn as some class other than its corpse class")
	}
	if got := frameIndex(boneless, d.Frame); got < bnDyingBase || got >= bnTailBase {
		t.Errorf("a body whose corpse class has no bone block draws frame %d, want one of the "+
			"dying block's [%d,%d) — the drawing it had before this story",
			got, bnDyingBase, bnTailBase)
	}
}

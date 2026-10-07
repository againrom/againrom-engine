package game

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// latecorpse1201DeadInput builds one Source.MapUnitID 0 original-dead
// candidate at the given stage, the shape deadSourceFault/deadStateFault
// (pkg/sim/originaldead.go) admit: stage in [2,5] (that function's own
// window, wider than the [2,4] draw window this story reads out of
// SAV-DEADLOAD-128), Timer 0, FineX/FineY 128, HP in [-600,-10] for a
// nonterminal stage or below -10000 with RuntimeID 0 for the terminal one.
// A stage 0 or 1 record cannot be built this way: deadStateFault refuses
// both, so this project's only public original-dead constructor admits no
// record below stage 2 for any caller, this test included.
func latecorpse1201DeadInput(id sim.EntityID, stage uint8) sim.OriginalDeadActor {
	runtime, hp := uint32(50+id), int16(-40)
	if stage == 5 {
		runtime, hp = 0, -10001
	}
	return sim.OriginalDeadActor{ID: id, Source: sim.OriginalDeadSource{
		Identity: 900 + uint32(id), ArchiveIndex: uint16(id) + 1, Class: 2,
		State: sim.DeadActorState{RuntimeID: runtime, Cell: 0x0304, FineX: 128, FineY: 128, Stage: stage, HP: hp},
	}}
}

// latecorpse1201World builds a bare 8x8 world holding exactly the
// MapUnitID-0 dead records the caller names, none of them ever a living
// entity (SAV-DEADLOAD-125) -- ImportOriginalDeadActors' own MapUnitID-0 arm
// never creates one, so this world needs no sim.Entity at all.
func latecorpse1201World(t *testing.T, batch []sim.OriginalDeadActor) *sim.World {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{}, nil, nil, sim.Relations{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalDeadActors(batch); err != nil {
		t.Fatal(err)
	}
	return w
}

// latecorpse1201Corpse is an eight-frame bone sheet a stage 2-4 record
// resolves through terrain.SelectBoneFrame at virtualDeadDraws' own fixed
// facing (world.go, sheetOctant(0) == 4 -- FacingDir(0) is 0, and octant 4
// is the "away" direction the +4 rotation gives a facing-0 sheet): with
// D != 5, unitSlot leaves that octant as the slot unchanged, so
// frame index = TailBase + 4*BoneSlot + (stage-2). TailBase 0 and BoneSlot 1
// put stages 2, 3 and 4 at frames 4, 5 and 6, each with its own pointer;
// eight frames leaves headroom past that.
func latecorpse1201Corpse() *terrain.UnitClass {
	corpse := worldFixtureArt(16, 16, 8, 14, 4, 4, 8)
	corpse.Anim = terrain.UnitAnim{BoneSlot: 1, TailBase: 0, D: 1}
	return &terrain.UnitClass{Corpse: corpse}
}

func TestLateCorpseConstructor1201ResolvedAndUnresolvedDraws(t *testing.T) {
	const (
		earlyResolved    sim.EntityID = 1
		lateResolved     sim.EntityID = 2
		terminalResolved sim.EntityID = 3
		midUnresolved    sim.EntityID = 4
	)
	w := latecorpse1201World(t, []sim.OriginalDeadActor{
		latecorpse1201DeadInput(earlyResolved, 2),
		latecorpse1201DeadInput(lateResolved, 4),
		latecorpse1201DeadInput(terminalResolved, 5),
		latecorpse1201DeadInput(midUnresolved, 3),
	})
	class := latecorpse1201Corpse()
	mw := &mapWorld{world: w,
		art:   map[sim.EntityID]*terrain.UnitClass{earlyResolved: class, lateResolved: class, terminalResolved: class},
		tiers: map[sim.EntityID]int{}, swing: map[sim.EntityID]int{}, died: map[sim.EntityID]int{}}

	draws := map[sim.EntityID]ui.MapEntity{}
	for _, d := range mw.entityDraws() {
		draws[sim.EntityID(d.ID)] = d
	}

	for _, id := range []sim.EntityID{earlyResolved, lateResolved} {
		d, ok := draws[id]
		if !ok {
			t.Fatalf("entity %d: no draw, want a resolved corpse draw inside the stage 2-4 window", id)
		}
		if d.Art == nil || d.Frame == nil {
			t.Fatalf("entity %d: Art/Frame nil though mw.art resolves this record", id)
		}
		if d.DrawCategory == 0 {
			t.Fatalf("entity %d: DrawCategory left at zero on a resolved draw", id)
		}
	}
	if _, ok := draws[terminalResolved]; ok {
		t.Fatal("stage-5 record drew: SAV-DEADLOAD-128's window excludes the terminal stage even though its body resolves")
	}
	if _, ok := draws[midUnresolved]; ok {
		t.Fatal("record with no resolving mw.art entry drew a placeholder; main draws nothing for it (story1201-adversarial.md Finding 1)")
	}
}

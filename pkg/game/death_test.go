package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	deathW, deathH = 8, 8
	deathSeed      = 1

	// The corpse sheet's dying block: base 40, four frames per direction,
	// eight stored directions. The unit's OWN sheet uses different numbers, so
	// a death drawn from the wrong class is a wrong index and not merely a
	// wrong pointer.
	corpseBase, corpseSlot, corpseDirs = 40, 4, 8

	// north is the SHEET octant a unit that has never turned draws in: 4 in the
	// sheet ordering (S 0, SW 1, W 2, NW 3, N 4, NE 5, E 6, SE 7). Written out
	// here rather than taken from the translation under test.
	north = 4
)

// deathFrameAt is the sheet-contract rule transcribed here: one dying frame
// per two ticks, held at the block's last frame once the block has played out,
// at the direction's own slot.
//
// A unit that has never turned falls facing NORTH, sheet octant 'north' below,
// where before 0081 it fell facing SOUTH. Neither is decoded: the old default
// was the sign-octant table's centre cell, reached by a unit whose remembered
// step was never written; the new one is the simulation facing byte's zero,
// which is the movement delta table's own index 0. What is unchanged is that a
// body keeps the direction it died in, which the west case below still measures.
func deathFrameAt(oct, tick int) int {
	phase := tick / 2
	if phase > corpseSlot-1 {
		phase = corpseSlot - 1
	}
	return corpseBase + oct*corpseSlot + phase
}

// deathCorpseArt is the class whose sheet carries the body: a dying block at
// the constants above, and enough frames to hold it.
func deathCorpseArt() *terrain.UnitClass {
	c := worldFixtureArt(16, 16, 8, 14, 4, 4, corpseBase+corpseDirs*corpseSlot)
	c.Anim = terrain.UnitAnim{S: 16, D: corpseDirs, DyingBase: corpseBase, DyingSlot: corpseSlot}
	return c
}

// deathLiveArt is the class an entity walks around as: a sixteen-frame sheet
// whose own descriptor names NO dying block, so nothing it could select is a
// death frame and the two sheets can never be confused.
func deathLiveArt() *terrain.UnitClass {
	c := worldFixtureArt(16, 16, 8, 14, 4, 4, 16)
	c.Anim = terrain.UnitAnim{S: 16, D: 8,
		MoveBase: 0, AttackBase: 16, DyingBase: 16, TailBase: 8,
		MoveSlot: 1, MoveWind: 0, IdleSlot: 1, Total: 16,
		MoveTrack: []int{0}, MoveOK: true,
		IdleTrack: []int{0}, IdleOK: true}
	return c
}

// deathBundle is the class set every world here resolves against, linked
// through the production resolution rather than by hand:
//
//	1 -> 2   a unit whose body is another class's sheet
//	2 -> 2   the corpse class itself
//	3 -> 4   a unit whose corpse class holds NO frames
//	4 -> 4   that frameless class
//	5 -> 6   a unit whose corpse class has no dying block
//	6 -> 6   that blockless class
//	7 -> 99  a unit naming a class this bundle does not hold
//	8 -> 8   a unit with no art at all, which draws the square
func deathBundle() *terrain.UnitSet {
	blockless := worldFixtureArt(16, 16, 8, 14, 4, 4, 32)
	blockless.Anim = terrain.UnitAnim{S: 16, D: 8, DyingBase: 16, DyingSlot: 0}
	classes := map[int32]*terrain.UnitClass{
		1: deathLiveArt(),
		2: deathCorpseArt(),
		3: deathLiveArt(),
		4: {Width: 16, Height: 16, CenterX: 8, CenterY: 14},
		5: deathLiveArt(),
		6: blockless,
		7: deathLiveArt(),
		8: {Width: 16, Height: 16, CenterX: 8, CenterY: 14},
	}
	linkCorpses(classes, map[int32]int32{1: 2, 2: 2, 3: 4, 4: 4, 5: 6, 6: 6, 7: 99, 8: 8})
	return &terrain.UnitSet{Classes: classes}
}

// deathWorld is a driver over a hand-built world, with a schedule so a blow
// lands at a tick this file names rather than through the front-end's seam.
func deathWorld(t *testing.T, sched [][]sim.Command, ents ...sim.Entity) *mapWorld {
	t.Helper()
	grid := make([]byte, deathW*deathH)
	w, err := sim.NewWorld(deathSeed, sim.Bounds{Width: deathW, Height: deathH}, sim.ModeCanonical, grid, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	v, err := ui.NewViewer("death", terrain.Grid{
		Width: deathW, Height: deathH, Tiles: make([]uint16, deathW*deathH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	return newMapWorld(w, sched, deathBundle(), v)
}

// deathUnit is one entity: a whole cell, the given class, at full health.
func deathUnit(id sim.EntityID, class int32, x, y int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, Class: class, HP: 100, MaxHP: 100}
}

// deathDraw is what the seam carries for one entity in the most recent push.
func deathDraw(t *testing.T, mw *mapWorld, id sim.EntityID) ui.MapEntity {
	t.Helper()
	for _, d := range mw.entityDraws() {
		if d.ID == uint32(id) {
			return d
		}
	}
	t.Fatalf("the push carries no entry for entity %d", id)
	return ui.MapEntity{}
}

// frameIndex is where a drawn frame sits in a class's own sheet, by pointer
// identity — the fixture gives every frame its own pointer, so this is a read
// of what was selected and not a second derivation of it.
func frameIndex(c *terrain.UnitClass, f *terrain.StaticFrame) int {
	for i, g := range c.Frames {
		if g == f {
			return i
		}
	}
	return -1
}

// A killed unit draws the corpse class's dying block from the tick it dies,
// one frame per two ticks, and then holds the last frame (AC-6, SC-6).
func TestAKilledUnitPlaysTheCorpseClassesFallAndThenHolds(t *testing.T) {
	sched := [][]sim.Command{{}, {{Kind: sim.KindKill, Entity: 1}}}
	mw := deathWorld(t, sched, deathUnit(1, 1, 3, 3))
	corpse := mw.units.Classes[2]

	// Alive on the tick-0 push and on the tick before the blow: its own art.
	if d := deathDraw(t, mw, 1); d.Art != mw.units.Classes[1] {
		t.Fatalf("a living unit is drawn as some other class")
	}
	mw.tick()
	if d := deathDraw(t, mw, 1); d.Art != mw.units.Classes[1] {
		t.Fatalf("a living unit is drawn as some other class after one tick")
	}

	// The blow lands on the second tick, and that push is elapsed 0.
	mw.tick()
	for elapsed := 0; elapsed <= 3*corpseSlot; elapsed++ {
		d := deathDraw(t, mw, 1)
		if d.Life != ui.LifeDead {
			t.Fatalf("elapsed %d: the seam reports life %d, want dead", elapsed, d.Life)
		}
		if d.Art != corpse {
			t.Fatalf("elapsed %d: drawn as class %p, want the corpse class %p", elapsed, d.Art, corpse)
		}
		want := deathFrameAt(north, elapsed)
		if got := frameIndex(corpse, d.Frame); got != want {
			t.Errorf("elapsed %d: frame %d, want %d", elapsed, got, want)
		}
		mw.tick()
	}

	// And it is still holding that frame much later.
	for i := 0; i < 40; i++ {
		mw.tick()
	}
	if got, want := frameIndex(corpse, deathDraw(t, mw, 1).Frame), deathFrameAt(north, 1000); got != want {
		t.Errorf("long after the fall the frame is %d, want the block's last, %d", got, want)
	}
}

// Downed and dead take the same path, and a downed unit's fall runs from the
// tick it was downed (AC-7, SC-6).
func TestDownedAndDeadBothFall(t *testing.T) {
	sched := [][]sim.Command{{
		{Kind: sim.KindDamage, Entity: 1, X: 100}, // exactly zero: downed
		{Kind: sim.KindKill, Entity: 2},           // -1: dead
	}}
	mw := deathWorld(t, sched, deathUnit(1, 1, 2, 2), deathUnit(2, 1, 5, 5))
	corpse := mw.units.Classes[2]

	mw.tick()
	for elapsed := 0; elapsed < 2*corpseSlot; elapsed++ {
		downed, dead := deathDraw(t, mw, 1), deathDraw(t, mw, 2)
		if downed.Life != ui.LifeDowned || dead.Life != ui.LifeDead {
			t.Fatalf("elapsed %d: lives are %d and %d, want downed and dead",
				elapsed, downed.Life, dead.Life)
		}
		want := deathFrameAt(north, elapsed)
		for _, c := range []struct {
			label string
			d     ui.MapEntity
		}{{"the downed unit", downed}, {"the dead unit", dead}} {
			if c.d.Art != corpse {
				t.Fatalf("elapsed %d: %s is not drawn as the corpse class", elapsed, c.label)
			}
			if got := frameIndex(corpse, c.d.Frame); got != want {
				t.Errorf("elapsed %d: %s draws frame %d, want %d", elapsed, c.label, got, want)
			}
		}
		mw.tick()
	}
}

// Never vanish: each of the three refusals keeps its unit drawn, with no
// death frame (AC-8, SC-6).
func TestNoUnitStopsBeingDrawnOnAccountOfDying(t *testing.T) {
	sched := [][]sim.Command{{
		{Kind: sim.KindKill, Entity: 3}, // corpse class holds no frames
		{Kind: sim.KindKill, Entity: 5}, // corpse class has no dying block
		{Kind: sim.KindKill, Entity: 7}, // names a class the bundle lacks
		{Kind: sim.KindKill, Entity: 8}, // no art at all: the square
	}}
	mw := deathWorld(t, sched,
		deathUnit(3, 3, 1, 1), deathUnit(5, 5, 2, 2),
		deathUnit(7, 7, 3, 3), deathUnit(8, 8, 4, 4))
	mw.tick()

	draws := mw.entityDraws()
	if len(draws) != 4 {
		t.Fatalf("the push holds %d entries for 4 entities — a dead unit was dropped", len(draws))
	}
	for _, tc := range []struct {
		label string
		id    sim.EntityID
		art   int32 // 0 means "no art: the square"
	}{
		{"a corpse class holding no frames", 3, 3},
		{"a corpse class with no dying block", 5, 5},
		{"a corpse the bundle does not hold", 7, 7},
		{"a unit with no art at all", 8, 0},
	} {
		d := deathDraw(t, mw, tc.id)
		if d.Life != ui.LifeDead {
			t.Fatalf("%s: the seam reports life %d, want dead", tc.label, d.Life)
		}
		if tc.art == 0 {
			if d.Frame != nil {
				t.Errorf("%s: carries a frame, want the square", tc.label)
			}
			continue
		}
		if d.Art != mw.units.Classes[tc.art] {
			t.Errorf("%s: drawn as some other class than its own", tc.label)
		}
		if d.Frame == nil {
			t.Errorf("%s: drawn with no frame at all — the unit vanished", tc.label)
		}
		if frameIndex(mw.units.Classes[tc.art], d.Frame) < 0 {
			t.Errorf("%s: the frame drawn is not in its own class's sheet", tc.label)
		}
	}
}

// A body keeps the direction it died facing, at every later tick (AC-9, SC-7).
func TestABodyKeepsTheDirectionItDiedFacing(t *testing.T) {
	// Walk west four cells, then kill. The world's own cells are what say
	// which way it went; the octant is this file's transcription of the spec's
	// facings, west being 2.
	const west = 2
	sched := make([][]sim.Command, 8)
	sched[0] = []sim.Command{{Kind: sim.KindMoveTo, Entity: 1, X: 2, Y: 4}}
	sched[5] = []sim.Command{{Kind: sim.KindKill, Entity: 1}}
	mw := deathWorld(t, sched, deathUnit(1, 1, 6, 4))
	corpse := mw.units.Classes[2]

	was := image.Point{X: 6, Y: 4}
	for i := 0; i < 5; i++ {
		mw.tick()
	}
	e, ok := mw.entity(1)
	if !ok || int(e.X) >= was.X {
		t.Fatalf("the entity did not walk west: it stands at %d, started at %d", e.X, was.X)
	}
	if !e.Alive() {
		t.Fatalf("the entity is already not alive before the blow")
	}

	mw.tick() // the blow
	for elapsed := 0; elapsed < 12; elapsed++ {
		d := deathDraw(t, mw, 1)
		if d.Step != (image.Point{}) {
			t.Fatalf("elapsed %d: a body carries a step %v — the simulation advanced it", elapsed, d.Step)
		}
		want := deathFrameAt(west, elapsed)
		if got := frameIndex(corpse, d.Frame); got != want {
			t.Errorf("elapsed %d: frame %d, want %d — the west slot of the dying block", elapsed, got, want)
		}
		mw.tick()
	}
}

// The snapshot stays a repeatable read: built twice with no advance between
// it answers identically, the death frames and their elapsed counts included
// (SC-6).
func TestASnapshotBuiltTwiceAnswersTheSameDeath(t *testing.T) {
	sched := [][]sim.Command{{{Kind: sim.KindKill, Entity: 1}}}
	mw := deathWorld(t, sched, deathUnit(1, 1, 3, 3), deathUnit(2, 1, 4, 4))

	for tick := 0; tick < 10; tick++ {
		first := mw.entityDraws()
		for i := 0; i < 3; i++ {
			again := mw.entityDraws()
			if len(again) != len(first) {
				t.Fatalf("tick %d: the snapshot changed length between builds", tick)
			}
			for j := range first {
				if !reflect.DeepEqual(again[j], first[j]) {
					t.Fatalf("tick %d entry %d: build %d answered %+v, the first answered %+v",
						tick, j, i+2, again[j], first[j])
				}
			}
		}
		mw.tick()
	}
}

// The clock is stamped once and the elapsed count is the scene clock's own
// difference — no per-entity offset, unlike the live selection's effective
// tick.
func TestTheDeathClockIsStampedOnceAndCarriesNoPerEntityOffset(t *testing.T) {
	// Two entities of very different id, killed on the same tick. If the
	// elapsed count carried the id the way the live selection does, their
	// falls would start at different frames.
	sched := [][]sim.Command{{}, {
		{Kind: sim.KindKill, Entity: 1},
		{Kind: sim.KindKill, Entity: 2},
	}}
	mw := deathWorld(t, sched, deathUnit(1, 1, 1, 1), deathUnit(2, 1, 6, 6))
	corpse := mw.units.Classes[2]

	mw.tick()
	mw.tick()
	if got := len(mw.died); got != 2 {
		t.Fatalf("the death clock holds %d entries for 2 dead entities", got)
	}
	stamp := mw.died[1]
	for _, id := range []sim.EntityID{1, 2} {
		if mw.died[id] != stamp {
			t.Errorf("entity %d was stamped at %d and entity 1 at %d — they died on one tick",
				id, mw.died[id], stamp)
		}
		if got, want := frameIndex(corpse, deathDraw(t, mw, id).Frame), deathFrameAt(north, 0); got != want {
			t.Errorf("entity %d starts its fall at frame %d, want %d", id, got, want)
		}
	}

	// The stamp does not move, however many ticks and however many builds.
	for i := 0; i < 20; i++ {
		mw.entityDraws()
		mw.tick()
		if mw.died[1] != stamp {
			t.Fatalf("the stamp for entity 1 moved to %d from %d", mw.died[1], stamp)
		}
	}
	if got := len(mw.died); got != 2 {
		t.Errorf("the death clock grew to %d entries — it is one per dead entity", got)
	}

	// And a living entity is never stamped at all.
	live := deathWorld(t, nil, deathUnit(1, 1, 3, 3))
	for i := 0; i < 5; i++ {
		live.tick()
	}
	if got := len(live.died); got != 0 {
		t.Errorf("the death clock holds %d entries in a world where nothing died", got)
	}
}

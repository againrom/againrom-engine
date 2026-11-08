package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// secondSparkWorld is a second-game caster (1) and target (2) with the Heal
// and Drain Life rows and the installed shape of their projectile rows:
// healing has 7 phases, Drain 9, and both home (R2-ASSET-101).
func secondSparkWorld(t *testing.T) *mapWorld {
	t.Helper()
	rules := []sim.SpellRule{{ID: 24}, {ID: 26}}
	sim.AssignSecondGameArms(rules)
	script, err := sim.NewROM2Script(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ents := []sim.Entity{
		{ID: 1, X: 3, Y: 3, HP: 100, MaxHP: 100, TokenSize: 1, Owner: 1},
		{ID: 2, X: 6, Y: 3, HP: 50, MaxHP: 100, TokenSize: 1, Owner: 1},
	}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, ents, script, rules)
	if err != nil {
		t.Fatal(err)
	}
	edition := base.GameROM2.Edition()
	frame := &terrain.StaticFrame{Width: 32, Height: 32, Pixels: make([]terrain.StaticPixel, 32*32)}
	art := &terrain.UnitClass{Width: 32, Height: 32, CenterX: 16, CenterY: 16, TileSize: 1,
		Frames: []*terrain.StaticFrame{frame}}
	sheets := emSheets(56, 60)
	sheets.Sheets[56].Phases, sheets.Sheets[60].Phases = 7, 9
	sheets.Homing = map[int]bool{56: true, 60: true}
	return &mapWorld{world: w, mission: &missionNotices{table: &mapload.Table{Edition: &edition}},
		units: &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{0: art}}, projectiles: sheets}
}

// secondSparkCast is unit 1's cast of spell at unit 2 as the tick wires it:
// the cast's release, then the mark rebuild with every mark of the tick.
func secondSparkCast(mw *mapWorld, spell uint16) {
	ev := sim.CastEvent{Caster: 1, Target: 2, Spell: spell, Weapon: true,
		FromX: 3, FromY: 3, ToX: 6, ToY: 3, TargetOwner: 1}
	mw.advanceEffectMarks(mw.observeCasts([]sim.CastEvent{ev})...)
}

// secondSparkTick is one later tick: the World step and the mark rebuild.
func secondSparkTick(mw *mapWorld) int {
	report := sim.StepReported(mw.world, nil)
	mw.advanceEffectMarks(report.ProjectileMarks...)
	return len(report.ProjectileMarks)
}

func secondSparkCount(mw *mapWorld, kind int) (uint16, int) {
	var count uint16
	n := 0
	for _, el := range mw.markElements {
		if el.Entity == 2 && el.Kind == kind && el.Transient {
			count, n = el.Count, n+1
		}
	}
	return count, n
}

// R2-ENGINE-691 gives a unit caster's cast object of 56 and 60 lifetime 1, and
// R2-ENGINE-700 has its one update set the target's mark once: no later tick
// sets it again and no record of the cast survives the next update.
func TestTheSecondGameHealAndDrainCastObjectUpdatesOnce(t *testing.T) {
	for _, tc := range []struct {
		spell uint16
		kind  int
	}{{24, 56}, {26, 60}} {
		t.Run(fmt.Sprint(tc.kind), func(t *testing.T) {
			mw := secondSparkWorld(t)
			secondSparkCast(mw, tc.spell)
			later := 0
			for range 12 {
				later += secondSparkTick(mw)
			}
			if later != 0 {
				t.Errorf("the cast object of %d set the mark again on %d later ticks; R2-ENGINE-691 gives it lifetime 1", tc.kind, later)
			}
			if n := len(mw.world.SavedProjectiles().Items); n != 0 {
				t.Errorf("%d cast records of %d survive 12 ticks", n, tc.kind)
			}
			if n := len(mw.world.ActiveEffects()); n != 0 {
				t.Errorf("the cast of %d built %d effects; R2-ENGINE-700 builds none", tc.kind, n)
			}
		})
	}
}

// R2-ENGINE-700: the mark is (ID<<16) OR 0x20, the generator runs with
// c = 32 .. 1 and the mark goes after 32 calls; a second cast inside that
// time overwrites the mark of that ID and starts it again at 32. A SAVE taken
// mid-run holds that mark and its LOAD continues it.
func TestTheSecondGameHealAndDrainSparksRunThirtyTwoSteps(t *testing.T) {
	for _, tc := range []struct {
		spell uint16
		kind  int
	}{{24, 56}, {26, 60}} {
		t.Run(fmt.Sprint(tc.kind), func(t *testing.T) {
			mw := secondSparkWorld(t)
			secondSparkCast(mw, tc.spell)
			for step := range 40 {
				count, n := secondSparkCount(mw, tc.kind)
				draws := len(mw.markDraws(2, mw.effectTileSize(2)))
				if step < 32 && (n != 1 || int(count) != 32-step || draws == 0) {
					t.Fatalf("step %d: %d marks of count %d drawing %d records; want one at c = %d", step, n, count, draws, 32-step)
				}
				if step >= 32 && (n != 0 || draws != 0) {
					t.Fatalf("step %d: the mark outlives its 32 calls (%d marks, %d records)", step, n, draws)
				}
				if step == 10 {
					rows := mw.captureTargetMarks()
					if len(rows) != 1 || rows[0].Element != uint32(tc.kind)<<16|uint32(count) {
						t.Fatalf("SAVE mid-run holds %+v; want one element %#x", rows, uint32(tc.kind)<<16|uint32(count))
					}
					if err := validateTargetMarks(rows); err != nil {
						t.Fatal(err)
					}
					before := mw.markDraws(2, mw.effectTileSize(2))
					mw.markElements = nil
					mw.restoreTargetMarks(rows)
					if after := mw.markDraws(2, mw.effectTileSize(2)); !reflect.DeepEqual(after, before) {
						t.Fatal("the LOADed mark draws other records")
					}
				}
				secondSparkTick(mw)
			}

			mw = secondSparkWorld(t)
			secondSparkCast(mw, tc.spell)
			for range 10 {
				secondSparkTick(mw)
			}
			secondSparkCast(mw, tc.spell)
			if count, n := secondSparkCount(mw, tc.kind); n != 1 || count != 32 {
				t.Fatalf("a second cast leaves %d marks of count %d; want the one mark overwritten at 32", n, count)
			}
		})
	}
}

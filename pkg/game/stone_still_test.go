package game

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func TestStoneDrawUsesFacingAndCanonicalPosition(t *testing.T) {
	for _, action := range []string{"standing", "midstep", "attack", "cast"} {
		t.Run(action, func(t *testing.T) {
			rule := sim.SpellRule{ID: 20, ManaCost: 1, School: 4, MaxRange: 8,
				TargetsUnit: true, SpellDuration: 60, EffectKind: sim.EffectAbsorption,
				EffectMode: sim.EffectDuration, EffectMagnitude: 5}
			ents := emWorld(t, 20, rule).Entities()
			e := &ents[1]
			e.Facing, e.DesiredFacing = 64, 64
			e.Owner = 2
			if action == "midstep" {
				e.Transit, e.TransitTotal = 20, 32
				e.HasTarget, e.TargetX, e.TargetY = true, 20, 10
				e.Stride = sim.NativeStride{Present: true, FromX: 11, FromY: 10, ToX: 12, ToY: 10,
					Rate: 8, StepX: 8, Direction: 2}
			}
			if action == "attack" {
				e.HasAttackTarget, e.AttackTarget, e.Reach = true, 1, 4
				e.AttackPhase, e.AttackCountdown, e.AttackCharge = sim.AttackCharging, 50, 50
			}
			w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 64, Height: 64}, sim.ModeCanonical,
				nil, ents, nil, []sim.SpellRule{rule})
			if err != nil {
				t.Fatal(err)
			}
			emCast(t, w, 20)
			frames := make([]*terrain.StaticFrame, 100)
			for i := range frames {
				frames[i] = &terrain.StaticFrame{Width: 1, Height: 1}
			}
			art := &terrain.UnitClass{Frames: frames, Anim: terrain.UnitAnim{S: 16, D: 8,
				MoveBase: 16, MoveSlot: 2, MoveOK: true, MoveTrack: []int{0, 1},
				AttackBase: 32, AttackSlot: 2, AttackOK: true, AttackTrack: []int{0, 1},
				TailBase: 48, IdleSlot: 2, IdleOK: true, IdleTrack: []int{0, 1}}}
			frozen, _ := w.Entity(2)
			previous := image.Pt(int(frozen.X), int(frozen.Y))
			if action == "midstep" {
				previous = image.Pt(11, 10)
			}
			mw := &mapWorld{world: w, units: &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{0: art}},
				prev: map[sim.EntityID]image.Point{2: previous}, walk: map[sim.EntityID]walkClock{},
				swing: map[sim.EntityID]int{}, phase: map[sim.EntityID]sim.AttackPhase{}, died: map[sim.EntityID]int{}}
			if action == "cast" {
				mw.castRun = map[sim.EntityID]castRun{2: {left: 50, span: 50}}
			}
			if action == "attack" && !frozen.HasAttackTarget {
				t.Fatal("attacking control lost its admitted attack before the curse")
			}
			wantFacing := ((int(frozen.DrawnFacing()) >> 4) + 8) & 15
			wantCell, wantX, wantY := image.Pt(int(frozen.X), int(frozen.Y)), uint8(128), uint8(128)
			if frozen.Stride.Present {
				x := frozen.Stride.FromX*256 + 128 + int32(frozen.Stride.StepX)*int32(frozen.TransitTotal-frozen.Transit)
				y := frozen.Stride.FromY*256 + 128 + int32(frozen.Stride.StepY)*int32(frozen.TransitTotal-frozen.Transit)
				wantCell, wantX, wantY = image.Pt(int(x>>8), int(y>>8)), uint8(x), uint8(y)
			}
			for tick := 0; tick < 40; tick++ {
				for _, draw := range mw.entityDraws() {
					if draw.ID != 2 {
						continue
					}
					if !draw.Stone || draw.Frame != frames[wantFacing] || draw.Mirror || !draw.FinePosition ||
						draw.Cell != wantCell || draw.FineX != wantX || draw.FineY != wantY {
						t.Fatalf("tick %d: frame=%d mirror=%t cell=%v fine=%t:%d,%d; want facing=%d cell=%v fine=%d,%d",
							tick, stoneFrameIndex(draw.Frame, frames), draw.Mirror, draw.Cell, draw.FinePosition,
							draw.FineX, draw.FineY, wantFacing, wantCell, wantX, wantY)
					}
				}
				mw.recordCells()
				sim.Step(w, nil)
				mw.scene++
				mw.advanceOdometers()
				mw.advanceSwings()
				mw.advanceCastRuns()
				if mw.walk[2] != (walkClock{}) || mw.swing[2] != 0 || action == "cast" && mw.castRun[2].left != 50 {
					t.Fatalf("held presentation clocks advanced: walk=%+v swing=%d cast=%+v", mw.walk[2], mw.swing[2], mw.castRun[2])
				}
			}
		})
	}
}

func stoneFrameIndex(frame *terrain.StaticFrame, frames []*terrain.StaticFrame) int {
	for i, f := range frames {
		if f == frame {
			return i
		}
	}
	return -1
}

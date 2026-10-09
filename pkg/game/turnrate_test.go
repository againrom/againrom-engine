package game

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func TestHalfTurnDrawsEveryStandingSixteenth(t *testing.T) {
	for _, layout := range []struct {
		name   string
		stored int
	}{{"plain", 16}, {"mirrored", 9}} {
		t.Run(layout.name, func(t *testing.T) {
			actor := sim.Entity{ID: 1, X: 4, Y: 4, Class: 1, HP: 200, MaxHP: 200,
				Speed: 16, RotationSpeed: 16}
			anim := terrain.UnitAnim{S: layout.stored, D: 8, MoveBase: 16, MoveSlot: 1,
				MoveTrack: []int{0}, MoveOK: true, TailBase: 24, IdleSlot: 1,
				IdleTrack: []int{0}, IdleOK: true}
			mw := swingWorld(t, anim, actor)
			mw.enqueue(1, 4, 5)
			for tick, direction := range []int{9, 10, 11, 12, 13, 14, 15, 0} {
				mw.tick()
				body := gaEntity(t, mw.world, actor.ID)
				if body.X != actor.X || body.Y != actor.Y || !body.Turning() {
					t.Fatalf("tick %d: body (%d,%d), turning %v; want standing at (%d,%d)",
						tick+1, body.X, body.Y, body.Turning(), actor.X, actor.Y)
				}
				wantFrame, wantMirror := direction, false
				if layout.stored == 9 && direction > 8 {
					wantFrame, wantMirror = 16-direction, true
				}
				before := mw.world.Hash()
				draw := swingDraw(t, mw, actor.ID)
				if draw.Frame != draw.Art.Frames[wantFrame] || draw.Mirror != wantMirror {
					t.Errorf("tick %d: drawn standing frame %d mirrored %v; want %d mirrored %v",
						tick+1, turnFrameIndex(t, draw.Art, draw.Frame), draw.Mirror, wantFrame, wantMirror)
				}
				if mw.world.Hash() != before {
					t.Fatal("drawing changed the simulation hash")
				}
			}
		})
	}
}

func TestCompletedTurnKeepsTheClientDirectionInLiveArt(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		facing                     uint8
		stored, directions         int
		standing, idle             int
		standingMirror, idleMirror bool
	}{
		{"plain_north_bend", 16, 16, 8, 9, 28, false, false},
		{"mirrored_north_bend", 16, 9, 5, 7, 28, true, false},
		{"plain_east_bend", 48, 16, 8, 11, 29, false, false},
		{"mirrored_east_bend", 48, 9, 5, 5, 27, true, true},
		{"plain_south_bend", 144, 16, 8, 1, 24, false, false},
		{"mirrored_south_bend", 144, 9, 5, 1, 24, false, false},
	} {
		for _, idle := range []bool{false, true} {
			phase := "standing"
			if idle {
				phase = "idle"
			}
			t.Run(tc.name+"/"+phase, func(t *testing.T) {
				actor := sim.Entity{ID: 1, X: 4, Y: 4, Class: 1, HP: 200, MaxHP: 200,
					Speed: 16, RotationSpeed: 16, Facing: tc.facing, DesiredFacing: tc.facing,
					TurnState: sim.TurnState{Present: true, DrawComplete: true, Drawn: tc.facing,
						DrawTarget: tc.facing >> 4}}
				anim := terrain.UnitAnim{S: tc.stored, D: tc.directions, TailBase: 24,
					IdleSlot: 1, IdleTrack: []int{0}, IdleOK: idle}
				mw := swingWorld(t, anim, actor)
				assertFrame := func(phase string, want int, mirrored bool) {
					t.Helper()
					hash := mw.world.Hash()
					draw := swingDraw(t, mw, actor.ID)
					frame := turnFrameIndex(t, draw.Art, draw.Frame)
					if frame != want || draw.Mirror != mirrored {
						t.Errorf("%s: submitted frame/mirror %d/%v, want %d/%v", phase,
							frame, draw.Mirror, want, mirrored)
					}
					if mw.world.Hash() != hash {
						t.Fatal("drawing changed the simulation hash")
					}
				}
				assertFrame("last turn frame", tc.standing, tc.standingMirror)
				sim.Step(mw.world, nil)
				body := gaEntity(t, mw.world, actor.ID)
				if body.DrawingTurn() || body.DrawnFacing() != tc.facing || body.X != actor.X || body.Y != actor.Y {
					t.Fatalf("completed turn state: drawn %d turning %v cell (%d,%d)",
						body.DrawnFacing(), body.DrawingTurn(), body.X, body.Y)
				}
				want, mirrored := tc.standing, tc.standingMirror
				if idle {
					want, mirrored = tc.idle, tc.idleMirror
				}
				assertFrame("after turn completion", want, mirrored)
			})
		}
	}
}

func turnFrameIndex(t *testing.T, art *terrain.UnitClass, frame *terrain.StaticFrame) int {
	t.Helper()
	if art == nil {
		t.Fatal("submitted actor has no class art")
	}
	for i, candidate := range art.Frames {
		if candidate == frame {
			return i
		}
	}
	t.Fatal("submitted frame does not belong to the submitted class")
	return -1
}

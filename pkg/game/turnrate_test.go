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

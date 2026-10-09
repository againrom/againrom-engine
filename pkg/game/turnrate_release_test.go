package game

import "testing"

func TestReleaseReniestaHalfTurnDrawsEveryStandingSixteenth(t *testing.T) {
	k := openKargallas(t)
	mw := k.mw
	actor := releaseEntity(t, mw, k.renie)
	if actor.Facing != 0 || actor.RotationSpeed != 16 {
		t.Fatalf("installed Reniesta facing/rate %d/%d, want 0/16", actor.Facing, actor.RotationSpeed)
	}
	mw.enqueue(uint32(actor.ID), int(actor.X), int(actor.Y+1))
	for tick, direction := range []int{9, 10, 11, 12, 13, 14, 15, 0} {
		mw.tick()
		mw.mission.open = false
		body := releaseEntity(t, mw, actor.ID)
		draw := swingDraw(t, mw, actor.ID)
		if draw.Art == nil {
			t.Fatal("installed actor has no submitted class art")
		}
		frame := -1
		for index, source := range draw.Art.TierFrames(mw.tiers[body.ID]) {
			if mw.ownerFrame(draw.Art, source, body.Owner) == draw.Frame {
				frame = index
				break
			}
		}
		if frame < 0 {
			t.Fatal("submitted frame does not belong to the actor's owner-coloured sheet")
		}
		wantFrame, wantMirror := direction, false
		if draw.Art.Anim.S == 9 && direction > 8 {
			wantFrame, wantMirror = 16-direction, true
		}
		t.Logf("tick %d: facing %d drawn %d submitted frame %d mirror %v standing %v", tick+1,
			body.Facing, body.DrawnFacing(), frame, draw.Mirror, body.X == actor.X && body.Y == actor.Y)
		if body.X != actor.X || body.Y != actor.Y || !body.Turning() {
			t.Fatalf("tick %d: actor moved to (%d,%d) or ended its turn early", tick+1, body.X, body.Y)
		}
		if frame != wantFrame || draw.Mirror != wantMirror {
			t.Errorf("tick %d: submitted frame %d mirror %v, want standing frame %d mirror %v", tick+1,
				frame, draw.Mirror, wantFrame, wantMirror)
		}
	}
}

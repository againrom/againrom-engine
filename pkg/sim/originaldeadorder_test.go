package sim

import "testing"

// A body restored from a SAV whose actor-state word still names patrol, or
// another order only a living actor holds, keeps the state the World gave it;
// the World stays one the byte form round-trips.
func TestOriginalActionDropsALivingOnlyStateOnABody(t *testing.T) {
	for _, state := range []uint8{actorStatePatrol, actorStateAcquire, actorStateRetreat, actorStateDefend, actorStateFollow} {
		w := mustWorld(t, 7, Bounds{Width: 8, Height: 8}, []Entity{
			{ID: 1, X: 2, Y: 2, HP: -35, MaxHP: 768, Decay: DecayBones},
			{ID: 2, X: 5, Y: 5, HP: 10, MaxHP: 10},
		})
		if err := w.ImportOriginalActorActions([]OriginalActorAction{
			{Entity: 1, ActorState: state, PostX: 3, PostY: 3},
			{Entity: 2, ActorState: actorStatePatrol, PostX: 5, PostY: 5},
		}); err != nil {
			t.Fatalf("state %#x: %v", state, err)
		}
		body, living := w.entities[0], w.entities[1]
		if body.ActorState != actorStateGuard || body.PatrolHeadX|body.PatrolHeadY|body.PatrolTailX|body.PatrolTailY != 0 || body.HasEscortTarget {
			t.Fatalf("state %#x: body restored with state %#x, ring (%d,%d)-(%d,%d), escort %v", state, body.ActorState,
				body.PatrolHeadX, body.PatrolHeadY, body.PatrolTailX, body.PatrolTailY, body.HasEscortTarget)
		}
		if living.ActorState != actorStatePatrol {
			t.Fatalf("living actor restored with state %#x, want patrol", living.ActorState)
		}
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var back World
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatalf("state %#x: byte form refused: %v", state, err)
		}
	}
}

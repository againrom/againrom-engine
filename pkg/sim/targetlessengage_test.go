package sim

import "testing"

func targetlessEngageWorld(t *testing.T, second bool) (*World, int, *SavedActorOrder) {
	t.Helper()
	w := guardRouteWorld(t, second, 0)
	i := indexOfEntity(w.entities, guardRouteFoe)
	w.entities[i].ActorState = actorStateEngage
	o := w.ensureSavedOrder(i)
	o.State = uint32(actorStateEngage)
	o.authorNative()
	return w, i, o
}

func TestSecondGameTargetlessEngageTakesAcquireAtItsCell(t *testing.T) {
	w, i, o := targetlessEngageWorld(t, true)
	w.savedEngage(i, o)
	e := &w.entities[i]
	if e.ActorState != actorStateAcquire || o.State != uint32(actorStateAcquire) {
		t.Fatalf("state %d order %d, want acquire", e.ActorState, o.State)
	}
	if e.PostX != e.X || e.PostY != e.Y {
		t.Fatalf("post %d,%d, want its cell %d,%d", e.PostX, e.PostY, e.X, e.Y)
	}
}

func TestSecondGameEngageKeepsAWalkInProgress(t *testing.T) {
	w, i, o := targetlessEngageWorld(t, true)
	e := &w.entities[i]
	e.TargetX, e.TargetY, e.HasTarget = 30, 30, true
	w.savedEngage(i, o)
	if e.ActorState != actorStateEngage || o.State != uint32(actorStateEngage) || !e.HasTarget {
		t.Fatalf("walking member changed: state %d order %d target %v", e.ActorState, o.State, e.HasTarget)
	}
}

func TestFirstGameTargetlessEngageStaysAsItWas(t *testing.T) {
	w, i, o := targetlessEngageWorld(t, false)
	w.savedEngage(i, o)
	if e := &w.entities[i]; e.ActorState != actorStateEngage || o.State != uint32(actorStateEngage) {
		t.Fatalf("first game changed: state %d order %d", e.ActorState, o.State)
	}
}

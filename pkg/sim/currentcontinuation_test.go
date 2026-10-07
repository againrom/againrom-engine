package sim

import "testing"

func TestCurrentContinuationRestoreIsAtomic(t *testing.T) {
	w := scrollFixtureWorld(t, 2, 1)
	bindOperationsPack(t, w, 0)
	before := w.Hash()
	item := w.carried[0][0].ObjectID
	if item == 0 {
		t.Fatal("fixture has no bound item")
	}
	for _, fault := range []string{"policy", "clock", "population", "action", "identity floor"} {
		t.Run(fault, func(t *testing.T) {
			policy := w.CurrentPolicy()
			policy.TickHigh, policy.ObjectCarrier = 1, false
			values := map[EntityID]ActorValues{}
			for _, e := range w.entities {
				values[e.ID] = e.Values()
			}
			actions := actionCopy(t, w.Actions())
			switch fault {
			case "policy":
				policy.Mode = 255
			case "clock":
				policy.ClockKnown = true
			case "population":
				delete(values, w.entities[0].ID)
			case "action":
				actions.Actors[0].AttackTargetKind = 255
			case "identity floor":
				invalid := entityIDLimit + 1
				policy.EntityIDFloor = &invalid
			}
			if err := w.RestoreCurrentContinuation(&policy, values, actions, nil); err == nil {
				t.Fatal("invalid continuation accepted")
			}
			if w.Hash() != before || w.savedObjects == nil || w.carried[0][0].ObjectID != item {
				t.Fatal("failed transaction changed current world or shared item storage")
			}
		})
	}
}

func TestCurrentContinuationKeepsNativeClockAndCarrierAbsence(t *testing.T) {
	w := scrollFixtureWorld(t, 2, 1)
	bindOperationsPack(t, w, 0)
	policy := w.CurrentPolicy()
	policy.TickHigh, policy.ObjectCarrier = 1, false
	values := map[EntityID]ActorValues{}
	for _, e := range w.entities {
		values[e.ID] = e.Values()
	}
	changed := values[w.entities[0].ID]
	changed.SeeInvisible = 7
	values[w.entities[0].ID] = changed
	if err := w.RestoreCurrentContinuation(&policy, values, actionCopy(t, w.Actions()), nil); err != nil {
		t.Fatal(err)
	}
	if w.tick != 1<<32 || w.hasSessionClock || w.savedObjects != nil || w.carried[0][0].ObjectID != 0 || w.entities[0].SeeInvisible != changed.SeeInvisible {
		t.Fatal("current policy was replaced by imported clock, registry or actor values")
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatal(err)
	}
}

func TestCurrentContinuationKeepsResolvedExternalIdentityReservations(t *testing.T) {
	w := scrollFixtureWorld(t, 2, 1)
	p := w.CurrentPolicy()
	w.ReserveEntityIDs([]EntityID{50})
	if err := w.RestoreCurrentContinuation(&p, nil, w.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	if next, ok := w.NextEntityID(); !ok || next != 51 {
		t.Fatal("policy erased a separately resolved absent actor binding", next, ok)
	}
}

func TestCurrentContinuationKeepsStoredFloorBelowLivingSummon(t *testing.T) {
	w := hlGhostWorld(t, 51, []SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}},
		hlGhostTemplate(), effectMage(1, 1, 1, 1<<25), spEnt(2, 2, 1))
	w.entities[1].HP, w.entities[1].Decay = -10, DecayBones
	spRunCast(w, Cast(1, 2, 25))
	if indexOfEntity(w.entities, 3) < 0 || w.entityIDFloor != 3 {
		t.Fatal("fixture needs a live summon above the stored reservation floor")
	}
	policy := w.CurrentPolicy()
	if policy.EntityIDFloor == nil || *policy.EntityIDFloor != 3 {
		t.Fatal("capture replaced the stored floor with the next available identity")
	}
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	ids := map[EntityID]EntityID{}
	for _, id := range cold.ActorIdentityReferences() {
		ids[id] = id
	}
	if err := cold.RestoreActorIdentities(ids, policy.EntityIDFloor); err != nil {
		t.Fatal(err)
	}
	if err := cold.RestoreCurrentContinuation(&policy, nil, w.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	if cold.entityIDFloor != 3 || cold.Hash() != w.Hash() {
		t.Fatal("current LOAD changed the stored floor or current World")
	}
	if next, ok := cold.NextEntityID(); !ok || next != 4 {
		t.Fatal("live summon no longer protects its own identity", next, ok)
	}
	i := indexOfEntity(cold.entities, 3)
	cold.entities[i].HP, cold.entities[i].Decay = decayGoneHP-1, DecayBones
	Step(&cold, nil)
	if indexOfEntity(cold.entities, 3) >= 0 || cold.entityIDFloor != 4 {
		t.Fatal("removal did not retain the departed summon's identity")
	}
	if next, ok := cold.NextEntityID(); !ok || next != 4 {
		t.Fatal("removed summon identity became reusable", next, ok)
	}
}

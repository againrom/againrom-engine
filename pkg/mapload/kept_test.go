package mapload_test

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A member kept from the last mission starts the next one with the action-run
// end he carried (SAV-1355); a member built in the town starts with an
// unknown run end, which the first tick sets.
func TestAKeptMemberStartsWithItsCarriedActionRunEnd(t *testing.T) {
	t.Parallel()

	kept := joinParty()
	kept[0].Kept = &mapload.KeptActor{Done: 1, Deadline: 577}
	w, st, err := mapload.StartMission(joinMap(), joinTable(), mapload.DifficultyNormal, kept)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if got := joinEntity(t, w, st.IDs[0]).ActionClock; got != (sim.ActionClock{Known: true, End: 577}) {
		t.Fatalf("kept member action clock = %+v, want known end 577", got)
	}

	w, st, err = mapload.StartMission(joinMap(), joinTable(), mapload.DifficultyNormal, joinParty())
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if got := joinEntity(t, w, st.IDs[0]).ActionClock; got.Known {
		t.Fatalf("town-built member action clock = %+v, want unknown", got)
	}
}

// The carrier is copied whole with the member, so a later write to one copy's
// lists does not reach the other.
func TestKeptActorClonesItsLists(t *testing.T) {
	t.Parallel()

	in := mapload.PartyMember{ID: "a", Kept: &mapload.KeptActor{Deadline: 9, StaticRoute: []uint16{1}, OrderRing: []uint16{2}}}
	out := mapload.CloneParty([]mapload.PartyMember{in})[0]
	if out.Kept == in.Kept || out.Kept.Deadline != 9 {
		t.Fatalf("clone kept = %+v", out.Kept)
	}
	out.Kept.StaticRoute[0], out.Kept.OrderRing[0] = 7, 7
	if in.Kept.StaticRoute[0] != 1 || in.Kept.OrderRing[0] != 2 {
		t.Fatal("clone shares its lists with the source")
	}
}

// The party a mission carries out holds no kept-actor state; the town fills
// it from the World for each member the mission kept.
func TestCarryPartyDropsTheKeptActor(t *testing.T) {
	t.Parallel()

	in := []mapload.PartyMember{{ID: "a", Kept: &mapload.KeptActor{Deadline: 9}}}
	out := mapload.CarryParty(in, nil, nil)
	if len(out) != 1 || out[0].Kept != nil || in[0].Kept == nil {
		t.Fatalf("carried kept = %+v, source kept = %+v", out[0].Kept, in[0].Kept)
	}
}

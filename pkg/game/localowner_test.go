package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestTheFrontEndIsToldWhichSlotThePlayerHolds is 0094 AC-14.
//
// The value pushed here is read by two rules a package away, and until 0094 it
// was a zero meaning "none established". A party member then owned slot 0 too, so
// the damage numeral's `e.Owner != localOwner` came out "mine" for the player by
// the accident of both sides being the same nothing. The party now owns slot 1,
// and a pushed zero would flip the drift of every figure over the player's own
// units — so this is asserted as an EQUALITY between the two, not as the
// literal 1. A story that reseats the player has one number to move and this
// test says so.
func TestTheFrontEndIsToldWhichSlotThePlayerHolds(t *testing.T) {
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)

	if got := v.LocalOwner(); got != 0 {
		t.Fatalf("a fresh viewer already holds slot %d — then the push below proves nothing", got)
	}
	w, st, err := mapload.StartMission(m, nil, mapload.DifficultyNormal, MissionParty(nil, nil, nil))
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	_ = newMapWorld(w, nil, nil, v)

	if len(st.IDs) == 0 {
		t.Fatal("the start placed no party — the equality below would hold vacuously")
	}
	var member sim.Entity
	for _, e := range w.Entities() {
		if e.ID == st.IDs[0] {
			member = e
		}
	}
	if got := v.LocalOwner(); got != member.Owner {
		t.Errorf("the front end was told the local participant holds slot %d while the "+
			"party stands on slot %d — the numeral drift of the player's own units "+
			"turns on these two agreeing", got, member.Owner)
	}
	if v.LocalOwner() != sim.SelfSlot {
		t.Errorf("the pushed slot is %d, want the roster slot the party takes (%d)",
			v.LocalOwner(), uint32(sim.SelfSlot))
	}
}

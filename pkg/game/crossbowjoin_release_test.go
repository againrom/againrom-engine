package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The mission 70 crossbowmen join the party when a party unit comes within 6 of
// (38,108). A joined actor stands in a new group at group order 0 and acts on
// its own guard state (AI-CMD-033, PARTY-JOIN-025), so a monk that walks up to
// one is engaged without an order.
func TestReleaseMission70JoinedCrossbowmenEngageAMonkWithoutAnOrder(t *testing.T) {
	f := openCampaignMission(t, 70)
	mw := f.live
	w := mw.world
	refs := mapload.ScriptUnits(mw.mission.state.Map, mw.mission.party)
	var bows []sim.EntityID
	for _, u := range []uint16{47, 48, 49, 50, 129, 130, 131, 132} {
		id, ok := refs[u]
		if !ok {
			t.Fatalf("script unit %d is not in mission 70", u)
		}
		if e, _ := w.Entity(id); e.Owner != 6 || e.Reach < 5 {
			t.Fatalf("script unit %d is owner %d reach %d, want an owner 6 ranged unit", u, e.Owner, e.Reach)
		}
		bows = append(bows, id)
	}
	hero := mapload.PartyEntity(mw.mission.state.Map, 0)
	if err := w.HeadlessPlace(hero, 38, 108); err != nil {
		t.Fatal(err)
	}
	missionTicks(mw, 40)
	for _, id := range bows {
		if e, _ := w.Entity(id); e.Owner != 1 {
			t.Fatalf("entity %d is still owner %d after the party reached the join radius", id, e.Owner)
		}
	}

	monk := refs[135]
	if m, _ := w.Entity(monk); m.Owner != 7 {
		t.Fatalf("script unit 135 is owner %d, want the monks' owner 7", m.Owner)
	}
	if err := w.HeadlessPlace(monk, 43, 83); err != nil {
		t.Fatal(err)
	}
	foe := map[sim.EntityID]bool{}
	for _, e := range w.Entities() {
		if e.Owner == 7 {
			foe[e.ID] = true
		}
	}
	const within = 64
	first := -1
	for i := 0; i < within && first < 0; i++ {
		missionTicks(mw, 1)
		for _, id := range bows {
			if e, _ := w.Entity(id); e.HasAttackTarget && foe[e.AttackTarget] {
				first = i
			}
		}
	}
	if first < 0 {
		t.Fatalf("no joined crossbowman engaged a monk within %d ticks of its arrival three cells away", within)
	}
	t.Logf("first crossbowman target %d ticks after the monk arrived", first)
}

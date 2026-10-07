package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseMission20SarindarKeepsHisRowClassArt(t *testing.T) {
	f := releaseFront(t)
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(20)(); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	if f.live == nil || f.live.mission == nil || f.live.mission.state == nil {
		t.Fatal("mission 20 opened without a live mission state")
	}
	// Mission entity 48 is Sarindar in both shipped language roots. The RU
	// display name is a legacy-encoded byte string, so the stable map entity is
	// the cross-install discriminator rather than localized text.
	const id sim.EntityID = 48
	member, found := f.live.mission.state.Start.Roster[id]
	if !found || !member.Mage {
		t.Fatalf("mission 20 entity %d = %+v, want Sarindar's mage roster row", id, member)
	}
	slots, ok := f.live.world.Equipped(id)
	if !ok {
		t.Fatalf("Sarindar entity %d has no live equipment array", id)
	}
	if slots[0] != 0 {
		t.Fatalf("Sarindar live weapon slot = %#04x, want empty discriminator", slots[0])
	}
	e, _ := f.live.entity(id)
	if e.TypeID != 23 {
		t.Fatalf("Sarindar type id %d, want the unarmed mage class 23", e.TypeID)
	}
	want := f.Units.Classes[23]
	if want == nil || f.live.art[id] != want {
		t.Fatalf("Sarindar tick-zero art = %p, want class 23 (%p)", f.live.art[id], want)
	}
	if member.Class != 23 || member.Body != "" {
		t.Fatalf("Sarindar roster class %d body %q, want class 23 and no derived body", member.Class, member.Body)
	}
}

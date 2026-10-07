package mapload

import (
	"testing"

	"againrom/pkg/data"
)

// OwnParty sets Rider from the member's class.
func TestOwnPartySetsTheRiderTermFromClass(t *testing.T) {
	in := []PartyMember{{ID: "a", Class: 21}, {ID: "b", Class: 19}, {ID: "c", Class: 33}, {ID: "d"}}
	out := OwnParty(in)
	for i, want := range []bool{true, true, false, false} {
		if got := out[i].Profile.Rider; got != want {
			t.Errorf("member %d (class %d): Rider %t, want %t", i, in[i].Class, got, want)
		}
	}
	if in[0].Profile.Rider {
		t.Error("OwnParty changed its input")
	}
	rider := data.Hero{Body: 30, Reaction: 30}
	out[0].Hero, out[3].Hero = rider, rider
	if got, _, _ := partySpawn(out[0], data.Loadout{}); got.Speed != 28 {
		t.Errorf("rider spawn speed %d, want 28", got.Speed)
	}
	if got, _, _ := partySpawn(out[3], data.Loadout{}); got.Speed != 18 {
		t.Errorf("foot spawn speed %d, want 18", got.Speed)
	}
}

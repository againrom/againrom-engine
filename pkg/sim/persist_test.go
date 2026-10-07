package sim

import (
	"reflect"
	"testing"
)

// The band and the boundary: which actors cross into the next mission.

// TestTheBandAdmitsThePersonArmAndRefusesTheShippedCreatureIDs is 0159 AC-1's
// sim half.
//
// The shipped creature rows declare 34 class ids and none of them is inside the
// band. All 34 are listed here rather than sampled, so a band moved by one at
// either end is caught by the id that sits next to it: 0x40 is the first value
// above the band and 79 and 80 are two of the shipped ids just past it.
func TestTheBandAdmitsThePersonArmAndRefusesTheShippedCreatureIDs(t *testing.T) {
	t.Parallel()

	if !InPersistBand(HumanTypeID) {
		t.Fatalf("the value the person arm writes, %#x, is outside the band", HumanTypeID)
	}
	for _, v := range []int32{0x21, 0x22, 0x23, 0x24, 0x3f} {
		if !InPersistBand(v) {
			t.Errorf("%#x is outside the band and should be inside it", v)
		}
	}
	shipped := []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 19, 21, 23, 24, 26, 27,
		64, 65, 66, 68, 69, 70, 71, 72, 73, 74, 75, 76, 79, 80}
	for _, v := range shipped {
		if InPersistBand(v) {
			t.Errorf("shipped creature id %d is inside the band and should not be", v)
		}
	}
	// The unresolved placement's own value, and the first value above the band.
	for _, v := range []int32{0, 0x20, 0x40, 0x41} {
		if InPersistBand(v) {
			t.Errorf("%#x is inside the band and should not be", v)
		}
	}
}

// TestABandTypeIDPaysNoDeathGold is 0159 AC-2's second clause.
//
// The gold roll is gated at strictly above 0x40 and the band ends below it, so
// giving a person a band type id cannot make him drop gold. The gold chance is
// set to 100 — certain, had the gate opened — and both treasure bounds are set
// high, so a single zero here is the gate and not an empty purse.
func TestABandTypeIDPaysNoDeathGold(t *testing.T) {
	t.Parallel()

	w := scriptWorld(t, nil, []Entity{
		{ID: 1, HP: 1, MaxHP: 1, TypeID: HumanTypeID,
			GoldChance: 100, TreasureMin: 500, TreasureMax: 500},
		{ID: 2, HP: 1, MaxHP: 1, TypeID: PersistHigh - 1,
			GoldChance: 100, TreasureMin: 500, TreasureMax: 500},
		{ID: 3, HP: 1, MaxHP: 1, TypeID: PersistHigh + 1,
			GoldChance: 100, TreasureMin: 500, TreasureMax: 500},
	})
	for i, want := range map[int]bool{0: false, 1: false, 2: true} {
		got := w.deathGold(i) > 0
		if got != want {
			t.Errorf("entity %d (type id %#x) paid gold = %v, want %v",
				i+1, w.entities[i].TypeID, got, want)
		}
	}
}

// boundaryWorld holds one actor of each kind the boundary has to separate: a
// live band actor of the human participant, a dead one, a live out-of-band one,
// and a live band actor of somebody else. Only the first crosses.
func boundaryWorld(t *testing.T, order []Entity) *World {
	t.Helper()
	return scriptWorld(t, nil, order)
}

func boundaryFixture() []Entity {
	return []Entity{
		{ID: 1, HP: 5, MaxHP: 5, Owner: SelfSlot, TypeID: HumanTypeID},
		{ID: 2, HP: 0, MaxHP: 5, Owner: SelfSlot, TypeID: HumanTypeID},
		{ID: 3, HP: 5, MaxHP: 5, Owner: SelfSlot, TypeID: 0x40},
		{ID: 4, HP: 5, MaxHP: 5, Owner: SelfSlot + 1, TypeID: HumanTypeID},
		{ID: 5, HP: 5, MaxHP: 5, Owner: SelfSlot, TypeID: 0},
	}
}

// TestTheBoundaryKeepsOnlyTheLiveBandActorsOfTheParticipant is 0159 AC-6.
func TestTheBoundaryKeepsOnlyTheLiveBandActorsOfTheParticipant(t *testing.T) {
	t.Parallel()

	w := boundaryWorld(t, boundaryFixture())
	want := []EntityID{1}
	if got := w.BoundarySurvivors(SelfSlot); !reflect.DeepEqual(got, want) {
		t.Errorf("the boundary keeps %v, want %v", got, want)
	}
	// The other slot's own band actor crosses under ITS OWN slot, which is what
	// says the test is per owner rather than per side.
	if got := w.BoundarySurvivors(SelfSlot + 1); !reflect.DeepEqual(got, []EntityID{4}) {
		t.Errorf("slot %d keeps %v, want [4]", SelfSlot+1, got)
	}
}

func TestTheBoundaryAnswerDoesNotDependOnStorageOrder(t *testing.T) {
	t.Parallel()

	forward := boundaryFixture()
	// Two live band actors, so the ORDER of the answer is measurable and not
	// only its membership.
	forward = append(forward, Entity{ID: 6, HP: 5, MaxHP: 5, Owner: SelfSlot, TypeID: HumanTypeID})
	reversed := make([]Entity, len(forward))
	for i := range forward {
		reversed[i] = forward[len(forward)-1-i]
	}
	want := []EntityID{1, 6}
	for name, ents := range map[string][]Entity{"forward": forward, "reversed": reversed} {
		w := boundaryWorld(t, ents)
		if got := w.BoundarySurvivors(SelfSlot); !reflect.DeepEqual(got, want) {
			t.Errorf("%s storage order keeps %v, want %v", name, got, want)
		}
	}
}

// TestAWorldThatIsNotThereKeepsNobody guards the nil arm CarryRoster relies on.
func TestAWorldThatIsNotThereKeepsNobody(t *testing.T) {
	t.Parallel()

	var w *World
	if got := w.BoundarySurvivors(SelfSlot); got != nil {
		t.Errorf("a nil world keeps %v, want nothing", got)
	}
}
